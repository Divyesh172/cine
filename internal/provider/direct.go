package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
)

func init() { Register("direct", newDirect) }

// direct is a generic HLS/direct-stream provider — the "Direct" slot and the
// Luffy-style path. It calls a USER-CONFIGURED scraper endpoint that returns JSON
// containing .m3u8/http(s) source URLs and hands them straight to mpv (no
// peerflix): instant, adaptive-bitrate playback. Nothing is hardcoded.
type direct struct {
	endpoint string
	referer  string
	label    string
}

func newDirect(opts map[string]interface{}) (Provider, error) {
	ep, _ := opts["endpoint"].(string)
	ep = strings.TrimSpace(ep)
	if ep == "" {
		return nil, fmt.Errorf("direct: 'endpoint' option is required")
	}
	ref, _ := opts["referer"].(string)
	label, _ := opts["label"].(string)
	if strings.TrimSpace(label) == "" {
		label = "Direct"
	}
	return &direct{endpoint: ep, referer: strings.TrimSpace(ref), label: label}, nil
}

func (d *direct) Name() string { return d.label }

func (d *direct) Search(ctx context.Context, q Query) ([]Stream, error) {
	data, err := directCurl(ctx, d.buildURL(q), d.referer)
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("direct: empty response from endpoint")
	}
	if trimmed[0] != '{' && trimmed[0] != '[' {
		return nil, fmt.Errorf("direct: expected JSON, got: %s", directSnippet(trimmed))
	}
	var root interface{}
	if err := json.Unmarshal(trimmed, &root); err != nil {
		return nil, fmt.Errorf("direct: decode: %w (body: %s)", err, directSnippet(trimmed))
	}
	streams := directExtract(root, d.Name(), q)
	if len(streams) == 0 {
		return nil, fmt.Errorf("direct: no playable http(s) sources in response")
	}
	return streams, nil
}

// buildURL fills placeholders in the configured endpoint template.
// Supported: {imdb} {title} {year} {type} {season} {episode}
func (d *direct) buildURL(q Query) string {
	mediaType := "movie"
	if q.Type == Series {
		mediaType = "series"
	}
	return strings.NewReplacer(
		"{imdb}", q.IMDbID,
		"{title}", url.QueryEscape(q.Title),
		"{year}", strconv.Itoa(q.Year),
		"{type}", mediaType,
		"{season}", strconv.Itoa(q.Season),
		"{episode}", strconv.Itoa(q.Episode),
	).Replace(d.endpoint)
}

// directCurl shells out to curl for the same reasons as the Torrentio provider
// (WSL IPv6 black-hole + Cloudflare TLS fingerprinting). An optional Referer is
// sent when configured, since some stream hosts require it.
func directCurl(ctx context.Context, u, referer string) ([]byte, error) {
	curlPath, err := exec.LookPath("curl")
	if err != nil {
		return nil, fmt.Errorf("direct: curl not found in PATH: %w", err)
	}
	args := []string{"-4", "-sS", "--http1.1", "--compressed", "--max-time", "30", "-H", "Accept: application/json"}
	if referer != "" {
		args = append(args, "-H", "Referer: "+referer)
	}
	args = append(args, u)
	cmd := exec.CommandContext(ctx, curlPath, args...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("direct: curl: %v: %s", err, strings.TrimSpace(errb.String()))
	}
	return out.Bytes(), nil
}

// directExtract walks arbitrary JSON and collects every object carrying an
// http(s) URL field, so it tolerates many backend shapes without a fixed schema.
func directExtract(root interface{}, providerName string, q Query) []Stream {
	var out []Stream
	seen := map[string]bool{}
	var walk func(v interface{})
	walk = func(v interface{}) {
		switch t := v.(type) {
		case []interface{}:
			for _, e := range t {
				walk(e)
			}
		case map[string]interface{}:
			u := directFirst(t, "url", "file", "link", "src", "stream", "playlist", "hls")
			if strings.HasPrefix(strings.ToLower(u), "http") {
				if !seen[u] {
					seen[u] = true
					title := directFirst(t, "title", "name", "release", "label", "server")
					if title == "" {
						title = q.Title
					}
					out = append(out, Stream{
						Provider: providerName,
						Title:    title,
						Quality:  directFirst(t, "quality", "resolution", "label", "name"),
						URL:      u,
					})
				}
				return
			}
			for _, e := range t {
				walk(e)
			}
		}
	}
	walk(root)
	return out
}

// directFirst returns the first non-empty string among the given keys (case-insensitive).
func directFirst(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	for mk, v := range m {
		lk := strings.ToLower(mk)
		for _, k := range keys {
			if lk == strings.ToLower(k) {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					return strings.TrimSpace(s)
				}
			}
		}
	}
	return ""
}

func directSnippet(b []byte) string {
	s := strings.TrimSpace(string(b))
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 220 {
		s = s[:220] + "..."
	}
	return s
}

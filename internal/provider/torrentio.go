package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func init() { Register("torrentio", newTorrentio) }

type torrentio struct {
	baseURL string // user-supplied; nothing hardcoded
}

func newTorrentio(opts map[string]interface{}) (Provider, error) {
	base, _ := opts["endpoint"].(string)
	if base == "" {
		return nil, fmt.Errorf("torrentio: 'endpoint' option is required")
	}
	return &torrentio{baseURL: strings.TrimRight(base, "/")}, nil
}

func (t *torrentio) Name() string { return "Torrentio" }

func (t *torrentio) Search(ctx context.Context, q Query) ([]Stream, error) {
	if q.IMDbID == "" {
		return nil, fmt.Errorf("IMDb id required")
	}
	var path string
	if q.Type == Series {
		s, e := q.Season, q.Episode
		if s == 0 {
			s = 1
		}
		if e == 0 {
			e = 1
		}
		path = fmt.Sprintf("/stream/series/%s:%d:%d.json", q.IMDbID, s, e)
	} else {
		path = fmt.Sprintf("/stream/movie/%s.json", q.IMDbID)
	}

	data, err := tioCurl(ctx, t.baseURL+path)
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, fmt.Errorf("torrentio: expected JSON, got: %s", tioSnippet(trimmed))
	}

	var body struct {
		Streams []struct {
			Name     string `json:"name"`
			Title    string `json:"title"`
			InfoHash string `json:"infoHash"`
			URL      string `json:"url"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(trimmed, &body); err != nil {
		return nil, fmt.Errorf("torrentio: decode: %w (body: %s)", err, tioSnippet(trimmed))
	}

	out := make([]Stream, 0, len(body.Streams))
	for _, s := range body.Streams {
		title := tioOneLine(s.Title)
		u := s.URL
		if u == "" && s.InfoHash != "" {
			u = tioMagnet(s.InfoHash, title)
		}
		if u == "" {
			continue
		}
		out = append(out, Stream{
			Provider: t.Name(),
			Title:    title,
			URL:      u,
			Seeders:  tioSeeders(s.Title),
			Size:     tioSize(s.Title),
		})
	}
	return out, nil
}

// tioCurl shells out to curl. Go's own TLS fingerprint gets blocked/tarpitted by
// Cloudflare; curl's does not. Keep curl's default User-Agent — claiming to be a
// browser while sending curl's TLS handshake trips bot detection.
func tioCurl(ctx context.Context, u string) ([]byte, error) {
	curlPath, err := exec.LookPath("curl")
	if err != nil {
		return nil, fmt.Errorf("torrentio: curl not found in PATH: %w", err)
	}
	cmd := exec.CommandContext(ctx, curlPath,
		"-4", "-sS", "--http1.1", "--compressed",
		"--max-time", "30",
		"-H", "Accept: application/json",
		u,
	)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("torrentio: curl: %v: %s", err, strings.TrimSpace(errb.String()))
	}
	return out.Bytes(), nil
}

var tioTrackers = []string{
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://open.tracker.cl:1337/announce",
	"udp://tracker.openbittorrent.com:6969/announce",
	"udp://exodus.desync.com:6969/announce",
	"udp://tracker.torrent.eu.org:451/announce",
}

func tioMagnet(hash, name string) string {
	var b strings.Builder
	b.WriteString("magnet:?xt=urn:btih:" + hash)
	if name != "" {
		b.WriteString("&dn=" + url.QueryEscape(name))
	}
	for _, tr := range tioTrackers {
		b.WriteString("&tr=" + url.QueryEscape(tr))
	}
	return b.String()
}

// \x{1F464} = seeders emoji, \x{1F4BE} = size emoji (escaped to keep source ASCII).
var tioSeedersRe = regexp.MustCompile(`\x{1F464}\s*(\d+)`)
var tioSizeRe = regexp.MustCompile(`\x{1F4BE}\s*([\d.]+\s*[KMGT]i?B)`)

func tioSeeders(s string) int {
	if m := tioSeedersRe.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}

func tioSize(s string) string {
	if m := tioSizeRe.FindStringSubmatch(s); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func tioOneLine(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "\n", " / ")
}

func tioSnippet(b []byte) string {
	s := strings.TrimSpace(string(b))
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 220 {
		s = s[:220] + "..."
	}
	return s
}

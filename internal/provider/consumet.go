package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func init() { Register("consumet", newConsumet) }

// consumet talks to a SELF-HOSTED Consumet API (github.com/consumet/api.consumet.org),
// using its movie/TV provider (flixhq by default). It runs the full
// search -> info -> watch flow and returns direct .m3u8 (HLS) sources that mpv
// plays instantly (no peerflix). Self-host it so no third-party backend can
// vanish on you:
//
//	git clone https://github.com/consumet/api.consumet.org && npm i && npm start
type consumet struct {
	baseURL  string // your instance, e.g. http://localhost:3000
	provider string // consumet movie provider; default "flixhq"
	server   string // streaming server; default "vidcloud"
	label    string
}

func newConsumet(opts map[string]interface{}) (Provider, error) {
	base, _ := opts["endpoint"].(string)
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return nil, fmt.Errorf("consumet: 'endpoint' option is required (your self-hosted base URL, e.g. http://localhost:3000)")
	}
	prov, _ := opts["provider"].(string)
	if strings.TrimSpace(prov) == "" {
		prov = "flixhq"
	}
	server, _ := opts["server"].(string)
	if strings.TrimSpace(server) == "" {
		server = "vidcloud"
	}
	label, _ := opts["label"].(string)
	if strings.TrimSpace(label) == "" {
		label = "Consumet"
	}
	return &consumet{
		baseURL:  base,
		provider: strings.TrimSpace(prov),
		server:   strings.TrimSpace(server),
		label:    label,
	}, nil
}

func (c *consumet) Name() string { return c.label }

func (c *consumet) Search(ctx context.Context, q Query) ([]Stream, error) {
	mediaID, err := c.findMedia(ctx, q)
	if err != nil {
		return nil, err
	}
	episodeID, err := c.findEpisode(ctx, mediaID, q)
	if err != nil {
		return nil, err
	}
	return c.fetchSources(ctx, mediaID, episodeID, q)
}

// findMedia searches by title and returns the best-matching media id.
func (c *consumet) findMedia(ctx context.Context, q Query) (string, error) {
	u := fmt.Sprintf("%s/movies/%s/%s", c.baseURL, c.provider, url.PathEscape(q.Title))
	root, err := c.getJSON(ctx, u)
	if err != nil {
		return "", err
	}
	results := conSlice(conMap(root)["results"])
	if len(results) == 0 {
		return "", fmt.Errorf("consumet: no results for %q", q.Title)
	}
	wantSeries := q.Type == Series
	year := ""
	if q.Year > 0 {
		year = strconv.Itoa(q.Year)
	}
	var firstTypeMatch string
	for _, r := range results {
		m := conMap(r)
		id := conStr(m, "id")
		if id == "" {
			continue
		}
		typ := strings.ToLower(conStr(m, "type"))
		isSeries := strings.Contains(typ, "tv") || strings.Contains(typ, "series")
		if isSeries != wantSeries {
			continue
		}
		if firstTypeMatch == "" {
			firstTypeMatch = id
		}
		titleMatch := strings.EqualFold(conStr(m, "title"), strings.TrimSpace(q.Title))
		yearMatch := year == "" || strings.Contains(conStr(m, "releaseDate"), year)
		if titleMatch && yearMatch {
			return id, nil
		}
	}
	if firstTypeMatch != "" {
		return firstTypeMatch, nil
	}
	return conStr(conMap(results[0]), "id"), nil
}

// findEpisode returns the episode id to stream: the sole entry for a movie, or
// the matching season/episode for a series.
func (c *consumet) findEpisode(ctx context.Context, mediaID string, q Query) (string, error) {
	u := fmt.Sprintf("%s/movies/%s/info?id=%s", c.baseURL, c.provider, url.QueryEscape(mediaID))
	root, err := c.getJSON(ctx, u)
	if err != nil {
		return "", err
	}
	episodes := conSlice(conMap(root)["episodes"])
	if len(episodes) == 0 {
		return "", fmt.Errorf("consumet: no episodes/streams for media %q", mediaID)
	}
	if q.Type != Series {
		return conStr(conMap(episodes[0]), "id"), nil
	}
	for _, e := range episodes {
		m := conMap(e)
		num, _ := conNum(m, "number")
		sea, hasSea := conNum(m, "season")
		if num == q.Episode && (!hasSea || q.Season == 0 || sea == q.Season) {
			return conStr(m, "id"), nil
		}
	}
	return "", fmt.Errorf("consumet: S%02dE%02d not found for %q", q.Season, q.Episode, q.Title)
}

// fetchSources hits /watch and returns HLS streams plus any HTTP headers the
// player must send (flixhq hosts usually require a Referer).
func (c *consumet) fetchSources(ctx context.Context, mediaID, episodeID string, q Query) ([]Stream, error) {
	u := fmt.Sprintf("%s/movies/%s/watch?episodeId=%s&mediaId=%s&server=%s",
		c.baseURL, c.provider,
		url.QueryEscape(episodeID), url.QueryEscape(mediaID), url.QueryEscape(c.server))
	root, err := c.getJSON(ctx, u)
	if err != nil {
		return nil, err
	}
	rootMap := conMap(root)

	headers := map[string]string{}
	for k, v := range conMap(rootMap["headers"]) {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			headers[k] = s
		}
	}
	if len(headers) == 0 {
		headers = nil
	}

	sources := conSlice(rootMap["sources"])
	if len(sources) == 0 {
		return nil, fmt.Errorf("consumet: no playable sources (try a different server= option)")
	}
	var out []Stream
	seen := map[string]bool{}
	for _, s := range sources {
		m := conMap(s)
		link := conStr(m, "url")
		if link == "" || !strings.HasPrefix(strings.ToLower(link), "http") || seen[link] {
			continue
		}
		seen[link] = true
		quality := conStr(m, "quality")
		title := q.Title + " (HLS)"
		if quality != "" {
			title = fmt.Sprintf("%s (%s HLS)", q.Title, quality)
		}
		out = append(out, Stream{
			Provider: c.Name(),
			Title:    title,
			Quality:  quality,
			URL:      link,
			Headers:  headers,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("consumet: sources present but none were http(s)")
	}
	return out, nil
}

// getJSON fetches a URL (via the shared curl helper) and decodes JSON.
func (c *consumet) getJSON(ctx context.Context, u string) (interface{}, error) {
	data, err := directCurl(ctx, u, "")
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("consumet: empty response (is your instance up at %q?)", c.baseURL)
	}
	if trimmed[0] != '{' && trimmed[0] != '[' {
		return nil, fmt.Errorf("consumet: expected JSON, got: %s", directSnippet(trimmed))
	}
	var root interface{}
	if err := json.Unmarshal(trimmed, &root); err != nil {
		return nil, fmt.Errorf("consumet: decode: %w (body: %s)", err, directSnippet(trimmed))
	}
	return root, nil
}

func conMap(v interface{}) map[string]interface{} {
	m, _ := v.(map[string]interface{})
	return m
}

func conSlice(v interface{}) []interface{} {
	s, _ := v.([]interface{})
	return s
}

func conStr(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if s, ok := m[key].(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func conNum(m map[string]interface{}, key string) (int, bool) {
	if m == nil {
		return 0, false
	}
	switch n := m[key].(type) {
	case float64:
		return int(n), true
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(n))
		return i, err == nil
	}
	return 0, false
}

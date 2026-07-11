package provider

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func init() { Register("jackett", newJackett) }

// jackett queries a SELF-HOSTED Jackett instance via its Torznab API. Jackett
// proxies dozens of torrent indexers (public + private) behind one normalized
// XML feed — the same backend Sonarr/Radarr use. Nothing is hardcoded: the
// endpoint, API key, and indexer all come from user config.
//
// Run it once (Docker):
//   docker run -d -p 9117:9117 -v jackett-config:/config linuxserver/jackett
// then open http://localhost:9117, add indexers, and copy the API key.
type jackett struct {
	baseURL    string // e.g. http://localhost:9117
	apiKey     string
	indexer    string // "all" (default) or a specific indexer id
	categories string // optional Torznab cat filter, e.g. "2000,5000"
	label      string
}

func newJackett(opts map[string]interface{}) (Provider, error) {
	base, _ := opts["endpoint"].(string)
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return nil, fmt.Errorf("jackett: 'endpoint' option is required (e.g. http://localhost:9117)")
	}
	key, _ := opts["api_key"].(string)
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("jackett: 'api_key' option is required (copy it from the Jackett dashboard)")
	}
	indexer, _ := opts["indexer"].(string)
	if strings.TrimSpace(indexer) == "" {
		indexer = "all"
	}
	cats, _ := opts["categories"].(string)
	label, _ := opts["label"].(string)
	if strings.TrimSpace(label) == "" {
		label = "Jackett"
	}
	return &jackett{
		baseURL:    base,
		apiKey:     key,
		indexer:    strings.TrimSpace(indexer),
		categories: strings.TrimSpace(cats),
		label:      strings.TrimSpace(label),
	}, nil
}

func (j *jackett) Name() string { return j.label }

func (j *jackett) Search(ctx context.Context, q Query) ([]Stream, error) {
	// Reuse the shared curl helper. Jackett is localhost so there are no
	// Cloudflare/IPv6 concerns, but staying consistent with the other providers.
	// Torznab always returns XML regardless of the Accept header.
	data, err := directCurl(ctx, j.buildURL(q), "")
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("jackett: empty response (is Jackett running at %q?)", j.baseURL)
	}
	if trimmed[0] != '<' {
		return nil, fmt.Errorf("jackett: expected XML, got: %s", directSnippet(trimmed))
	}
	var resp torznabResp
	if err := xml.Unmarshal(trimmed, &resp); err != nil {
		return nil, fmt.Errorf("jackett: decode: %w (body: %s)", err, directSnippet(trimmed))
	}

	out := make([]Stream, 0, len(resp.Channel.Items))
	for _, it := range resp.Channel.Items {
		u := it.magnet()
		if u == "" {
			u = it.Enclosure.URL // Jackett .torrent proxy URL
		}
		if u == "" {
			u = it.Link
		}
		if u == "" {
			continue
		}
		size := jackettSize(it.Size)
		if size == "" {
			if b, perr := strconv.ParseInt(it.attr("size"), 10, 64); perr == nil {
				size = jackettSize(b)
			}
		}
		out = append(out, Stream{
			Provider: j.Name(),
			Title:    tioOneLine(it.Title),
			URL:      u,
			Seeders:  it.seeders(),
			Size:     size,
		})
	}
	return out, nil
}

// buildURL constructs the Torznab query. A generic t=search with a well-formed q
// works across every indexer type (movies, TV — and, later, books/music), unlike
// the imdb-keyed t=movie/t=tvsearch modes that not all indexers implement.
func (j *jackett) buildURL(q Query) string {
	params := url.Values{}
	params.Set("apikey", j.apiKey)
	params.Set("t", "search")
	params.Set("q", j.queryString(q))
	if j.categories != "" {
		params.Set("cat", j.categories)
	}
	return fmt.Sprintf("%s/api/v2.0/indexers/%s/results/torznab/api?%s",
		j.baseURL, url.PathEscape(j.indexer), params.Encode())
}

func (j *jackett) queryString(q Query) string {
	if q.Type == Series {
		s, e := q.Season, q.Episode
		if s == 0 {
			s = 1
		}
		if e == 0 {
			e = 1
		}
		return fmt.Sprintf("%s S%02dE%02d", q.Title, s, e)
	}
	if q.Year > 0 {
		return fmt.Sprintf("%s %d", q.Title, q.Year)
	}
	return q.Title
}

// --- Torznab XML shapes ---

type torznabResp struct {
	Channel struct {
		Items []torznabItem `xml:"item"`
	} `xml:"channel"`
}

type torznabItem struct {
	Title     string `xml:"title"`
	Link      string `xml:"link"`
	Size      int64  `xml:"size"`
	Enclosure struct {
		URL  string `xml:"url,attr"`
		Type string `xml:"type,attr"`
	} `xml:"enclosure"`
	// <torznab:attr name="..." value="..."/> — matched by local name "attr".
	Attrs []struct {
		Name  string `xml:"name,attr"`
		Value string `xml:"value,attr"`
	} `xml:"attr"`
}

func (it torznabItem) attr(name string) string {
	for _, a := range it.Attrs {
		if strings.EqualFold(a.Name, name) {
			return a.Value
		}
	}
	return ""
}

func (it torznabItem) seeders() int {
	n, _ := strconv.Atoi(strings.TrimSpace(it.attr("seeders")))
	return n
}

// magnet returns a magnet: URI if the item exposes one (torznab magneturl attr,
// or a magnet directly in enclosure/link); otherwise "".
func (it torznabItem) magnet() string {
	if m := it.attr("magneturl"); strings.HasPrefix(m, "magnet:") {
		return m
	}
	if strings.HasPrefix(it.Enclosure.URL, "magnet:") {
		return it.Enclosure.URL
	}
	if strings.HasPrefix(it.Link, "magnet:") {
		return it.Link
	}
	return ""
}

// jackettSize renders a byte count as a compact human string ("1.68 GB").
func jackettSize(b int64) string {
	if b <= 0 {
		return ""
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

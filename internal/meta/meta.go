package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const base = "https://v3-cinemeta.strem.io"

type Client struct{ http *http.Client }

func New() *Client { return &Client{http: &http.Client{Timeout: 10 * time.Second}} }

type Result struct {
	Title  string
	Year   string
	Type   string // "movie" or "series"
	IMDbID string // already "tt..." — no second lookup needed
	Poster string
}

// Search queries Cinemeta's movie and series catalogs. No key required, and the
// returned IMDb id can go straight into a provider Query (Torrentio, etc.).
func (c *Client) Search(ctx context.Context, query string) ([]Result, error) {
	var out []Result
	for _, typ := range []string{"movie", "series"} {
		u := fmt.Sprintf("%s/catalog/%s/top/search=%s.json", base, typ, url.PathEscape(query))
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		var body struct {
			Metas []struct {
				ID          string `json:"id"` // "tt0903747"
				Name        string `json:"name"`
				ReleaseInfo string `json:"releaseInfo"` // "2008-2013"
				Poster      string `json:"poster"`
				Type        string `json:"type"`
			} `json:"metas"`
		}
		err = json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		for _, m := range body.Metas {
			out = append(out, Result{
				Title: m.Name, Year: m.ReleaseInfo, Type: m.Type,
				IMDbID: m.ID, Poster: m.Poster,
			})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no results for %q", query)
	}
	return out, nil
}

// Resolve searches Cinemeta and returns the best match for a title + type
// ("movie" or "series"). Prefers an exact (case-insensitive) title match,
// otherwise the first result of the requested type, otherwise the first result.
func (c *Client) Resolve(ctx context.Context, title, wantType string) (Result, error) {
	results, err := c.Search(ctx, title)
	if err != nil {
		return Result{}, err
	}
	var firstOfType *Result
	for i := range results {
		if results[i].Type != wantType {
			continue
		}
		if firstOfType == nil {
			firstOfType = &results[i]
		}
		if strings.EqualFold(strings.TrimSpace(results[i].Title), strings.TrimSpace(title)) {
			return results[i], nil
		}
	}
	if firstOfType != nil {
		return *firstOfType, nil
	}
	return results[0], nil
}

// Episodes fetches the season/episode list for a series via the meta endpoint:
// GET /meta/series/{imdbID}.json → videos[]. Use this to fill Query.Season/Episode.

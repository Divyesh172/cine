package provider

import "context"

// MediaType distinguishes movies from series.
type MediaType string

const (
	Movie  MediaType = "movie"
	Series MediaType = "series"
)

// Query is what a provider is asked to resolve into playable streams.
type Query struct {
	Title   string
	Year    int
	Type    MediaType
	IMDbID  string // preferred key when available
	Season  int    // series only
	Episode int    // series only
}

// Stream is a single playable result.
type Stream struct {
	Provider string
	Title    string // human label, e.g. "1080p WEB-DL"
	Quality  string // "1080p", "2160p", ...
	URL      string // magnet: or http(s):
	Seeders  int
	Size     string
	Headers  map[string]string // optional HTTP headers the player must send (e.g. Referer for HLS)
}

// Provider is the single interface every plugin implements.
// Adding a new source = implementing this + registering it. Core never changes.
type Provider interface {
	Name() string
	Search(ctx context.Context, q Query) ([]Stream, error)
}

// Factory builds a Provider from its user config options.
type Factory func(opts map[string]interface{}) (Provider, error)

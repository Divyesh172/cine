package provider

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/Divyesh172/cine/internal/quality"
)

var factories = map[string]Factory{}

// Register wires a provider type to its factory. Call from init() in each plugin.
func Register(typ string, f Factory) { factories[typ] = f }

type Registry struct{ providers []Provider }

// Build instantiates only the enabled providers from config.
func Build(specs []struct {
	Type    string
	Enabled bool
	Options map[string]interface{}
}) (*Registry, error) {
	r := &Registry{}
	for _, s := range specs {
		if !s.Enabled {
			continue
		}
		f, ok := factories[s.Type]
		if !ok {
			return nil, fmt.Errorf("unknown provider type %q", s.Type)
		}
		p, err := f(s.Options)
		if err != nil {
			return nil, fmt.Errorf("init provider %q: %w", s.Type, err)
		}
		r.providers = append(r.providers, p)
	}
	return r, nil
}

// SearchAll queries every provider in parallel. One failure never kills the run.
func (r *Registry) SearchAll(ctx context.Context, q Query) []Stream {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	var (
		wg  sync.WaitGroup
		mu  sync.Mutex
		out []Stream
	)
	for _, p := range r.providers {
		wg.Add(1)
		go func(p Provider) {
			defer wg.Done()
			streams, err := p.Search(ctx, q)
			if err != nil {
				fmt.Printf("  ✗ %s: %v\n", p.Name(), err)
				return
			}
			fmt.Printf("  ✓ %s (%d)\n", p.Name(), len(streams))
			mu.Lock()
			out = append(out, streams...)
			mu.Unlock()
		}(p)
	}
	wg.Wait()

	// Enrich each stream with parsed quality, then rank best-first.
	type ranked struct {
		stream Stream
		score  int
	}
	rankedStreams := make([]ranked, len(out))
	for i, s := range out {
		info := quality.Parse(s.Title)
		if s.Quality == "" {
			s.Quality = info.Label()
		}
		rankedStreams[i] = ranked{stream: s, score: info.Score()}
	}
	sort.SliceStable(rankedStreams, func(i, j int) bool {
		if rankedStreams[i].score != rankedStreams[j].score {
			return rankedStreams[i].score > rankedStreams[j].score
		}
		return rankedStreams[i].stream.Seeders > rankedStreams[j].stream.Seeders
	})
	final := make([]Stream, len(rankedStreams))
	for i := range rankedStreams {
		final[i] = rankedStreams[i].stream
	}
	// Safety cap: even after ranking, a broad indexer query can return hundreds
	// of hits. Keep the best-scoring slice so the selector and peerflix bridge
	// stay responsive.
	const maxResults = 100
	if len(final) > maxResults {
		final = final[:maxResults]
	}
	return final
}

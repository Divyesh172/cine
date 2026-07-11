package resolver

import (
	"fmt"
	"strings"
	"time"
)

// Resolver turns a magnet/torrent link into a plain HTTP URL that mpv can play,
// returning a cleanup func to stop the bridge process once playback ends.
type Resolver interface {
	Resolve(magnet string) (url string, cleanup func(), err error)
}

// IsMagnet reports whether a stream URL must be resolved before mpv can play it.
func IsMagnet(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.HasPrefix(s, "magnet:") || strings.HasSuffix(s, ".torrent")
}

// New builds a resolver from config values. "peerflix" (default) and
// "webtorrent" both use the peerflix bridge; "none" disables magnet playback.
func New(typ, bin string, args []string, timeoutSecs int) (Resolver, error) {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "", "peerflix", "webtorrent":
		return NewPeerflix(bin, args, time.Duration(timeoutSecs)*time.Second), nil
	case "none":
		return nil, fmt.Errorf("this stream is a magnet but resolver.type is \"none\" — set resolver.type: peerflix in ~/.config/cine/config.yaml (then: npm install -g peerflix)")
	default:
		return nil, fmt.Errorf("unknown resolver type %q (use \"peerflix\" or \"none\")", typ)
	}
}

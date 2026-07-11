package provider

import (
	"strings"
	"testing"
)

func TestTioSeeders(t *testing.T) {
	// \U0001F464 is the seeders emoji used by Torrentio titles.
	title := "Movie 1080p\n\U0001F464 42 \U0001F4BE 1.4 GB"
	if got := tioSeeders(title); got != 42 {
		t.Errorf("tioSeeders = %d, want 42", got)
	}
}

func TestTioSize(t *testing.T) {
	title := "Movie 1080p \U0001F464 42 \U0001F4BE 1.4 GB"
	if got := tioSize(title); got != "1.4 GB" {
		t.Errorf("tioSize = %q, want '1.4 GB'", got)
	}
}

func TestTioMagnet(t *testing.T) {
	m := tioMagnet("abc123", "Some Movie")
	if !strings.HasPrefix(m, "magnet:?xt=urn:btih:abc123") {
		t.Errorf("magnet prefix wrong: %s", m)
	}
	if !strings.Contains(m, "&tr=") {
		t.Error("expected trackers injected into magnet")
	}
	if !strings.Contains(m, "dn=Some+Movie") {
		t.Errorf("expected display name in magnet, got %s", m)
	}
}

func TestTioOneLine(t *testing.T) {
	if got := tioOneLine("a\nb"); got != "a / b" {
		t.Errorf("tioOneLine = %q, want 'a / b'", got)
	}
}

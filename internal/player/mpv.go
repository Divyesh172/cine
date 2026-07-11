package player

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Play launches the configured player with the given stream URL.
// magnet: links require an mpv build with a torrent hook (e.g. peerflix/webtorrent),
// otherwise pass an http(s) stream from a debrid/direct provider.
func Play(player string, extraArgs []string, url, title string, headers map[string]string) error {
	if player == "" {
		player = "mpv"
	}
	var args []string
	// User-supplied flags first (e.g. --vo=x11 for headless/WSL environments).
	args = append(args, extraArgs...)
	// For direct media files, skip mpv's youtube-dl hook so a bad URL surfaces a
	// clear ffmpeg/HTTP error instead of a confusing ytdl fallback.
	if isDirectMedia(url) {
		args = append(args, "--ytdl=no")
	}
	// Forward required HTTP headers (many HLS hosts 403 without a Referer).
	if len(headers) > 0 {
		var fields []string
		for k, v := range headers {
			if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
				continue
			}
			fields = append(fields, k+": "+v)
		}
		if len(fields) > 0 {
			args = append(args, "--http-header-fields="+strings.Join(fields, ","))
		}
	}
	if title != "" {
		args = append(args, "--force-media-title="+title)
	}
	args = append(args, url)
	fmt.Fprintf(os.Stderr, "→ %s %s\n", player, strings.Join(args, " "))
	cmd := exec.Command(player, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("launch %s: %w", player, err)
	}
	return nil
}

func isDirectMedia(u string) bool {
	l := strings.ToLower(u)
	// HLS/DASH manifests often carry query strings, so match anywhere.
	if strings.Contains(l, ".m3u8") || strings.Contains(l, ".mpd") {
		return true
	}
	// Strip any query/fragment before checking plain file extensions.
	if i := strings.IndexAny(l, "?#"); i >= 0 {
		l = l[:i]
	}
	for _, ext := range []string{".mp4", ".mkv", ".mov", ".webm", ".avi", ".m4v", ".ts", ".flv"} {
		if strings.HasSuffix(l, ext) {
			return true
		}
	}
	return false
}

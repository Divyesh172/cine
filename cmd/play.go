package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/Divyesh172/cine/internal/config"
	"github.com/Divyesh172/cine/internal/meta"
	"github.com/Divyesh172/cine/internal/player"
	"github.com/Divyesh172/cine/internal/provider"
	"github.com/Divyesh172/cine/internal/resolver"
	"github.com/Divyesh172/cine/internal/store"
	"github.com/Divyesh172/cine/internal/ui"
	"github.com/spf13/cobra"
)

// seRe matches a trailing SxxExx token, e.g. "S02E05" (case-insensitive).
var seRe = regexp.MustCompile(`(?i)\s*s(\d{1,2})e(\d{1,2})\s*$`)

func parsePlayArgs(args []string) (title string, season, episode int, isSeries bool) {
	joined := strings.TrimSpace(strings.Join(args, " "))
	if m := seRe.FindStringSubmatch(joined); m != nil {
		season, _ = strconv.Atoi(m[1])
		episode, _ = strconv.Atoi(m[2])
		title = strings.TrimSpace(joined[:len(joined)-len(m[0])])
		return title, season, episode, true
	}
	return joined, 0, 0, false
}

// parseYear pulls a leading 4-digit year out of Cinemeta's ReleaseInfo string
// (e.g. "2022" or "2008-2013"); returns 0 when absent. Feeding the year into the
// provider Query lets Jackett narrow a broad title search ("The Conversation" ->
// "The Conversation 2022") so it stops returning hundreds of loose matches.
func parseYear(s string) int {
	s = strings.TrimSpace(s)
	if len(s) >= 4 {
		if y, err := strconv.Atoi(s[:4]); err == nil {
			return y
		}
	}
	return 0
}

var downloadMode bool

func init() {
	playCmd.Flags().BoolVarP(&downloadMode, "download", "d", false,
		"print and open the selected magnet in your torrent client instead of streaming")
}

var playCmd = &cobra.Command{
	Use:   "play [title] [SxxExx]",
	Short: "Resolve streams via providers and play in mpv",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		title, season, episode, isSeries := parsePlayArgs(args)
		if title == "" {
			return fmt.Errorf("please provide a title")
		}
		wantType := "movie"
		if isSeries {
			wantType = "series"
		}

		// 1. Metadata search, then let the user pick the right title (Bubble Tea).
		results, err := meta.New().Search(context.Background(), title)
		if err != nil {
			return err
		}
		var candidates []meta.Result
		for _, r := range results {
			if r.Type == wantType {
				candidates = append(candidates, r)
			}
		}
		if len(candidates) == 0 {
			candidates = results
		}
		titleLabels := make([]string, len(candidates))
		for i, r := range candidates {
			titleLabels[i] = fmt.Sprintf("%s (%s) [%s]", r.Title, r.Year, r.IMDbID)
		}
		ti, err := ui.Select("Select a title:", titleLabels)
		if err != nil {
			return err
		}
		hit := candidates[ti]
		fmt.Printf("Matched: %s (%s) [%s]\n", hit.Title, hit.Year, hit.IMDbID)

		// 2. Build enabled providers from config (no hardcoded sources).
		specs := make([]struct {
			Type    string
			Enabled bool
			Options map[string]interface{}
		}, 0, len(cfg.Providers))
		for _, p := range cfg.Providers {
			specs = append(specs, struct {
				Type    string
				Enabled bool
				Options map[string]interface{}
			}{p.Type, p.Enabled, p.Options})
		}
		reg, err := provider.Build(specs)
		if err != nil {
			return err
		}

		// 3. Query providers with the resolved IMDb id + season/episode.
		q := provider.Query{IMDbID: hit.IMDbID, Title: hit.Title, Type: provider.Movie, Year: parseYear(hit.Year)}
		if isSeries {
			q.Type = provider.Series
			q.Season = season
			q.Episode = episode
		}
		fmt.Println("Searching providers...")
		streams := reg.SearchAll(context.Background(), q)
		if len(streams) == 0 {
			return fmt.Errorf("no streams found")
		}

		// 4. Let the user pick a stream (Bubble Tea).
		streamLabels := make([]string, len(streams))
		for i, s := range streams {
			quality := s.Quality
			if quality == "" {
				quality = "?"
			}
			streamLabels[i] = fmt.Sprintf("[%s] %s · %s (%d seeders)", quality, s.Title, s.Provider, s.Seeders)
		}
		si, err := ui.Select("Select a stream:", streamLabels)
		if err != nil {
			return err
		}
		sel := streams[si]

		// Record the play in history (best-effort; never blocks playback).
		if st, serr := store.Open(); serr == nil {
			_ = st.AddHistory(hit.Title, wantType, hit.IMDbID, season, episode)
			_ = st.Close()
		}

		// Download mode (--download/-d): hand the link to your torrent client and
		// exit, instead of streaming through peerflix + mpv.
		if downloadMode {
			return downloadStream(sel)
		}

		// 5. If the pick is a magnet/torrent, bridge it to a local HTTP URL that
		// mpv can play (peerflix). Direct http(s) streams skip this entirely.
		playURL := sel.URL
		if resolver.IsMagnet(playURL) {
			res, err := resolver.New(cfg.Resolver.Type, cfg.Resolver.Bin, cfg.Resolver.Args, cfg.Resolver.Timeout)
			if err != nil {
				return err
			}
			fmt.Println("Resolving torrent via bridge — buffering from peers, this can take a moment...")
			httpURL, cleanup, err := res.Resolve(playURL)
			if err != nil {
				return err
			}
			defer cleanup()
			playURL = httpURL
		}

		fmt.Printf("Launching %s: %s\n", cfg.Player, sel.Title)
		return player.Play(cfg.Player, cfg.PlayerArgs, playURL, sel.Title, sel.Headers)
	},
}

// downloadStream hands the selected stream to your system's default handler (a
// torrent client for magnets) instead of streaming it, and always prints the
// link so you can copy it manually. This is the "download it and play locally"
// path — no peerflix, no mpv.
func downloadStream(s provider.Stream) error {
	fmt.Printf("\nSelected: %s\n", s.Title)
	if s.Quality != "" {
		fmt.Printf("Quality:  %s\n", s.Quality)
	}
	if s.Size != "" {
		fmt.Printf("Size:     %s\n", s.Size)
	}
	if s.Seeders > 0 {
		fmt.Printf("Seeders:  %d\n", s.Seeders)
	}
	fmt.Printf("\nLink:\n%s\n\n", s.URL)

	if err := openInDefaultApp(s.URL); err != nil {
		fmt.Printf("Couldn't auto-open a torrent client (%v).\n", err)
		fmt.Println("Copy the link above into your torrent client (e.g. qBittorrent).")
		return nil
	}
	fmt.Println("Handed the link to your default torrent client.")
	fmt.Println("When it finishes, play the file with:  mpv \"path\\to\\file.mkv\"")
	return nil
}

// openInDefaultApp opens a URL/magnet with the OS default handler. On Windows it
// uses rundll32 (never cmd.exe) so the '&' and '%' in a magnet URI aren't mangled.
func openInDefaultApp(target string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", target).Start()
	case "darwin":
		return exec.Command("open", target).Start()
	case "android":
		// Termux (Go reports GOOS=android): hand the magnet/URL to an Android app
		// (torrent client, browser) via the activity manager. Prefer termux-open-url
		// when it's installed; otherwise fall back to `am start`.
		if _, err := exec.LookPath("termux-open-url"); err == nil {
			return exec.Command("termux-open-url", target).Start()
		}
		return exec.Command("am", "start", "-a", "android.intent.action.VIEW", "-d", target).Start()
	default:
		return exec.Command("xdg-open", target).Start()
	}
}

package cmd

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Divyesh172/cine/internal/config"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check that cine's tools and configuration are healthy",
	RunE: func(cmd *cobra.Command, args []string) error {
		problems := 0

		fmt.Println("cine doctor")
		fmt.Println("===========")

		cfg, cfgErr := config.Load()
		player := "mpv"
		if cfgErr == nil && cfg.Player != "" {
			player = cfg.Player
		}

		fmt.Println("\nRequired tools:")
		problems += checkBinary("curl", true)
		problems += checkBinary(player, true)

		fmt.Println("\nOptional tools:")
		problems += checkBinary("peerflix", false)
		problems += checkBinary("chafa", false)

		fmt.Println("\nConfiguration:")
		if cfgErr != nil {
			fmt.Printf("  x config: %v\n", cfgErr)
			problems++
		} else {
			fmt.Printf("  ok config loaded (player=%s, resolver=%s)\n", cfg.Player, cfg.Resolver.Type)
			enabled := 0
			for _, p := range cfg.Providers {
				if p.Enabled {
					enabled++
				}
			}
			if enabled == 0 {
				fmt.Println("  ! no providers enabled - enable the 'demo' provider to smoke-test playback")
			} else {
				fmt.Printf("  ok %d provider(s) enabled\n", enabled)
			}
		}

		if cfgErr == nil {
			fmt.Println("\nProvider endpoints:")
			for _, p := range cfg.Providers {
				if !p.Enabled {
					continue
				}
				ep, _ := p.Options["endpoint"].(string)
				if ep == "" {
					fmt.Printf("  - %s (%s): built-in, no endpoint\n", p.Name, p.Type)
					continue
				}
				problems += checkEndpoint(p.Name, ep)
			}
		}

		fmt.Println("\nData directory:")
		problems += checkDataDir()

		fmt.Println()
		if problems == 0 {
			fmt.Println("All checks passed. cine is ready to use.")
			return nil
		}
		return fmt.Errorf("%d problem(s) found - see output above", problems)
	},
}

func checkBinary(name string, required bool) int {
	path, err := exec.LookPath(name)
	if err != nil {
		if required {
			fmt.Printf("  x %s not found in PATH (required)\n", name)
			return 1
		}
		fmt.Printf("  ! %s not found in PATH (optional)\n", name)
		return 0
	}
	fmt.Printf("  ok %s (%s)\n", name, path)
	return 0
}

func checkEndpoint(name, endpoint string) int {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		fmt.Printf("  x %s: invalid endpoint %q\n", name, endpoint)
		return 1
	}
	host := u.Host
	if u.Port() == "" {
		if u.Scheme == "https" {
			host = net.JoinHostPort(u.Hostname(), "443")
		} else {
			host = net.JoinHostPort(u.Hostname(), "80")
		}
	}
	conn, err := net.DialTimeout("tcp", host, 5*time.Second)
	if err != nil {
		fmt.Printf("  x %s: cannot reach %s (%v)\n", name, endpoint, err)
		return 1
	}
	_ = conn.Close()
	fmt.Printf("  ok %s: %s reachable\n", name, endpoint)
	return 0
}

func checkDataDir() int {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".local", "share", "cine")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Printf("  x cannot create %s (%v)\n", dir, err)
		return 1
	}
	probe := filepath.Join(dir, ".doctor_probe")
	if err := os.WriteFile(probe, []byte("ok"), 0o644); err != nil {
		fmt.Printf("  x %s not writable (%v)\n", dir, err)
		return 1
	}
	_ = os.Remove(probe)
	fmt.Printf("  ok %s writable\n", dir)
	return 0
}

func init() { rootCmd.AddCommand(doctorCmd) }

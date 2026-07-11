package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Build metadata, injected at build/release time via -ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the cine version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cine %s (commit %s, built %s, %s/%s, %s)\n",
			Version, Commit, Date, runtime.GOOS, runtime.GOARCH, runtime.Version())
	},
}

func init() { rootCmd.AddCommand(versionCmd) }

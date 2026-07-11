package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/Divyesh172/cine/internal/meta"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search movies and TV shows (metadata only)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := meta.New()
		results, err := client.Search(context.Background(), args[0])
		if err != nil {
			return err
		}
		for i, r := range results {
			marker := " "
			if i == 0 {
				marker = "▶"
			}
			fmt.Printf("%s %s (%s)  [%s]\n", marker, r.Title, r.Year, r.IMDbID)
		}
		return nil
	},
}

package cmd

import (
	"fmt"

	"github.com/Divyesh172/cine/internal/store"
	"github.com/spf13/cobra"
)

var historyLimit int

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show recently played titles",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := store.Open()
		if err != nil {
			return err
		}
		defer st.Close()

		items, err := st.RecentHistory(historyLimit)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			fmt.Println("No history yet — play something first.")
			return nil
		}
		for _, h := range items {
			when := "-"
			if !h.WatchedAt.IsZero() {
				when = h.WatchedAt.Local().Format("2006-01-02 15:04")
			}
			label := h.Title
			if h.MediaType == "series" && (h.Season > 0 || h.Episode > 0) {
				label = fmt.Sprintf("%s S%02dE%02d", h.Title, h.Season, h.Episode)
			}
			fmt.Printf("%s  %-45s [%s]\n", when, label, h.IMDbID)
		}
		return nil
	},
}

func init() {
	historyCmd.Flags().IntVarP(&historyLimit, "limit", "n", 20, "max entries to show")
	rootCmd.AddCommand(historyCmd)
}

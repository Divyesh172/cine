package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/Divyesh172/cine/internal/meta"
	"github.com/Divyesh172/cine/internal/store"
	"github.com/Divyesh172/cine/internal/ui"
	"github.com/spf13/cobra"
)

var favoritesCmd = &cobra.Command{
	Use:     "favorites",
	Aliases: []string{"fav", "favourites"},
	Short:   "List your favorite titles",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := store.Open()
		if err != nil {
			return err
		}
		defer st.Close()

		favs, err := st.ListFavorites()
		if err != nil {
			return err
		}
		if len(favs) == 0 {
			fmt.Println("No favorites yet — add one with: cine favorites add \"<title>\"")
			return nil
		}
		for _, f := range favs {
			fmt.Printf("* %-45s [%s] (%s)\n", f.Title, f.IMDbID, f.MediaType)
		}
		return nil
	},
}

var favAddCmd = &cobra.Command{
	Use:   "add [title]",
	Short: "Search a title and add it to favorites",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.TrimSpace(strings.Join(args, " "))
		results, err := meta.New().Search(context.Background(), query)
		if err != nil {
			return err
		}
		if len(results) == 0 {
			return fmt.Errorf("no matches for %q", query)
		}
		labels := make([]string, len(results))
		for i, r := range results {
			labels[i] = fmt.Sprintf("%s (%s) [%s] - %s", r.Title, r.Year, r.IMDbID, r.Type)
		}
		idx, err := ui.Select("Add to favorites:", labels)
		if err != nil {
			return err
		}
		hit := results[idx]

		st, err := store.Open()
		if err != nil {
			return err
		}
		defer st.Close()
		if err := st.AddFavorite(hit.IMDbID, hit.Title, hit.Type); err != nil {
			return err
		}
		fmt.Printf("Added %s (%s) to favorites.\n", hit.Title, hit.Year)
		return nil
	},
}

var favRmCmd = &cobra.Command{
	Use:     "rm",
	Aliases: []string{"remove"},
	Short:   "Pick a favorite to remove",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := store.Open()
		if err != nil {
			return err
		}
		defer st.Close()

		favs, err := st.ListFavorites()
		if err != nil {
			return err
		}
		if len(favs) == 0 {
			fmt.Println("No favorites to remove.")
			return nil
		}
		labels := make([]string, len(favs))
		for i, f := range favs {
			labels[i] = fmt.Sprintf("%s [%s]", f.Title, f.IMDbID)
		}
		idx, err := ui.Select("Remove which favorite?", labels)
		if err != nil {
			return err
		}
		target := favs[idx]
		if err := st.RemoveFavorite(target.IMDbID); err != nil {
			return err
		}
		fmt.Printf("Removed %s from favorites.\n", target.Title)
		return nil
	},
}

func init() {
	favoritesCmd.AddCommand(favAddCmd)
	favoritesCmd.AddCommand(favRmCmd)
	rootCmd.AddCommand(favoritesCmd)
}

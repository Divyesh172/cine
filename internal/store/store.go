package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }

type HistoryItem struct {
	Title     string
	MediaType string
	IMDbID    string
	Season    int
	Episode   int
	WatchedAt time.Time
}

type Favorite struct {
	IMDbID    string
	Title     string
	MediaType string
}

func Open() (*Store, error) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".local", "share", "cine")
	_ = os.MkdirAll(dir, 0o755)
	db, err := sql.Open("sqlite", filepath.Join(dir, "cine.db"))
	if err != nil {
		return nil, err
	}
	schema := `
	CREATE TABLE IF NOT EXISTS history (
		id INTEGER PRIMARY KEY,
		title TEXT, media_type TEXT, imdb_id TEXT,
		season INT, episode INT, watched_at DATETIME
	);
	CREATE TABLE IF NOT EXISTS favorites (
		imdb_id TEXT PRIMARY KEY, title TEXT, media_type TEXT
	);`
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) AddHistory(title, mtype, imdb string, season, episode int) error {
	_, err := s.db.Exec(
		`INSERT INTO history (title, media_type, imdb_id, season, episode, watched_at) VALUES (?,?,?,?,?,?)`,
		title, mtype, imdb, season, episode, time.Now().Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) RecentHistory(limit int) ([]HistoryItem, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(
		`SELECT title, media_type, imdb_id, season, episode, watched_at<br>		 FROM history ORDER BY watched_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HistoryItem
	for rows.Next() {
		var h HistoryItem
		var when sql.NullString
		if err := rows.Scan(&h.Title, &h.MediaType, &h.IMDbID, &h.Season, &h.Episode, &when); err != nil {
			return nil, err
		}
		h.WatchedAt = parseWhen(when.String)
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *Store) AddFavorite(imdb, title, mtype string) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO favorites (imdb_id, title, media_type) VALUES (?,?,?)`,
		imdb, title, mtype,
	)
	return err
}

func (s *Store) ListFavorites() ([]Favorite, error) {
	rows, err := s.db.Query(`SELECT imdb_id, title, media_type FROM favorites ORDER BY title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Favorite
	for rows.Next() {
		var f Favorite
		if err := rows.Scan(&f.IMDbID, &f.Title, &f.MediaType); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Store) RemoveFavorite(imdb string) error {
	_, err := s.db.Exec(`DELETE FROM favorites WHERE imdb_id = ?`, imdb)
	return err
}

// parseWhen tolerates the couple of timestamp formats sqlite may hand back.
func parseWhen(s string) time.Time {
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

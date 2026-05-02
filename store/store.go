package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"notas/config"
)

type Note struct {
	ID        int64    `json:"id"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Status    string   `json:"status"`
	Pinned    bool     `json:"pinned"`
	Tags      []string `json:"tags"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
	Source    string   `json:"source"`
	OriginID  string   `json:"origin_id"`
}

var tagRegex = regexp.MustCompile(`(?i)#([a-z0-9_]+)`)

func extractTags(text string) []string {
	matches := tagRegex.FindAllStringSubmatch(text, -1)
	tagMap := make(map[string]bool)
	var tags []string
	for _, m := range matches {
		tag := strings.ToLower(m[1])
		if !tagMap[tag] {
			tagMap[tag] = true
			tags = append(tags, tag)
		}
	}
	return tags
}

type Stats struct {
	TotalNotes int
}

type TagStats struct {
	Name  string
	Count int
}

type Store struct {
	db       *sql.DB
	obsidian *ObsidianProvider
	notion   *NotionProvider
}

// New initializes the SQLite database and performs automatic migration from JSON if present.
func New(dbPath string, cfg config.Config) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	s := &Store{db: db}

	if cfg.ObsidianVaultPath != "" {
		s.obsidian = NewObsidianProvider(cfg.ObsidianVaultPath)
		_ = s.obsidian.Init()
	}

	if cfg.NotionToken != "" && cfg.NotionDatabaseID != "" {
		s.notion = NewNotionProvider(cfg.NotionToken, cfg.NotionDatabaseID)
		_ = s.notion.Init()
	}

	if err := s.initSchema(); err != nil {
		return nil, err
	}

	// Automatic migration from old JSON to new SQLite DB
	jsonPath := strings.Replace(dbPath, ".db", ".json", 1)
	if _, err := os.Stat(jsonPath); err == nil {
		s.migrateFromJSON(jsonPath)
	}

	return s, nil
}

func (s *Store) initSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS notes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		status TEXT NOT NULL,
		pinned BOOLEAN NOT NULL DEFAULT 0,
		tags TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);`
	_, err := s.db.Exec(query)
	return err
}

func (s *Store) migrateFromJSON(jsonPath string) {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return
	}
	var notes []Note
	if err := json.Unmarshal(data, &notes); err != nil {
		return
	}

	var count int
	s.db.QueryRow(`SELECT COUNT(*) FROM notes`).Scan(&count)
	if count > 0 {
		os.Rename(jsonPath, jsonPath+".bak")
		return
	}

	tx, err := s.db.Begin()
	if err != nil {
		return
	}

	stmt, err := tx.Prepare(`INSERT INTO notes (id, title, content, status, pinned, tags, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return
	}
	defer stmt.Close()

	for _, n := range notes {
		tagsJSON, _ := json.Marshal(n.Tags)
		stmt.Exec(n.ID, n.Title, n.Content, n.Status, n.Pinned, string(tagsJSON), n.CreatedAt, n.UpdatedAt)
	}
	tx.Commit()

	os.Rename(jsonPath, jsonPath+".bak")
}

// scanNote is a helper to scan DB rows into Note structs.
type Scanner interface {
	Scan(dest ...interface{}) error
}

func scanNote(scanner Scanner) (Note, bool) {
	var n Note
	var tagsJSON string
	err := scanner.Scan(&n.ID, &n.Title, &n.Content, &n.Status, &n.Pinned, &tagsJSON, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return n, false
	}
	_ = json.Unmarshal([]byte(tagsJSON), &n.Tags)
	n.Source = "local"
	return n, true
}

func (s *Store) SaveNote(title, content, status string) (Note, error) {
	now := time.Now().Format(time.RFC3339)
	tags := extractTags(title + " " + content)
	tagsJSON, _ := json.Marshal(tags)

	res, err := s.db.Exec(`
		INSERT INTO notes (title, content, status, pinned, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, title, content, status, false, string(tagsJSON), now, now)

	if err != nil {
		return Note{}, err
	}

	id, _ := res.LastInsertId()
	return Note{
		ID:        id,
		Title:     title,
		Content:   content,
		Status:    status,
		Pinned:    false,
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
		Source:    "local",
	}, nil
}

func (s *Store) UpdateNote(id int64, title, content string) error {
	if s.notion != nil && id >= 20000000 {
		return s.notion.UpdateNote(id, title, content, "saved", false, extractTags(title+" "+content))
	}
	if s.obsidian != nil && id >= 10000000 {
		return s.obsidian.UpdateNote(id, title, content, "saved", false, extractTags(title+" "+content))
	}

	now := time.Now().Format(time.RFC3339)
	tags := extractTags(title + " " + content)
	tagsJSON, _ := json.Marshal(tags)

	_, err := s.db.Exec(`
		UPDATE notes
		SET title = ?, content = ?, tags = ?, updated_at = ?
		WHERE id = ?
	`, title, content, string(tagsJSON), now, id)
	return err
}

func (s *Store) DeleteNote(id int64) error {
	if s.notion != nil && id >= 20000000 {
		return s.notion.DeleteNote(id)
	}
	if s.obsidian != nil && id >= 10000000 {
		return s.obsidian.DeleteNote(id)
	}
	_, err := s.db.Exec(`DELETE FROM notes WHERE id = ?`, id)
	return err
}

func (s *Store) TogglePin(id int64) error {
	if s.notion != nil && id >= 20000000 {
		return s.notion.PinNote(id, true)
	}
	if s.obsidian != nil && id >= 10000000 {
		return s.obsidian.PinNote(id, true) // Toggle logic needed if we ever support it in Obsidian
	}
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(`
		UPDATE notes
		SET pinned = NOT pinned, updated_at = ?
		WHERE id = ?
	`, now, id)
	return err
}

func (s *Store) GetRecentNotes() []Note {
	rows, err := s.db.Query(`
		SELECT id, title, content, status, pinned, tags, created_at, updated_at
		FROM notes
		ORDER BY pinned DESC, updated_at DESC
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var list []Note
	for rows.Next() {
		if n, ok := scanNote(rows); ok {
			list = append(list, n)
		}
	}

	if s.obsidian != nil {
		obsNotes, _ := s.obsidian.GetNotes("", "")
		list = append(list, obsNotes...)
	}

	if s.notion != nil {
		notionNotes, _ := s.notion.GetNotes("", "")
		list = append(list, notionNotes...)
	}

	if s.obsidian != nil || s.notion != nil {
		// Re-sort everything since we merged sources
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].Pinned != list[j].Pinned {
				return list[i].Pinned
			}
			return list[i].UpdatedAt > list[j].UpdatedAt
		})
	}

	return list
}

func (s *Store) SearchNotes(query string) []Note {
	q := strings.TrimSpace(query)
	if q == "" {
		return s.GetRecentNotes()
	}

	isTagSearch := strings.HasPrefix(strings.ToLower(q), "tag:")
	var list []Note

	if isTagSearch {
		searchTag := strings.TrimPrefix(strings.ToLower(q), "tag:")
		// Fast SQLite text search within JSON string
		searchStr := fmt.Sprintf(`%%"%s"%%`, searchTag)
		rows, err := s.db.Query(`
			SELECT id, title, content, status, pinned, tags, created_at, updated_at
			FROM notes
			WHERE tags LIKE ?
			ORDER BY pinned DESC, updated_at DESC
		`, searchStr)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				if n, ok := scanNote(rows); ok {
					// Validate exact match after LIKE
					for _, t := range n.Tags {
						if t == searchTag {
							list = append(list, n)
							break
						}
					}
				}
			}
		}

		if s.obsidian != nil {
			obsNotes, _ := s.obsidian.GetNotes("", searchTag)
			list = append(list, obsNotes...)
		}

		if s.notion != nil {
			notionNotes, _ := s.notion.GetNotes("", searchTag)
			list = append(list, notionNotes...)
		}

		if s.obsidian != nil || s.notion != nil {
			sort.SliceStable(list, func(i, j int) bool {
				if list[i].Pinned != list[j].Pinned {
					return list[i].Pinned
				}
				return list[i].UpdatedAt > list[j].UpdatedAt
			})
		}

		return list
	}

	searchPattern := "%" + q + "%"
	rows, err := s.db.Query(`
		SELECT id, title, content, status, pinned, tags, created_at, updated_at
		FROM notes
		WHERE title LIKE ? OR content LIKE ?
		ORDER BY pinned DESC, updated_at DESC
		LIMIT 100
	`, searchPattern, searchPattern)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			if n, ok := scanNote(rows); ok {
				list = append(list, n)
			}
		}
	}

	if s.obsidian != nil {
		obsNotes, _ := s.obsidian.GetNotes(q, "")
		list = append(list, obsNotes...)
	}

	if s.notion != nil {
		notionNotes, _ := s.notion.GetNotes(q, "")
		list = append(list, notionNotes...)
	}

	if s.obsidian != nil || s.notion != nil {
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].Pinned != list[j].Pinned {
				return list[i].Pinned
			}
			return list[i].UpdatedAt > list[j].UpdatedAt
		})
	}

	return list
}

func (s *Store) GetNote(id int64) (Note, bool) {
	if s.notion != nil && id >= 20000000 {
		n, err := s.notion.GetNoteByID(id)
		if err != nil {
			return Note{}, false
		}
		return *n, true
	}
	if s.obsidian != nil && id >= 10000000 && id < 20000000 {
		n, err := s.obsidian.GetNoteByID(id)
		if err != nil {
			return Note{}, false
		}
		return *n, true
	}
	row := s.db.QueryRow(`SELECT id, title, content, status, pinned, tags, created_at, updated_at FROM notes WHERE id = ?`, id)
	return scanNote(row)
}

func (s *Store) GetStats() Stats {
	var count int
	s.db.QueryRow(`SELECT COUNT(*) FROM notes`).Scan(&count)
	return Stats{TotalNotes: count}
}

func (s *Store) GetTags() []TagStats {
	var list []TagStats
	tagMap := make(map[string]int)

	if s.obsidian != nil {
		tags, _ := s.obsidian.GetTags()
		for _, t := range tags {
			tagMap[t.Name] += t.Count
		}
	}

	if s.notion != nil {
		tags, _ := s.notion.GetTags()
		for _, t := range tags {
			tagMap[t.Name] += t.Count
		}
	}

	rows, err := s.db.Query(`SELECT tags FROM notes`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tagsJSON string
			if err := rows.Scan(&tagsJSON); err == nil {
				var tags []string
				if json.Unmarshal([]byte(tagsJSON), &tags) == nil {
					for _, t := range tags {
						tagMap[t]++
					}
				}
			}
		}
	}

	for k, v := range tagMap {
		list = append(list, TagStats{Name: k, Count: v})
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].Count != list[j].Count {
			return list[i].Count > list[j].Count
		}
		return list[i].Name < list[j].Name
	})

	return list
}

func (s *Store) ExportNotes(ids []int64, exportDir string) (int, error) {
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return 0, err
	}

	query := `SELECT id, title, content, status, pinned, tags, created_at, updated_at FROM notes`
	var args []interface{}

	if len(ids) > 0 {
		placeholders := make([]string, len(ids))
		for i, id := range ids {
			placeholders[i] = "?"
			args = append(args, id)
		}
		query += " WHERE id IN (" + strings.Join(placeholders, ",") + ")"
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		if note, ok := scanNote(rows); ok {
			safeTitle := strings.ReplaceAll(strings.ToLower(note.Title), " ", "-")
			safeTitle = strings.ReplaceAll(safeTitle, "/", "-")
			filename := fmt.Sprintf("%d_%s.md", note.ID, safeTitle)
			path := filepath.Join(exportDir, filename)

			content := fmt.Sprintf("---\ntitle: %s\nid: %d\ndate: %s\ntags: [%s]\n---\n\n%s\n",
				note.Title, note.ID, note.CreatedAt, strings.Join(note.Tags, ", "), note.Content)

			if err := os.WriteFile(path, []byte(content), 0644); err == nil {
				count++
			}
		}
	}
	return count, nil
}

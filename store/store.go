package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type Note struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Status    string   `json:"status"`
	Pinned    bool     `json:"pinned"`
	Tags      []string `json:"tags"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
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
	path  string
	notes map[int64]Note
	mu    sync.RWMutex
}

func New(filepath string) (*Store, error) {
	s := &Store{
		path:  filepath,
		notes: make(map[int64]Note),
	}
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	var notesList []Note
	if err := json.Unmarshal(data, &notesList); err != nil {
		return err
	}

	for _, n := range notesList {
		s.notes[n.ID] = n
	}
	return nil
}

func (s *Store) save() error {
	var notesList []Note
	for _, n := range s.notes {
		notesList = append(notesList, n)
	}

	data, err := json.MarshalIndent(notesList, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0644)
}

func (s *Store) SaveNote(title, content, status string) (Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	
	var maxID int64
	for id := range s.notes {
		if id > maxID {
			maxID = id
		}
	}
	
	note := Note{
		ID:        maxID + 1,
		Title:     title,
		Content:   content,
		Status:    status,
		Tags:      extractTags(title + " " + content),
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.notes[note.ID] = note
	return note, s.save()
}

func (s *Store) UpdateNote(id int64, title, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	note, exists := s.notes[id]
	if !exists {
		return os.ErrNotExist
	}

	note.Title = title
	note.Content = content
	note.Tags = extractTags(title + " " + content)
	note.UpdatedAt = time.Now().Format(time.RFC3339)
	s.notes[id] = note

	return s.save()
}

func (s *Store) DeleteNote(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.notes[id]; !exists {
		return os.ErrNotExist
	}

	delete(s.notes, id)
	return s.save()
}

func (s *Store) TogglePin(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	note, exists := s.notes[id]
	if !exists {
		return os.ErrNotExist
	}

	note.Pinned = !note.Pinned
	note.UpdatedAt = time.Now().Format(time.RFC3339)
	s.notes[id] = note

	return s.save()
}

func (s *Store) GetRecentNotes() []Note {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []Note
	for _, n := range s.notes {
		list = append(list, n)
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].Pinned != list[j].Pinned {
			return list[i].Pinned
		}
		return list[i].UpdatedAt > list[j].UpdatedAt
	})

	return list
}

func (s *Store) SearchNotes(query string) []Note {
	s.mu.RLock()
	defer s.mu.RUnlock()

	q := strings.ToLower(strings.TrimSpace(query))
	isTagSearch := strings.HasPrefix(q, "tag:")
	var searchTag string
	if isTagSearch {
		searchTag = strings.TrimPrefix(q, "tag:")
	}

	var list []Note
	for _, n := range s.notes {
		if isTagSearch {
			hasTag := false
			for _, t := range n.Tags {
				if t == searchTag {
					hasTag = true
					break
				}
			}
			if hasTag {
				list = append(list, n)
			}
		} else {
			if strings.Contains(strings.ToLower(n.Title), q) || strings.Contains(strings.ToLower(n.Content), q) {
				list = append(list, n)
			}
		}
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].Pinned != list[j].Pinned {
			return list[i].Pinned
		}
		return list[i].UpdatedAt > list[j].UpdatedAt
	})

	return list
}

func (s *Store) GetNote(id int64) (Note, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.notes[id]
	return n, ok
}

func (s *Store) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Stats{TotalNotes: len(s.notes)}
}

func (s *Store) GetTags() []TagStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tagMap := make(map[string]int)
	for _, n := range s.notes {
		for _, tag := range n.Tags {
			tagMap[tag]++
		}
	}

	var tags []TagStats
	for name, count := range tagMap {
		tags = append(tags, TagStats{Name: name, Count: count})
	}

	sort.Slice(tags, func(i, j int) bool {
		if tags[i].Count != tags[j].Count {
			return tags[i].Count > tags[j].Count
		}
		return tags[i].Name < tags[j].Name
	})

	return tags
}

func (s *Store) ExportNotes(ids []int64, exportDir string) (int, error) {
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return 0, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, note := range s.notes {
		exportThis := false
		if len(ids) == 0 {
			exportThis = true
		} else {
			for _, id := range ids {
				if id == note.ID {
					exportThis = true
					break
				}
			}
		}

		if !exportThis {
			continue
		}

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
	return count, nil
}

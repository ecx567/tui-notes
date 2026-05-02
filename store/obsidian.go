package store

import (
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ObsidianProvider struct {
	VaultPath string
	idMap     map[int64]string
	mu        sync.RWMutex
}

func NewObsidianProvider(vaultPath string) *ObsidianProvider {
	return &ObsidianProvider{
		VaultPath: vaultPath,
		idMap:     make(map[int64]string),
	}
}

func (p *ObsidianProvider) Init() error {
	if p.VaultPath == "" {
		return nil
	}
	info, err := os.Stat(p.VaultPath)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("obsidian vault path is invalid: %s", p.VaultPath)
	}
	return nil
}

func (p *ObsidianProvider) Close() error {
	return nil
}

func (p *ObsidianProvider) generateID(path string) int64 {
	// Simple stable hash based on relative path from vault
	relPath, err := filepath.Rel(p.VaultPath, path)
	if err != nil {
		relPath = path
	}
	// Use CRC32. To avoid collisions with DB, set the high bit (essentially making it negative or large positive)
	// But int64 from uint32 is always positive. Let's just add 10,000,000 to the uint32 to keep it out of local DB range.
	hash := crc32.ChecksumIEEE([]byte(relPath))
	return int64(hash) + 10000000
}

func (p *ObsidianProvider) GetNotes(query, tag string) ([]Note, error) {
	var notes []Note

	if p.VaultPath == "" {
		return notes, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Clear idMap on each full fetch
	p.idMap = make(map[int64]string)

	err := filepath.Walk(p.VaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // ignore errors
		}
		if info.IsDir() {
			// Skip hidden directories (like .obsidian, .git)
			if strings.HasPrefix(info.Name(), ".") && path != p.VaultPath {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			contentBytes, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			content := string(contentBytes)
			title := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))

			// Filter by query
			if query != "" {
				q := strings.ToLower(query)
				if !strings.Contains(strings.ToLower(title), q) && !strings.Contains(strings.ToLower(content), q) {
					return nil
				}
			}

			// Extract tags
			tags := extractTags(content)

			// Filter by tag
			if tag != "" {
				tagLower := strings.ToLower(tag)
				hasTag := false
				for _, t := range tags {
					if t == tagLower {
						hasTag = true
						break
					}
				}
				if !hasTag {
					return nil
				}
			}

			id := p.generateID(path)
			p.idMap[id] = path

			createdAt := info.ModTime().Format(time.RFC3339)

			notes = append(notes, Note{
				ID:        id,
				Title:     title,
				Content:   content,
				Status:    "done",
				Pinned:    false,
				Tags:      tags,
				CreatedAt: createdAt,
				UpdatedAt: createdAt,
				Source:    "obsidian",
				OriginID:  path,
			})
		}
		return nil
	})

	return notes, err
}

func (p *ObsidianProvider) GetNoteByID(id int64) (*Note, error) {
	p.mu.RLock()
	path, ok := p.idMap[id]
	p.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("note id %d not found in obsidian vault", id)
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(contentBytes)
	title := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))

	tags := extractTags(content)

	createdAt := info.ModTime().Format(time.RFC3339)

	return &Note{
		ID:        id,
		Title:     title,
		Content:   content,
		Status:    "done",
		Pinned:    false,
		Tags:      tags,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Source:    "obsidian",
		OriginID:  path,
	}, nil
}

func (p *ObsidianProvider) SaveNote(title, content, status string, pinned bool, tags []string) (Note, error) {
	if p.VaultPath == "" {
		return Note{}, fmt.Errorf("obsidian vault path not configured")
	}

	// We append tags to the bottom if they aren't already there
	for _, t := range tags {
		if !strings.Contains(content, "#"+t) {
			if !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			content += "#" + t + " "
		}
	}
	content = strings.TrimSpace(content)

	// In Obsidian, file name is the title. Sanitize title for filename
	safeTitle := strings.ReplaceAll(title, "/", "-")
	safeTitle = strings.ReplaceAll(safeTitle, "\\", "-")
	path := filepath.Join(p.VaultPath, safeTitle+".md")

	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return Note{}, err
	}

	id := p.generateID(path)
	p.mu.Lock()
	p.idMap[id] = path
	p.mu.Unlock()

	now := time.Now().Format(time.RFC3339)

	return Note{
		ID:        id,
		Title:     title,
		Content:   content,
		Status:    "done",
		Pinned:    false,
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
		Source:    "obsidian",
		OriginID:  path,
	}, nil
}

func (p *ObsidianProvider) UpdateNote(id int64, title, content, status string, pinned bool, tags []string) error {
	p.mu.RLock()
	path, ok := p.idMap[id]
	p.mu.RUnlock()

	if !ok {
		return fmt.Errorf("note id %d not found in obsidian vault", id)
	}

	for _, t := range tags {
		if !strings.Contains(content, "#"+t) {
			if !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			content += "#" + t + " "
		}
	}
	content = strings.TrimSpace(content)

	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return err
	}

	// If title changed, we need to rename the file
	safeTitle := strings.ReplaceAll(title, "/", "-")
	safeTitle = strings.ReplaceAll(safeTitle, "\\", "-")
	newPath := filepath.Join(filepath.Dir(path), safeTitle+".md")
	
	if newPath != path {
		err = os.Rename(path, newPath)
		if err == nil {
			newID := p.generateID(newPath)
			p.mu.Lock()
			delete(p.idMap, id)
			p.idMap[newID] = newPath
			p.mu.Unlock()
		}
	}

	return nil
}

func (p *ObsidianProvider) DeleteNote(id int64) error {
	p.mu.RLock()
	path, ok := p.idMap[id]
	p.mu.RUnlock()

	if !ok {
		return fmt.Errorf("note id %d not found in obsidian vault", id)
	}

	err := os.Remove(path)
	if err != nil {
		return err
	}

	p.mu.Lock()
	delete(p.idMap, id)
	p.mu.Unlock()

	return nil
}

func (p *ObsidianProvider) PinNote(id int64, pinned bool) error {
	// Pinning not natively supported by standard md files without frontmatter.
	// For now, it's a no-op.
	return nil
}

func (p *ObsidianProvider) GetTags() ([]TagStats, error) {
	// Not critical for now, aggregate provider will do it or we can just calculate it here
	// This can be slow if we parse everything just for tags, but we'll re-use GetNotes logic
	notes, err := p.GetNotes("", "")
	if err != nil {
		return nil, err
	}
	
	tagMap := make(map[string]int)
	for _, n := range notes {
		for _, t := range n.Tags {
			tagMap[t]++
		}
	}

	var tags []TagStats
	for name, count := range tagMap {
		tags = append(tags, TagStats{Name: name, Count: count})
	}
	return tags, nil
}

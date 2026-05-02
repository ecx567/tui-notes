package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"sync"
	"time"
)

const notionAPIURL = "https://api.notion.com/v1"
const notionVersion = "2022-06-28"

type NotionProvider struct {
	Token      string
	DatabaseID string
	client     *http.Client
	idMap      map[int64]string // Maps our int64 ID to Notion's string UUID
	mu         sync.RWMutex
}

func NewNotionProvider(token, databaseID string) *NotionProvider {
	return &NotionProvider{
		Token:      token,
		DatabaseID: databaseID,
		client:     &http.Client{Timeout: 10 * time.Second},
		idMap:      make(map[int64]string),
	}
}

func (p *NotionProvider) Init() error {
	if p.Token == "" || p.DatabaseID == "" {
		return nil // Not configured
	}
	
	// Test connection by fetching the database metadata
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/databases/%s", notionAPIURL, p.DatabaseID), nil)
	if err != nil {
		return err
	}
	
	req.Header.Add("Authorization", "Bearer "+p.Token)
	req.Header.Add("Notion-Version", notionVersion)
	
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to notion: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("notion api error (%d): %s", resp.StatusCode, string(body))
	}
	
	return nil
}

func (p *NotionProvider) Close() error {
	return nil
}

func (p *NotionProvider) generateID(uuid string) int64 {
	hash := crc32.ChecksumIEEE([]byte(uuid))
	// Add 20,000,000 to keep it out of local DB and Obsidian range
	return int64(hash) + 20000000
}

func (p *NotionProvider) GetNotes(query, tag string) ([]Note, error) {
	var notes []Note
	if p.Token == "" || p.DatabaseID == "" {
		return notes, nil
	}
	
	// Query the database
	// For simplicity, we fetch all (or up to 100) and filter in-memory initially
	// A more robust implementation would use Notion's filtering API if query/tag is provided.
	
	reqBody := []byte(`{"page_size": 100}`)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/databases/%s/query", notionAPIURL, p.DatabaseID), bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	
	req.Header.Add("Authorization", "Bearer "+p.Token)
	req.Header.Add("Notion-Version", notionVersion)
	req.Header.Add("Content-Type", "application/json")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("notion query failed: %d", resp.StatusCode)
	}
	
	// Fast parse just enough to get IDs and titles (we'll fetch content only when asked via GetNoteByID)
	var result struct {
		Results []struct {
			ID         string `json:"id"`
			Properties map[string]interface{} `json:"properties"`
			CreatedTime string `json:"created_time"`
		} `json:"results"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	p.mu.Lock()
	p.idMap = make(map[int64]string) // Reset map
	
	for _, page := range result.Results {
		id := p.generateID(page.ID)
		p.idMap[id] = page.ID
		
		// Attempt to extract title from properties dynamically
		title := "Untitled"
		for _, prop := range page.Properties {
			propMap, ok := prop.(map[string]interface{})
			if !ok {
				continue
			}
			
			if propMap["type"] == "title" {
				titleArr, ok := propMap["title"].([]interface{})
				if ok && len(titleArr) > 0 {
					firstTitleObj, ok := titleArr[0].(map[string]interface{})
					if ok {
						if plainText, ok := firstTitleObj["plain_text"].(string); ok {
							title = plainText
						}
					}
				}
			}
		}
		
		// TODO: Extract tags and status dynamically
		
		notes = append(notes, Note{
			ID:        id,
			Title:     title,
			Content:   "(Contenido cargado bajo demanda desde Notion)",
			Status:    "done",
			Pinned:    false,
			Tags:      []string{},
			CreatedAt: page.CreatedTime,
			UpdatedAt: page.CreatedTime,
			Source:    "notion",
			OriginID:  page.ID,
		})
	}
	p.mu.Unlock()
	
	return notes, nil
}

func (p *NotionProvider) GetNoteByID(id int64) (*Note, error) {
	p.mu.RLock()
	uuid, ok := p.idMap[id]
	p.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("note id %d not found in notion memory map", id)
	}
	
	// Fetch Blocks (Content)
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/blocks/%s/children?page_size=100", notionAPIURL, uuid), nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Add("Authorization", "Bearer "+p.Token)
	req.Header.Add("Notion-Version", notionVersion)
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch notion blocks: %d", resp.StatusCode)
	}
	
	var result struct {
		Results []struct {
			Type string `json:"type"`
			Paragraph struct {
				RichText []struct {
					PlainText string `json:"plain_text"`
				} `json:"rich_text"`
			} `json:"paragraph"`
			// Other blocks (heading_1, bulleted_list_item, etc.) would be mapped here for a full implementation
		} `json:"results"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	var contentBuilder string
	for _, block := range result.Results {
		if block.Type == "paragraph" {
			for _, text := range block.Paragraph.RichText {
				contentBuilder += text.PlainText
			}
			contentBuilder += "\n"
		} else {
			contentBuilder += fmt.Sprintf("[Bloque no soportado: %s]\n", block.Type)
		}
	}
	
	// Fallback creation time
	now := time.Now().Format(time.RFC3339)
	
	return &Note{
		ID:        id,
		Title:     "Notion Page", // We could fetch the page metadata again, or rely on the list view
		Content:   contentBuilder,
		Status:    "done",
		Source:    "notion",
		OriginID:  uuid,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (p *NotionProvider) SaveNote(title, content, status string, pinned bool, tags []string) (Note, error) {
	return Note{}, fmt.Errorf("creating notes in notion is currently not supported (read-only mode)")
}

func (p *NotionProvider) UpdateNote(id int64, title, content, status string, pinned bool, tags []string) error {
	return fmt.Errorf("updating notes in notion is currently not supported (read-only mode)")
}

func (p *NotionProvider) DeleteNote(id int64) error {
	return fmt.Errorf("deleting notes in notion is currently not supported (read-only mode)")
}

func (p *NotionProvider) PinNote(id int64, pinned bool) error {
	return nil
}

func (p *NotionProvider) GetTags() ([]TagStats, error) {
	return []TagStats{}, nil
}

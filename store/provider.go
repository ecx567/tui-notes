package store

// NoteProvider defines the interface that all note sources (local SQLite, Obsidian, Notion)
// must implement to be searchable and interactable within the TUI.
type NoteProvider interface {
	Name() string
	GetRecentNotes() []Note
	SearchNotes(query string) []Note
	GetNote(id string) (Note, bool)
	SaveNote(title, content, status string) (Note, error)
	UpdateNote(id string, title, content string) error
	DeleteNote(id string) error
}

// Ensure the local SQLite Store implements NoteProvider in the future.
// Note: Since SQLite uses int64 for IDs and external providers use string UUIDs or file paths,
// we'll need to adapt the UI to handle string IDs. For now, we define the interface.

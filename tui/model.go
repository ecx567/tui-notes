package tui

import (
	"os"
	"notas/store"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type Screen int

const (
	ScreenDashboard Screen = iota
	ScreenSearch
	ScreenRecent
	ScreenTags
	ScreenNoteDetail
	ScreenCreateNote
	ScreenEditorSelect
	ScreenExportSelect
	ScreenFileExplorer
)

// Messages
type statsLoadedMsg struct{ stats store.Stats }
type searchResultsMsg struct {
	results []store.Note
	query   string
}
type recentNotesMsg struct{ notes []store.Note }
type noteDetailMsg struct{ note store.Note }
type noteSavedMsg struct{ note store.Note }
type noteDeletedMsg struct{}
type filesLoadedMsg struct{ files []os.DirEntry }
type errMsg struct{ err error }
type editorFinishedMsg struct {
	id      int64
	content string
}

type Editor struct {
	Name    string
	Command string
	Args    []string
}

type Model struct {
	store      *store.Store
	Screen     Screen
	PrevScreen Screen
	Width      int
	Height     int
	Cursor     int
	Scroll     int
	ErrorMsg   string
	SuccessMsg string

	Stats store.Stats

	SearchInput   textinput.Model
	SearchQuery   string
	SearchResults []store.Note

	RecentNotes []store.Note
	TagsList    []store.TagStats

	SelectedNote *store.Note
	DetailScroll int

	// Create Note
	EditingNoteID  int64
	NoteTitleInput textinput.Model
	NoteContent    textarea.Model
	EditingField   int // 0 for title, 1 for content

	// Editor
	AvailableEditors []Editor
	EditorCursor     int

	// Export
	ExportSelected map[int64]bool

	// File Explorer
	CurrentPath      string
	Files            []os.DirEntry
	FileCursor       int
	FileScroll       int
	ViewingLocalFile bool
	LocalFilePath    string
}

func New(s *store.Store) Model {
	ti := textinput.New()
	ti.Placeholder = "Buscar notas..."
	ti.CharLimit = 256
	ti.Width = 60

	titleInput := textinput.New()
	titleInput.Placeholder = "Título de la nota..."
	titleInput.CharLimit = 100
	titleInput.Width = 60

	ta := textarea.New()
	ta.Placeholder = "Escribe tu nota aquí...\n\n(Auto-guardado activo)\nPresiona Esc para salir y guardar."
	ta.ShowLineNumbers = false
	ta.SetHeight(10)
	ta.SetWidth(60)
	ta.CharLimit = 10000

	return Model{
		store:          s,
		Screen:         ScreenDashboard,
		SearchInput:    ti,
		NoteTitleInput: titleInput,
		NoteContent:    ta,
		ExportSelected: make(map[int64]bool),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		textarea.Blink,
		m.loadStatsCmd(),
		tea.EnterAltScreen,
	)
}

// ─── Data Loaders ───────────────────────────────────────────────────────

func (m Model) loadStatsCmd() tea.Cmd {
	return func() tea.Msg {
		return statsLoadedMsg{stats: m.store.GetStats()}
	}
}

func (m Model) loadRecentNotesCmd() tea.Cmd {
	return func() tea.Msg {
		return recentNotesMsg{notes: m.store.GetRecentNotes()}
	}
}

type tagsLoadedMsg struct{ tags []store.TagStats }

func (m Model) loadTagsCmd() tea.Cmd {
	return func() tea.Msg {
		return tagsLoadedMsg{tags: m.store.GetTags()}
	}
}

func (m Model) searchNotesCmd(query string) tea.Cmd {
	return func() tea.Msg {
		return searchResultsMsg{
			query:   query,
			results: m.store.SearchNotes(query),
		}
	}
}

func (m Model) loadNoteDetailCmd(id int64) tea.Cmd {
	return func() tea.Msg {
		if note, ok := m.store.GetNote(id); ok {
			return noteDetailMsg{note: note}
		}
		return nil
	}
}

func (m Model) saveNoteCmd(title, content string) tea.Cmd {
	return func() tea.Msg {
		if title == "" {
			title = "Nota sin título"
		}
		note, err := m.store.SaveNote(title, content, "saved")
		if err != nil {
			return nil
		}
		return noteSavedMsg{note: note}
	}
}

func (m Model) updateNoteCmd(id int64, title, content string) tea.Cmd {
	return func() tea.Msg {
		if title == "" {
			title = "Nota sin título"
		}
		err := m.store.UpdateNote(id, title, content)
		if err != nil {
			return nil
		}
		if note, ok := m.store.GetNote(id); ok {
			return noteSavedMsg{note: note}
		}
		return nil
	}
}

func (m Model) deleteNoteCmd(id int64) tea.Cmd {
	return func() tea.Msg {
		if err := m.store.DeleteNote(id); err != nil {
			return errMsg{err}
		}
		return noteDeletedMsg{}
	}
}

func (m Model) togglePinCmd(id int64) tea.Cmd {
	return func() tea.Msg {
		if err := m.store.TogglePin(id); err != nil {
			return errMsg{err}
		}
		// We reuse noteDeletedMsg here to trigger a screen refresh that preserves state
		// Or we can return a new message type, but noteSavedMsg resets to dashboard.
		// Wait, noteSavedMsg reloads stats and might change screen. Let's return a special msg or just reload the current list.
		// Actually, let's just use noteDeletedMsg because it handles reloading Recent or Search lists properly.
		return noteDeletedMsg{}
	}
}

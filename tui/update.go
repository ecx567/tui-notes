package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"notas/store"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ─── Update ──────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.NoteContent.SetWidth(msg.Width - 10)
		m.NoteContent.SetHeight(msg.Height - 15)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.Screen == ScreenSearch && m.SearchInput.Focused() {
			return m.handleSearchInputKeys(msg)
		}
		if m.Screen == ScreenExportSelect && m.SearchInput.Focused() {
			return m.handleExportInputKeys(msg)
		}
		if m.Screen == ScreenFileExplorer && m.SearchInput.Focused() {
			return m.handleFileExplorerInputKeys(msg)
		}
		if m.Screen == ScreenCreateNote {
			return m.handleCreateNoteMsg(msg)
		}
		return m.handleKeyPress(msg.String())

	case statsLoadedMsg:
		m.Stats = msg.stats
		return m, nil

	case searchResultsMsg:
		m.SearchResults = msg.results
		m.SearchQuery = msg.query
		return m, nil

	case recentNotesMsg:
		m.RecentNotes = msg.notes
		return m, nil

	case tagsLoadedMsg:
		m.TagsList = msg.tags
		return m, nil

	case filesLoadedMsg:
		m.Files = msg.files
		return m, nil

	case noteDetailMsg:
		m.SelectedNote = &msg.note
		m.Screen = ScreenNoteDetail
		m.DetailScroll = 0
		return m, nil
	
	case noteSavedMsg:
		if m.ViewingLocalFile && m.SelectedNote != nil && m.SelectedNote.ID == 0 {
			m.SelectedNote.ID = msg.note.ID
			m.EditingNoteID = msg.note.ID
		}
		return m, m.loadStatsCmd()
		
	case noteDeletedMsg:
		m.Screen = m.PrevScreen
		m.SelectedNote = nil
		m.Cursor = 0
		if m.PrevScreen == ScreenRecent {
			return m, m.loadRecentNotesCmd()
		} else if m.PrevScreen == ScreenSearch {
			return m, m.searchNotesCmd(m.SearchQuery)
		}
		return m, m.loadStatsCmd()

	case editorFinishedMsg:
		var cmd tea.Cmd
		if msg.content != "" && msg.content != m.SelectedNote.Content {
			if m.ViewingLocalFile {
				// Save to local file
				os.WriteFile(m.LocalFilePath, []byte(msg.content), 0644)
				m.SelectedNote.Content = msg.content
				
				// Sync to store
				if m.SelectedNote.ID != 0 {
					m.store.UpdateNote(m.SelectedNote.ID, m.SelectedNote.Title, msg.content)
				} else {
					cmd = m.saveNoteCmd(m.SelectedNote.Title, msg.content)
				}
			} else {
				// Update note content if changed
				m.store.UpdateNote(msg.id, m.SelectedNote.Title, msg.content)
				if note, ok := m.store.GetNote(msg.id); ok {
					m.SelectedNote = &note
				}
			}
		}
		m.Screen = ScreenNoteDetail
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKeyPress(key string) (tea.Model, tea.Cmd) {
	m.ErrorMsg = ""
	m.SuccessMsg = ""
	switch m.Screen {
	case ScreenDashboard:
		return m.handleDashboardKeys(key)
	case ScreenSearch:
		return m.handleSearchKeys(key)
	case ScreenRecent:
		return m.handleRecentKeys(key)
	case ScreenTags:
		return m.handleTagsKeys(key)
	case ScreenNoteDetail:
		return m.handleNoteDetailKeys(key)
	case ScreenEditorSelect:
		return m.handleEditorSelectKeys(key)
	case ScreenFileExplorer:
		return m.handleFileExplorerKeys(key)
	}
	return m, nil
}

// ─── Dashboard ───────────────────────────────────────────────────────────────

var dashboardMenuItems = []string{
	"Buscar notas",
	"Notas recientes",
	"Explorar etiquetas",
	"Exportar notas",
	"Explorar archivos",
	"Crear nota nueva",
	"Salir",
}

func (m Model) handleDashboardKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "down", "j":
		if m.Cursor < len(dashboardMenuItems)-1 {
			m.Cursor++
		}
	case "enter", " ":
		return m.handleDashboardSelection()
	case "s", "/":
		m.PrevScreen = ScreenDashboard
		m.Screen = ScreenSearch
		m.Cursor = 0
		m.SearchInput.SetValue("")
		m.SearchInput.Focus()
		return m, nil
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleDashboardSelection() (tea.Model, tea.Cmd) {
	switch m.Cursor {
	case 0:
		m.PrevScreen = ScreenDashboard
		m.Screen = ScreenSearch
		m.Cursor = 0
		m.Scroll = 0
		m.SearchInput.SetValue("")
		m.SearchInput.Focus()
		// Emit an empty search immediately to list everything
		return m, tea.Batch(textinput.Blink, m.searchNotesCmd(""))
	case 1:
		m.PrevScreen = ScreenDashboard
		m.Screen = ScreenRecent
		m.Cursor = 0
		m.Scroll = 0
		return m, m.loadRecentNotesCmd()
	case 2:
		m.PrevScreen = ScreenDashboard
		m.Screen = ScreenTags
		m.Cursor = 0
		m.Scroll = 0
		return m, m.loadTagsCmd()
	case 3:
		m.PrevScreen = ScreenDashboard
		m.Screen = ScreenExportSelect
		m.Cursor = 0
		m.Scroll = 0
		m.SearchInput.SetValue("")
		m.SearchInput.Focus()
		// Auto-select all by default when entering Export screen
		m.ExportSelected = make(map[int64]bool)
		return m, tea.Batch(textinput.Blink, m.searchNotesCmd(""))
	case 4:
		m.PrevScreen = ScreenDashboard
		m.Screen = ScreenFileExplorer
		m.FileCursor = 0
		m.FileScroll = 0
		
		if m.CurrentPath == "" {
			home, err := os.UserHomeDir()
			if err == nil {
				m.CurrentPath = home
			} else {
				m.CurrentPath = "."
			}
		}
		m.SearchInput.SetValue("")
		m.SearchInput.Focus()
		return m, tea.Batch(textinput.Blink, m.loadFilesCmd())
	case 5:
		m.PrevScreen = ScreenDashboard
		m.Screen = ScreenCreateNote
		m.EditingNoteID = 0
		m.EditingField = 0
		m.NoteTitleInput.SetValue("")
		m.NoteContent.SetValue("")
		m.NoteTitleInput.Focus()
		m.NoteContent.Blur()
		return m, textinput.Blink
	case 6:
		return m, tea.Quit
	}
	return m, nil
}

// ─── Create Note ─────────────────────────────────────────────────────────────

func (m Model) handleCreateNoteMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.String() {
	case "esc":
		// Auto-save on exit
		title := m.NoteTitleInput.Value()
		content := m.NoteContent.Value()
		if title != "" || content != "" {
			if m.ViewingLocalFile {
				os.WriteFile(m.LocalFilePath, []byte(content), 0644)
			}
			if m.EditingNoteID != 0 {
				cmds = append(cmds, m.updateNoteCmd(m.EditingNoteID, title, content))
			} else {
				cmds = append(cmds, m.saveNoteCmd(title, content))
			}
		}
		
		if m.SelectedNote != nil {
			m.SelectedNote.Title = title
			m.SelectedNote.Content = content
		}

		m.EditingNoteID = 0
		m.Screen = m.PrevScreen
		m.Cursor = 0
		m.NoteTitleInput.Blur()
		m.NoteContent.Blur()
		if m.Screen == ScreenDashboard {
			cmds = append(cmds, m.loadStatsCmd())
		}
		return m, tea.Batch(cmds...)

	case "tab":
		if m.EditingField == 0 {
			m.EditingField = 1
			m.NoteTitleInput.Blur()
			m.NoteContent.Focus()
			cmds = append(cmds, textarea.Blink)
		} else {
			m.EditingField = 0
			m.NoteContent.Blur()
			m.NoteTitleInput.Focus()
			cmds = append(cmds, textinput.Blink)
		}
		return m, tea.Batch(cmds...)
		
	case "enter":
		if m.EditingField == 0 {
			m.EditingField = 1
			m.NoteTitleInput.Blur()
			m.NoteContent.Focus()
			return m, textarea.Blink
		}
		// If in textarea, enter is just a newline, so it falls through to textarea.Update
	}

	var cmd tea.Cmd
	if m.EditingField == 0 {
		m.NoteTitleInput, cmd = m.NoteTitleInput.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		m.NoteContent, cmd = m.NoteContent.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// ─── Search Input ────────────────────────────────────────────────────────────

func (m Model) handleSearchInputKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg.String() {
	case "up":
		if m.Cursor > 0 {
			m.Cursor--
			if m.Cursor < m.Scroll {
				m.Scroll = m.Cursor
			}
		}
		return m, nil
	case "down":
		visibleItems := (m.Height - 12) / 2
		if visibleItems < 3 {
			visibleItems = 3
		}
		if m.Cursor < len(m.SearchResults)-1 {
			m.Cursor++
			if m.Cursor >= m.Scroll+visibleItems {
				m.Scroll = m.Cursor - visibleItems + 1
			}
		}
		return m, nil
	case "p":
		if len(m.SearchResults) > 0 && m.Cursor < len(m.SearchResults) {
			id := m.SearchResults[m.Cursor].ID
			m.PrevScreen = ScreenSearch
			return m, m.togglePinCmd(id)
		}
		return m, nil
	case "enter":
		if len(m.SearchResults) > 0 && m.Cursor < len(m.SearchResults) {
			id := m.SearchResults[m.Cursor].ID
			m.PrevScreen = ScreenSearch
			return m, m.loadNoteDetailCmd(id)
		}
		return m, nil
	case "esc":
		m.SearchInput.Blur()
		m.Screen = ScreenDashboard
		m.Cursor = 0
		return m, m.loadStatsCmd()
	}

	prevValue := m.SearchInput.Value()
	m.SearchInput, cmd = m.SearchInput.Update(msg)
	cmds = append(cmds, cmd)

	if m.SearchInput.Value() != prevValue {
		m.Cursor = 0
		m.Scroll = 0
		m.SearchQuery = m.SearchInput.Value()
		cmds = append(cmds, m.searchNotesCmd(m.SearchQuery))
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleSearchKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.Screen = ScreenDashboard
		m.Cursor = 0
		return m, m.loadStatsCmd()
	case "i", "/":
		m.SearchInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

// ─── Recent Notes ────────────────────────────────────────────────────────────

func (m Model) handleRecentKeys(key string) (tea.Model, tea.Cmd) {
	visibleItems := (m.Height - 8) / 2
	if visibleItems < 3 {
		visibleItems = 3
	}

	switch key {
	case "up", "k":
		if m.Cursor > 0 {
			m.Cursor--
			if m.Cursor < m.Scroll {
				m.Scroll = m.Cursor
			}
		}
	case "down", "j":
		if m.Cursor < len(m.RecentNotes)-1 {
			m.Cursor++
			if m.Cursor >= m.Scroll+visibleItems {
				m.Scroll = m.Cursor - visibleItems + 1
			}
		}
	case "p":
		if len(m.RecentNotes) > 0 && m.Cursor < len(m.RecentNotes) {
			id := m.RecentNotes[m.Cursor].ID
			m.PrevScreen = ScreenRecent
			return m, m.togglePinCmd(id)
		}
	case "enter":
		if len(m.RecentNotes) > 0 && m.Cursor < len(m.RecentNotes) {
			id := m.RecentNotes[m.Cursor].ID
			m.PrevScreen = ScreenRecent
			return m, m.loadNoteDetailCmd(id)
		}
	case "esc", "q":
		m.Screen = ScreenDashboard
		m.Cursor = 0
		m.Scroll = 0
		return m, m.loadStatsCmd()
	}
	return m, nil
}

// ─── Note Detail ─────────────────────────────────────────────────────────────

func (m Model) handleNoteDetailKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.DetailScroll > 0 {
			m.DetailScroll--
		}
	case "down", "j":
		m.DetailScroll++
	case "e":
		m.EditingNoteID = m.SelectedNote.ID
		m.Screen = ScreenCreateNote
		m.EditingField = 0
		m.NoteTitleInput.SetValue(m.SelectedNote.Title)
		m.NoteContent.SetValue(m.SelectedNote.Content)
		m.NoteTitleInput.Focus()
		m.NoteContent.Blur()
		return m, textinput.Blink
	case "o":
		m.AvailableEditors = detectEditors()
		m.EditorCursor = 0
		m.Screen = ScreenEditorSelect
		return m, nil
	case "p":
		if m.ViewingLocalFile { return m, nil }
		m.PrevScreen = m.Screen // Preserve current screen
		return m, m.togglePinCmd(m.SelectedNote.ID)
	case "x", "delete":
		if m.ViewingLocalFile { return m, nil }
		return m, m.deleteNoteCmd(m.SelectedNote.ID)
	case "esc", "q":
		m.Screen = m.PrevScreen
		m.Cursor = 0
		m.DetailScroll = 0
		m.ViewingLocalFile = false
		m.LocalFilePath = ""
		
		if m.PrevScreen == ScreenRecent {
			return m, m.loadRecentNotesCmd()
		} else if m.PrevScreen == ScreenSearch {
			return m, m.searchNotesCmd(m.SearchQuery)
		} else if m.PrevScreen == ScreenFileExplorer {
			return m, m.loadFilesCmd()
		}
		return m, m.loadStatsCmd()
	}
	return m, nil
}

// ─── External Editor ─────────────────────────────────────────────────────────

func detectEditors() []Editor {
	var available []Editor

	if env := os.Getenv("EDITOR"); env != "" {
		available = append(available, Editor{Name: "Sistema Default ($EDITOR)", Command: env})
	}

	candidates := []Editor{
		{Name: "Antigravity", Command: "antigravity"},
		{Name: "Kiro", Command: "kiro"},
		{Name: "VS Code", Command: "code", Args: []string{"--wait"}},
		{Name: "Cursor", Command: "cursor", Args: []string{"--wait"}},
		{Name: "Zed", Command: "zed", Args: []string{"--wait"}},
		{Name: "Sublime Text", Command: "subl", Args: []string{"--wait"}},
		{Name: "Neovim", Command: "nvim"},
		{Name: "Vim", Command: "vim"},
		{Name: "Helix", Command: "hx"},
		{Name: "Micro", Command: "micro"},
		{Name: "Emacs", Command: "emacs"},
		{Name: "Nano", Command: "nano"},
		{Name: "Notepad", Command: "notepad"},
	}

	for _, c := range candidates {
		if _, err := exec.LookPath(c.Command); err == nil {
			available = append(available, c)
		}
	}

	return available
}

func (m Model) handleEditorSelectKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.EditorCursor > 0 {
			m.EditorCursor--
		}
	case "down", "j":
		if m.EditorCursor < len(m.AvailableEditors)-1 {
			m.EditorCursor++
		}
	case "esc", "q":
		m.Screen = ScreenNoteDetail
		return m, nil
	case "enter":
		if len(m.AvailableEditors) == 0 {
			return m, nil
		}
		
		editor := m.AvailableEditors[m.EditorCursor]
		id := m.SelectedNote.ID
		content := m.SelectedNote.Content
		
		f, err := os.CreateTemp("", "nota-*.md")
		if err != nil {
			m.ErrorMsg = "No se pudo crear archivo temporal"
			return m, nil
		}
		
		var filePathToEdit string
		if m.ViewingLocalFile {
			filePathToEdit = m.LocalFilePath
		} else {
			f.WriteString(content)
			f.Close()
			filePathToEdit = f.Name()
		}
		
		args := append(editor.Args, filePathToEdit)
		c := exec.Command(editor.Command, args...)
		
		return m, tea.ExecProcess(c, func(err error) tea.Msg {
			if !m.ViewingLocalFile {
				if err != nil {
					os.Remove(filePathToEdit)
					return errMsg{err}
				}
				newContent, _ := os.ReadFile(filePathToEdit)
				os.Remove(filePathToEdit)
				return editorFinishedMsg{
					id:      id,
					content: string(newContent),
				}
			} else {
				if err != nil {
					return errMsg{err}
				}
				newContent, _ := os.ReadFile(filePathToEdit)
				return editorFinishedMsg{
					id:      0,
					content: string(newContent),
				}
			}
		})
	}
	return m, nil
}

// ─── Tags Explorer ───────────────────────────────────────────────────────────

func (m Model) handleTagsKeys(key string) (tea.Model, tea.Cmd) {
	visibleItems := (m.Height - 8) / 2
	if visibleItems < 3 {
		visibleItems = 3
	}

	switch key {
	case "up", "k":
		if m.Cursor > 0 {
			m.Cursor--
			if m.Cursor < m.Scroll {
				m.Scroll = m.Cursor
			}
		}
	case "down", "j":
		if m.Cursor < len(m.TagsList)-1 {
			m.Cursor++
			if m.Cursor >= m.Scroll+visibleItems {
				m.Scroll = m.Cursor - visibleItems + 1
			}
		}
	case "enter":
		if len(m.TagsList) > 0 && m.Cursor < len(m.TagsList) {
			tagName := m.TagsList[m.Cursor].Name
			m.PrevScreen = ScreenDashboard
			m.Screen = ScreenSearch
			m.Cursor = 0
			m.Scroll = 0
			query := "tag:" + tagName
			m.SearchInput.SetValue(query)
			m.SearchQuery = query
			m.SearchInput.Focus()
			return m, m.searchNotesCmd(query)
		}
	case "esc", "q":
		m.Screen = ScreenDashboard
		m.Cursor = 0
		m.Scroll = 0
		return m, m.loadStatsCmd()
	}
	return m, nil
}

// ─── Export Select ───────────────────────────────────────────────────────────

func (m Model) handleExportInputKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	visibleItems := (m.Height - 12) / 2
	if visibleItems < 3 {
		visibleItems = 3
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg.String() {
	case "up":
		if m.Cursor > 0 {
			m.Cursor--
			if m.Cursor < m.Scroll {
				m.Scroll = m.Cursor
			}
		}
		return m, nil
	case "down":
		if m.Cursor < len(m.SearchResults)-1 {
			m.Cursor++
			if m.Cursor >= m.Scroll+visibleItems {
				m.Scroll = m.Cursor - visibleItems + 1
			}
		}
		return m, nil
	}

	// Hotkeys for selection only work if input is empty or we use Ctrl+ keys.
	// But actually, we can check if it's a specific key.
	// If the user wants to search, they type. "a" will type "a".
	// So we need Ctrl+A to select all, Ctrl+Space or Shift+Space to toggle, or just use tab?
	// Let's use tab to toggle, ctrl+a to select all, enter to export.
	switch msg.String() {
	case "tab":
		if len(m.SearchResults) > 0 && m.Cursor < len(m.SearchResults) {
			id := m.SearchResults[m.Cursor].ID
			m.ExportSelected[id] = !m.ExportSelected[id]
		}
		return m, nil
	case "ctrl+a":
		allSelected := true
		for _, n := range m.SearchResults {
			if !m.ExportSelected[n.ID] {
				allSelected = false
				break
			}
		}
		for _, n := range m.SearchResults {
			m.ExportSelected[n.ID] = !allSelected
		}
		return m, nil
	case "enter":
		var idsToExport []int64
		for id, selected := range m.ExportSelected {
			if selected {
				idsToExport = append(idsToExport, id)
			}
		}
		
		if len(idsToExport) == 0 {
			m.ErrorMsg = "No has seleccionado ninguna nota para exportar"
			return m, nil
		}
		
		count, err := m.store.ExportNotes(idsToExport, "./export")
		if err != nil {
			m.ErrorMsg = "Error exportando: " + err.Error()
			return m, nil
		}
		
		m.SuccessMsg = fmt.Sprintf("¡%d notas exportadas a ./export!", count)
		m.Screen = ScreenDashboard
		m.Cursor = 0
		return m, m.loadStatsCmd()
		
	case "esc":
		m.SearchInput.Blur()
		m.Screen = ScreenDashboard
		m.Cursor = 0
		return m, m.loadStatsCmd()
	}

	prevValue := m.SearchInput.Value()
	m.SearchInput, cmd = m.SearchInput.Update(msg)
	cmds = append(cmds, cmd)

	if m.SearchInput.Value() != prevValue {
		m.Cursor = 0
		m.Scroll = 0
		m.SearchQuery = m.SearchInput.Value()
		cmds = append(cmds, m.searchNotesCmd(m.SearchQuery))
	}

	return m, tea.Batch(cmds...)
}

// ─── File Explorer ───────────────────────────────────────────────────────────

func (m Model) handleFileExplorerInputKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg.String() {
	case "backspace":
		if m.SearchInput.Value() == "" {
			parent := filepath.Dir(m.CurrentPath)
			if parent != m.CurrentPath {
				m.CurrentPath = parent
				m.FileCursor = 0
				m.FileScroll = 0
				return m, m.loadFilesCmd()
			}
			return m, nil
		}
	case "up":
		if m.FileCursor > 0 {
			m.FileCursor--
			if m.FileCursor < m.FileScroll {
				m.FileScroll = m.FileCursor
			}
		}
		return m, nil
	case "down":
		visibleItems := (m.Height - 10) / 2
		if visibleItems < 3 {
			visibleItems = 3
		}
		if m.FileCursor < len(m.Files)-1 {
			m.FileCursor++
			if m.FileCursor >= m.FileScroll+visibleItems {
				m.FileScroll = m.FileCursor - visibleItems + 1
			}
		}
		return m, nil
	case "enter":
		// Check if input is a valid absolute or relative path
		input := m.SearchInput.Value()
		if input != "" && (strings.Contains(input, string(os.PathSeparator)) || strings.Contains(input, "/")) {
			info, err := os.Stat(input)
			if err == nil && info.IsDir() {
				m.CurrentPath = input
				m.SearchInput.SetValue("")
				m.FileCursor = 0
				m.FileScroll = 0
				return m, m.loadFilesCmd()
			}
		}
		
		if len(m.Files) == 0 {
			return m, nil
		}
		
		selected := m.Files[m.FileCursor]
		fullPath := filepath.Join(m.CurrentPath, selected.Name())
		
		if selected.IsDir() {
			m.CurrentPath = fullPath
			m.SearchInput.SetValue("")
			m.FileCursor = 0
			m.FileScroll = 0
			return m, m.loadFilesCmd()
		}
		
		// It's a file
		ext := strings.ToLower(filepath.Ext(selected.Name()))
		if ext == ".txt" || ext == ".md" || ext == ".json" || ext == ".go" || ext == ".html" || ext == ".css" || ext == ".js" {
			content, err := os.ReadFile(fullPath)
			if err != nil {
				m.ErrorMsg = "Error leyendo: " + err.Error()
				return m, nil
			}
			
			dummy := store.Note{
				ID:        0, 
				Title:     selected.Name(),
				Content:   string(content),
				Status:    "local",
				CreatedAt: "Ahora",
			}
			
			m.SelectedNote = &dummy
			m.ViewingLocalFile = true
			m.LocalFilePath = fullPath
			m.PrevScreen = m.Screen
			m.Screen = ScreenNoteDetail
			m.DetailScroll = 0
			return m, nil
		} else {
			m.ViewingLocalFile = true
			m.LocalFilePath = fullPath
			m.SelectedNote = &store.Note{ID: 0, Title: selected.Name(), Content: ""} 
			
			m.AvailableEditors = detectEditors()
			m.EditorCursor = 0
			m.PrevScreen = m.Screen
			m.Screen = ScreenEditorSelect
			return m, nil
		}
	case "esc":
		m.SearchInput.Blur()
		m.Screen = ScreenDashboard
		m.FileCursor = 0
		m.FileScroll = 0
		return m, m.loadStatsCmd()
	}

	prevValue := m.SearchInput.Value()
	m.SearchInput, cmd = m.SearchInput.Update(msg)
	cmds = append(cmds, cmd)

	if m.SearchInput.Value() != prevValue {
		m.FileCursor = 0
		m.FileScroll = 0
		cmds = append(cmds, m.loadFilesCmd())
	}

	return m, tea.Batch(cmds...)
}

func (m Model) loadFilesCmd() tea.Cmd {
	return func() tea.Msg {
		entries, err := os.ReadDir(m.CurrentPath)
		if err != nil {
			return errMsg{err}
		}
		
		query := strings.ToLower(m.SearchInput.Value())
		
		var dirs []os.DirEntry
		var files []os.DirEntry
		for _, e := range entries {
			// Skip if it doesn't match query (unless query is a path separator)
			if query != "" && !strings.Contains(query, string(os.PathSeparator)) && !strings.Contains(query, "/") {
				if !strings.Contains(strings.ToLower(e.Name()), query) {
					continue
				}
			}
			
			if e.IsDir() {
				dirs = append(dirs, e)
			} else {
				files = append(files, e)
			}
		}
		
		return filesLoadedMsg{files: append(dirs, files...)}
	}
}

func (m Model) handleFileExplorerKeys(key string) (tea.Model, tea.Cmd) {
	visibleItems := (m.Height - 8) / 2
	if visibleItems < 3 {
		visibleItems = 3
	}

	switch key {
	case "up", "k":
		if m.FileCursor > 0 {
			m.FileCursor--
			if m.FileCursor < m.FileScroll {
				m.FileScroll = m.FileCursor
			}
		}
	case "down", "j":
		if m.FileCursor < len(m.Files)-1 {
			m.FileCursor++
			if m.FileCursor >= m.FileScroll+visibleItems {
				m.FileScroll = m.FileCursor - visibleItems + 1
			}
		}
	case "backspace", "h", "left":
		m.CurrentPath = filepath.Dir(m.CurrentPath)
		m.FileCursor = 0
		m.FileScroll = 0
		return m, m.loadFilesCmd()
	case "enter", "l", "right":
		if len(m.Files) == 0 {
			return m, nil
		}
		selected := m.Files[m.FileCursor]
		fullPath := filepath.Join(m.CurrentPath, selected.Name())
		
		if selected.IsDir() {
			m.CurrentPath = fullPath
			m.FileCursor = 0
			m.FileScroll = 0
			return m, m.loadFilesCmd()
		}
		
		// It's a file
		ext := strings.ToLower(filepath.Ext(selected.Name()))
		if ext == ".txt" || ext == ".md" || ext == ".json" || ext == ".go" || ext == ".html" || ext == ".css" || ext == ".js" {
			content, err := os.ReadFile(fullPath)
			if err != nil {
				m.ErrorMsg = "Error leyendo: " + err.Error()
				return m, nil
			}
			
			// Create dummy note
			dummy := store.Note{
				ID:        0, // 0 signifies it's local
				Title:     selected.Name(),
				Content:   string(content),
				Status:    "local",
				CreatedAt: "Ahora",
			}
			
			m.SelectedNote = &dummy
			m.ViewingLocalFile = true
			m.LocalFilePath = fullPath
			m.PrevScreen = m.Screen
			m.Screen = ScreenNoteDetail
			m.DetailScroll = 0
			return m, nil
		} else {
			// Not supported natively, open in external editor
			m.ViewingLocalFile = true
			m.LocalFilePath = fullPath
			m.SelectedNote = &store.Note{ID: 0, Title: selected.Name(), Content: ""} // Minimal dummy
			
			m.AvailableEditors = detectEditors()
			m.EditorCursor = 0
			m.PrevScreen = m.Screen
			m.Screen = ScreenEditorSelect
			return m, nil
		}
		
	case "esc", "q":
		m.Screen = ScreenDashboard
		m.FileCursor = 0
		m.FileScroll = 0
		return m, nil
	}
	
	return m, nil
}

package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/glamour"
)

// ─── Logo ────────────────────────────────────────────────────────────────────

func renderLogo() string {
	logoText := []string{
		`███    ██  ██████  ████████  █████  ███████ `,
		`████   ██ ██    ██    ██    ██   ██ ██      `,
		`██ ██  ██ ██    ██    ██    ███████ ███████ `,
		`██  ██ ██ ██    ██    ██    ██   ██      ██ `,
		`██   ████  ██████     ██    ██   ██ ███████ `,
	}

	frameStyle := lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(colorOverlay).Padding(0, 1)
	colors := []lipgloss.Color{colorMauve, colorLavender, colorBlue, colorTeal, colorGreen}

	var b strings.Builder

	for i, line := range logoText {
		b.WriteString(" " + lipgloss.NewStyle().Foreground(colors[i]).Bold(true).Render(line) + "\n")
	}
	
	b.WriteString("\n" + lipgloss.NewStyle().Foreground(colorSubtext).Italic(true).Render("  > Un simple blog de notas") + "\n")

	return frameStyle.Render(b.String())
}

// ─── View (main router) ─────────────────────────────────────────────────────

func (m Model) View() string {
	var content string

	switch m.Screen {
	case ScreenDashboard:
		content = m.viewDashboard()
	case ScreenSearch:
		content = m.viewSearch()
	case ScreenRecent:
		content = m.viewRecent()
	case ScreenTags:
		content = m.viewTags()
	case ScreenNoteDetail:
		content = m.viewNoteDetail()
	case ScreenCreateNote:
		content = m.viewCreateNote()
	case ScreenEditorSelect:
		content = m.viewEditorSelect()
	case ScreenExportSelect:
		content = m.viewExportSelect()
	case ScreenFileExplorer:
		content = m.viewFileExplorer()
	default:
		content = "Unknown screen"
	}

	if m.ErrorMsg != "" {
		content += "\n" + errorStyle.Render("Error: "+m.ErrorMsg)
	}
	if m.SuccessMsg != "" {
		content += "\n" + lipgloss.NewStyle().Foreground(colorGreen).Render("✔ "+m.SuccessMsg)
	}

	return appStyle.Render(content)
}

func (m Model) viewDashboard() string {
	var b strings.Builder

	b.WriteString(renderLogo())
	b.WriteString("\n")

	statsContent := fmt.Sprintf("%s %s", statNumberStyle.Render(fmt.Sprintf("%d", m.Stats.TotalNotes)), statLabelStyle.Render("notas creadas"))
	b.WriteString(statCardStyle.Render(statsContent))
	b.WriteString("\n")

	b.WriteString(titleStyle.Render("  Menu"))
	b.WriteString("\n")

	for i, item := range dashboardMenuItems {
		if i == m.Cursor {
			b.WriteString(menuSelectedStyle.Render("▸ " + item))
		} else {
			b.WriteString(menuItemStyle.Render("  " + item))
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("\n  j/k navigate • enter select • s search • q quit"))
	return b.String()
}

func (m Model) viewSearch() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("  Búsqueda Dinámica"))
	b.WriteString("\n\n")
	b.WriteString(searchInputStyle.Render(m.SearchInput.View()))
	b.WriteString("\n\n")

	count := len(m.SearchResults)
	if count == 0 {
		if m.SearchQuery != "" {
			b.WriteString(noResultsStyle.Render("  No se encontraron notas."))
		} else {
			b.WriteString(helpStyle.Render("  Empieza a escribir para filtrar las notas..."))
		}
	} else {
		visibleItems := (m.Height - 12) / 2
		if visibleItems < 3 {
			visibleItems = 3
		}

		end := m.Scroll + visibleItems
		if end > count {
			end = count
		}

		for i := m.Scroll; i < end; i++ {
			r := m.SearchResults[i]
			b.WriteString(m.renderNoteListItem(i, r.ID, r.Status, r.Title, r.Content, r.Tags, r.CreatedAt, r.Pinned))
		}

		if count > visibleItems {
			b.WriteString(fmt.Sprintf("\n  %s", timestampStyle.Render(fmt.Sprintf("mostrando %d-%d de %d", m.Scroll+1, end, count))))
		}
	}

	b.WriteString(helpStyle.Render("\n  Flechas ↑/↓ navegar • enter detalle • esc salir"))
	return b.String()
}

func (m Model) viewExportSelect() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("  Exportar Notas"))
	b.WriteString("\n\n")
	b.WriteString(searchInputStyle.Render(m.SearchInput.View()))
	b.WriteString("\n\n")

	count := len(m.SearchResults)
	if count == 0 {
		if m.SearchQuery != "" {
			b.WriteString(noResultsStyle.Render("  No se encontraron notas."))
		} else {
			b.WriteString(helpStyle.Render("  Empieza a escribir para filtrar las notas..."))
		}
	} else {
		visibleItems := (m.Height - 12) / 2
		if visibleItems < 3 {
			visibleItems = 3
		}

		end := m.Scroll + visibleItems
		if end > count {
			end = count
		}

		for i := m.Scroll; i < end; i++ {
			r := m.SearchResults[i]
			
			// Custom render item with checkbox
			cursor := "  "
			style := listItemStyle
			if i == m.Cursor {
				cursor = "▸ "
				style = listSelectedStyle
			}

			checkbox := "[ ]"
			if m.ExportSelected[r.ID] {
				checkbox = lipgloss.NewStyle().Foreground(colorGreen).Render("[✓]")
			}

			var formattedTags []string
			for _, t := range r.Tags {
				formattedTags = append(formattedTags, "#"+t)
			}
			tagStr := ""
			if len(formattedTags) > 0 {
				tagStr = " " + lipgloss.NewStyle().Foreground(colorTeal).Render(strings.Join(formattedTags, " "))
			}

			line := fmt.Sprintf("%s%s %s %s %s%s\n",
				cursor,
				checkbox,
				idStyle.Render(fmt.Sprintf("#%-5d", r.ID)),
				style.Render(truncateStr(r.Title, 50)),
				tagStr,
				timestampStyle.Render(localTime(r.CreatedAt)))

			b.WriteString(line)
		}

		if count > visibleItems {
			b.WriteString(fmt.Sprintf("\n  %s", timestampStyle.Render(fmt.Sprintf("mostrando %d-%d de %d", m.Scroll+1, end, count))))
		}
	}

	b.WriteString(helpStyle.Render("\n  ↑/↓ mover • tab seleccionar • ctrl+a todo • enter exportar • esc salir"))
	return b.String()
}

func (m Model) viewRecent() string {
	var b strings.Builder

	count := len(m.RecentNotes)
	b.WriteString(headerStyle.Render(fmt.Sprintf("  Notas Recientes — %d total", count)))
	b.WriteString("\n")

	if count == 0 {
		b.WriteString(noResultsStyle.Render("No hay notas recientes."))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("  esc back"))
		return b.String()
	}

	visibleItems := (m.Height - 8) / 2
	if visibleItems < 3 {
		visibleItems = 3
	}

	end := m.Scroll + visibleItems
	if end > count {
		end = count
	}

	for i := m.Scroll; i < end; i++ {
		o := m.RecentNotes[i]
		b.WriteString(m.renderNoteListItem(i, o.ID, o.Status, o.Title, o.Content, o.Tags, o.CreatedAt, o.Pinned))
	}

	if count > visibleItems {
		b.WriteString(fmt.Sprintf("\n  %s", timestampStyle.Render(fmt.Sprintf("mostrando %d-%d de %d", m.Scroll+1, end, count))))
	}

	b.WriteString(helpStyle.Render("\n  j/k navigate • enter detail • esc back"))
	return b.String()
}

func (m Model) viewNoteDetail() string {
	var b strings.Builder

	if m.SelectedNote == nil {
		b.WriteString(headerStyle.Render("  Detalle de Nota"))
		b.WriteString("\n")
		b.WriteString(noResultsStyle.Render("Cargando..."))
		return b.String()
	}

	note := m.SelectedNote
	b.WriteString(headerStyle.Render("  Nota #" + fmt.Sprintf("%d", note.ID)))
	if note.Pinned {
		b.WriteString(lipgloss.NewStyle().Foreground(colorYellow).Render(" 📌 FIJADA"))
	}
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("%s %s\n", detailLabelStyle.Render("Estado:"), typeBadgeStyle.Render(note.Status)))
	b.WriteString(fmt.Sprintf("%s %s\n", detailLabelStyle.Render("Título:"), detailValueStyle.Bold(true).Render(note.Title)))
	b.WriteString(fmt.Sprintf("%s %s\n", detailLabelStyle.Render("Creado:"), timestampStyle.Render(localTime(note.CreatedAt))))
	if len(note.Tags) > 0 {
		var formatted []string
		for _, t := range note.Tags {
			formatted = append(formatted, "#"+t)
		}
		b.WriteString(fmt.Sprintf("%s %s\n", detailLabelStyle.Render("Tags:"), lipgloss.NewStyle().Foreground(colorTeal).Render(strings.Join(formatted, " "))))
	}

	b.WriteString("\n")
	b.WriteString(sectionHeadingStyle.Render("  Contenido"))
	b.WriteString("\n")

	wrapWidth := m.Width - 6
	if wrapWidth < 20 {
		wrapWidth = 20
	}
	
	var renderedContent string
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("catppuccin-mocha"),
		glamour.WithWordWrap(wrapWidth),
	)
	if err != nil {
		// Fallback to generic dark if catppuccin-mocha is not available in this glamour version
		r, _ = glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(wrapWidth),
		)
	}
	
	if r != nil {
		out, err := r.Render(note.Content)
		if err == nil {
			renderedContent = strings.TrimSpace(out)
		} else {
			renderedContent = note.Content
		}
	} else {
		renderedContent = note.Content
	}

	// Apply left padding to match the design
	renderedContent = lipgloss.NewStyle().PaddingLeft(2).Render(renderedContent)

	contentLines := strings.Split(renderedContent, "\n")
	maxLines := m.Height - 16
	if maxLines < 5 {
		maxLines = 5
	}

	maxScroll := len(contentLines) - maxLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.DetailScroll > maxScroll {
		m.DetailScroll = maxScroll
	}

	end := m.DetailScroll + maxLines
	if end > len(contentLines) {
		end = len(contentLines)
	}

	for i := m.DetailScroll; i < end; i++ {
		b.WriteString(contentLines[i])
		b.WriteString("\n")
	}

	if len(contentLines) > maxLines {
		b.WriteString(fmt.Sprintf("\n  %s", timestampStyle.Render(fmt.Sprintf("linea %d-%d de %d", m.DetailScroll+1, end, len(contentLines)))))
	}

	b.WriteString("\n  ")
	b.WriteString(helpStyle.Render("j/k scroll • e editar interno • o abrir en editor • x borrar • p fijar/desfijar • esc back"))

	return b.String()
}

func (m Model) viewTags() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("  Explorador de Etiquetas"))
	b.WriteString("\n\n")

	if len(m.TagsList) == 0 {
		b.WriteString(noResultsStyle.Render("  No hay etiquetas guardadas."))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("  esc volver"))
		return b.String()
	}

	visibleItems := (m.Height - 8) / 2
	if visibleItems < 3 {
		visibleItems = 3
	}

	start := m.Scroll
	end := m.Scroll + visibleItems
	if end > len(m.TagsList) {
		end = len(m.TagsList)
	}

	for i := start; i < end; i++ {
		t := m.TagsList[i]
		
		cursor := "  "
		style := listItemStyle
		if i == m.Cursor {
			cursor = "> "
			style = listSelectedStyle
		}

		name := lipgloss.NewStyle().Foreground(colorTeal).Bold(i == m.Cursor).Render("#" + t.Name)
		count := timestampStyle.Render(fmt.Sprintf("(%d notas)", t.Count))

		line := fmt.Sprintf("%s%s %s\n", cursor, style.Render(name), count)
		b.WriteString(line)
	}

	if len(m.TagsList) > visibleItems {
		b.WriteString(fmt.Sprintf("\n  %s\n", timestampStyle.Render(fmt.Sprintf("%d-%d de %d", start+1, end, len(m.TagsList)))))
	} else {
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("  ↑/↓ mover • enter filtrar • esc volver"))

	return b.String()
}

func (m Model) viewEditorSelect() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("  Abrir en Editor Externo"))
	b.WriteString("\n\n")

	if len(m.AvailableEditors) == 0 {
		b.WriteString(noResultsStyle.Render("  No se encontraron editores instalados."))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("  esc volver"))
		return b.String()
	}

	b.WriteString(helpStyle.Render("  Selecciona tu editor preferido:\n\n"))

	for i, ed := range m.AvailableEditors {
		cursor := "  "
		style := listItemStyle
		if i == m.EditorCursor {
			cursor = "> "
			style = listSelectedStyle
		}
		
		name := fmt.Sprintf("%-20s", ed.Name)
		cmd := fmt.Sprintf("[%s]", ed.Command)
		
		line := fmt.Sprintf("%s%s %s\n", cursor, style.Render(name), typeBadgeStyle.Render(cmd))
		b.WriteString(line)
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  ↑/↓ mover • enter abrir • esc cancelar"))
	
	return b.String()
}

func (m Model) viewCreateNote() string {
	var b strings.Builder
	title := "  Crear Nueva Nota"
	if m.EditingNoteID != 0 {
		title = fmt.Sprintf("  Editando Nota #%d", m.EditingNoteID)
	}
	b.WriteString(headerStyle.Render(title))
	b.WriteString("\n\n")

	// Render Title Input
	titleLabel := detailLabelStyle.Render("  Título:")
	if m.EditingField == 0 {
		titleLabel = lipgloss.NewStyle().Foreground(colorLavender).Bold(true).Render("▶ Título:")
	}
	b.WriteString(titleLabel)
	b.WriteString("\n")
	b.WriteString(m.NoteTitleInput.View())
	b.WriteString("\n\n")

	// Render Content Textarea
	contentLabel := detailLabelStyle.Render("  Contenido:")
	if m.EditingField == 1 {
		contentLabel = lipgloss.NewStyle().Foreground(colorLavender).Bold(true).Render("▶ Contenido:")
	}
	b.WriteString(contentLabel)
	b.WriteString("\n")
	b.WriteString(m.NoteContent.View())
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("  tab/enter cambiar campo • esc salir y auto-guardar"))
	return b.String()
}

func (m Model) renderNoteListItem(index int, id int64, status, title, content string, tags []string, createdAt string, pinned bool) string {
	cursor := "  "
	style := listItemStyle
	if index == m.Cursor {
		cursor = "▸ "
		style = listSelectedStyle
	}

	var formattedTags []string
	for _, t := range tags {
		formattedTags = append(formattedTags, "#"+t)
	}
	tagStr := ""
	if len(formattedTags) > 0 {
		tagStr = " " + lipgloss.NewStyle().Foreground(colorTeal).Render(strings.Join(formattedTags, " "))
	}

	pinStr := ""
	if pinned {
		pinStr = lipgloss.NewStyle().Foreground(colorYellow).Render(" 📌")
	}

	line := fmt.Sprintf("%s%s %s %s%s%s  %s\n",
		cursor,
		idStyle.Render(fmt.Sprintf("#%-5d", id)),
		typeBadgeStyle.Render(fmt.Sprintf("[%-8s]", status)),
		style.Render(truncateStr(title, 50)),
		pinStr,
		tagStr,
		timestampStyle.Render(localTime(createdAt)))

	preview := truncateStr(content, 80)
	if preview != "" {
		line += contentPreviewStyle.Render(preview) + "\n"
	}

	return line
}

func localTime(utc string) string {
	if t, err := time.Parse(time.RFC3339, utc); err == nil {
		return t.Local().Format("2006-01-02 15:04:05")
	}
	return utc
}

func truncateStr(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

// ─── File Explorer ───────────────────────────────────────────────────────────

func (m Model) viewFileExplorer() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("  Explorador de Archivos"))
	b.WriteString("\n\n")
	b.WriteString(detailLabelStyle.Render("  Directorio actual: ") + detailValueStyle.Render(m.CurrentPath))
	b.WriteString("\n\n")
	b.WriteString(m.SearchInput.View())
	b.WriteString("\n\n")

	count := len(m.Files)
	if count == 0 {
		b.WriteString(noResultsStyle.Render("  (Directorio vacío)"))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("  backspace volver • esc salir"))
		return b.String()
	}

	visibleItems := (m.Height - 8) / 2
	if visibleItems < 3 {
		visibleItems = 3
	}

	start := m.FileScroll
	end := m.FileScroll + visibleItems
	if end > count {
		end = count
	}

	for i := start; i < end; i++ {
		entry := m.Files[i]
		
		cursor := "  "
		style := listItemStyle
		if i == m.FileCursor {
			cursor = "> "
			style = listSelectedStyle
		}

		icon := "📄"
		if entry.IsDir() {
			icon = "📁"
			style = style.Copy().Foreground(colorBlue).Bold(true)
		}

		name := style.Render(icon + " " + truncateStr(entry.Name(), 50))
		line := fmt.Sprintf("%s%s\n", cursor, name)
		b.WriteString(line)
	}

	if count > visibleItems {
		b.WriteString(fmt.Sprintf("\n  %s\n", timestampStyle.Render(fmt.Sprintf("%d-%d de %d", start+1, end, count))))
	} else {
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("  ↑/↓ mover • enter abrir • backspace carpeta superior • esc salir"))

	return b.String()
}

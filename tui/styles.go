package tui

import (
	"notas/config"

	"github.com/charmbracelet/lipgloss"
)

// ─── Dynamic Colors (set from config palette) ───────────────────────────────
var (
	colorBase     lipgloss.Color
	colorSurface  lipgloss.Color
	colorOverlay  lipgloss.Color
	colorText     lipgloss.Color
	colorSubtext  lipgloss.Color
	colorLavender lipgloss.Color
	colorGreen    lipgloss.Color
	colorPeach    lipgloss.Color
	colorRed      lipgloss.Color
	colorBlue     lipgloss.Color
	colorMauve    lipgloss.Color
	colorYellow   lipgloss.Color
	colorTeal     lipgloss.Color
)

// ─── Layout Styles ───────────────────────────────────────────────────────────
var (
	appStyle    lipgloss.Style
	headerStyle lipgloss.Style
	helpStyle   lipgloss.Style
	errorStyle  lipgloss.Style
)

// ─── Dashboard Styles ────────────────────────────────────────────────────────
var (
	statNumberStyle   lipgloss.Style
	statLabelStyle    lipgloss.Style
	statCardStyle     lipgloss.Style
	menuItemStyle     lipgloss.Style
	menuSelectedStyle lipgloss.Style
	titleStyle        lipgloss.Style
)

// ─── List Styles ─────────────────────────────────────────────────────────────
var (
	listItemStyle       lipgloss.Style
	listSelectedStyle   lipgloss.Style
	typeBadgeStyle      lipgloss.Style
	idStyle             lipgloss.Style
	timestampStyle      lipgloss.Style
	contentPreviewStyle lipgloss.Style
)

// ─── Detail View Styles ──────────────────────────────────────────────────────
var (
	sectionHeadingStyle lipgloss.Style
	detailContentStyle  lipgloss.Style
	detailLabelStyle    lipgloss.Style
	detailValueStyle    lipgloss.Style
)

// ─── Search Styles ───────────────────────────────────────────────────────────
var (
	searchInputStyle lipgloss.Style
	noResultsStyle   lipgloss.Style
)

// ApplyTheme rebuilds all styles from the given palette.
func ApplyTheme(p config.Palette) {
	// Set color variables
	colorBase = p.Base
	colorSurface = p.Surface
	colorOverlay = p.Overlay
	colorText = p.Text
	colorSubtext = p.Subtext
	colorLavender = p.Lavender
	colorGreen = p.Green
	colorPeach = p.Peach
	colorRed = p.Red
	colorBlue = p.Blue
	colorMauve = p.Mauve
	colorYellow = p.Yellow
	colorTeal = p.Teal

	// Rebuild styles
	appStyle = lipgloss.NewStyle().Foreground(colorText).Padding(1, 2)
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(colorLavender).BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).BorderForeground(colorOverlay).PaddingBottom(1).MarginBottom(1)
	helpStyle = lipgloss.NewStyle().Foreground(colorSubtext).MarginTop(1)
	errorStyle = lipgloss.NewStyle().Foreground(colorRed).Bold(true).Padding(0, 1)

	statNumberStyle = lipgloss.NewStyle().Bold(true).Foreground(colorGreen).Width(8).Align(lipgloss.Right)
	statLabelStyle = lipgloss.NewStyle().Foreground(colorText).PaddingLeft(2)
	statCardStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(colorOverlay).Padding(0, 2)
	menuItemStyle = lipgloss.NewStyle().Foreground(colorText).PaddingLeft(2)
	menuSelectedStyle = lipgloss.NewStyle().Foreground(colorLavender).Bold(true).PaddingLeft(1)
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorMauve).MarginBottom(1)

	listItemStyle = lipgloss.NewStyle().Foreground(colorText).PaddingLeft(2)
	listSelectedStyle = lipgloss.NewStyle().Foreground(colorLavender).Bold(true).PaddingLeft(1)
	typeBadgeStyle = lipgloss.NewStyle().Foreground(colorPeach).Bold(true)
	idStyle = lipgloss.NewStyle().Foreground(colorBlue)
	timestampStyle = lipgloss.NewStyle().Foreground(colorSubtext).Italic(true)
	contentPreviewStyle = lipgloss.NewStyle().Foreground(colorSubtext).PaddingLeft(4)

	sectionHeadingStyle = lipgloss.NewStyle().Bold(true).Foreground(colorMauve).MarginTop(1).MarginBottom(1)
	detailContentStyle = lipgloss.NewStyle().Foreground(colorText).PaddingLeft(2)
	detailLabelStyle = lipgloss.NewStyle().Foreground(colorSubtext).Width(14).Align(lipgloss.Right).PaddingRight(1)
	detailValueStyle = lipgloss.NewStyle().Foreground(colorText)

	searchInputStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(colorLavender).Foreground(colorText).Padding(0, 1).MarginBottom(1)
	noResultsStyle = lipgloss.NewStyle().Foreground(colorSubtext).Italic(true).PaddingLeft(2).MarginTop(1)
}

func init() {
	// Initialize with default theme so styles are never nil
	ApplyTheme(config.GetPalette("catppuccin"))
}

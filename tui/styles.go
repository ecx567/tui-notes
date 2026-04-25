package tui

import "github.com/charmbracelet/lipgloss"

// ─── Colors (Catppuccin Mocha palette) ─────────────────────────────────────────
var (
	colorBase     = lipgloss.Color("#1e1e2e")
	colorSurface  = lipgloss.Color("#313244")
	colorOverlay  = lipgloss.Color("#585b70")
	colorText     = lipgloss.Color("#cdd6f4")
	colorSubtext  = lipgloss.Color("#a6adc8")
	colorLavender = lipgloss.Color("#b4befe")
	colorGreen    = lipgloss.Color("#a6e3a1")
	colorPeach    = lipgloss.Color("#fab387")
	colorRed      = lipgloss.Color("#f38ba8")
	colorBlue     = lipgloss.Color("#89b4fa")
	colorMauve    = lipgloss.Color("#cba6f7")
	colorYellow   = lipgloss.Color("#f9e2af")
	colorTeal     = lipgloss.Color("#94e2d5")
)

// ─── Layout Styles ───────────────────────────────────────────────────────────
var (
	appStyle    = lipgloss.NewStyle().Foreground(colorText).Padding(1, 2)
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(colorLavender).BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).BorderForeground(colorOverlay).PaddingBottom(1).MarginBottom(1)
	helpStyle   = lipgloss.NewStyle().Foreground(colorSubtext).MarginTop(1)
	errorStyle  = lipgloss.NewStyle().Foreground(colorRed).Bold(true).Padding(0, 1)
)

// ─── Dashboard Styles ────────────────────────────────────────────────────────
var (
	statNumberStyle   = lipgloss.NewStyle().Bold(true).Foreground(colorGreen).Width(8).Align(lipgloss.Right)
	statLabelStyle    = lipgloss.NewStyle().Foreground(colorText).PaddingLeft(2)
	statCardStyle     = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(colorOverlay).Padding(0, 2)
	menuItemStyle     = lipgloss.NewStyle().Foreground(colorText).PaddingLeft(2)
	menuSelectedStyle = lipgloss.NewStyle().Foreground(colorLavender).Bold(true).PaddingLeft(1)
	titleStyle        = lipgloss.NewStyle().Bold(true).Foreground(colorMauve).MarginBottom(1)
)

// ─── List Styles ─────────────────────────────────────────────────────────────
var (
	listItemStyle       = lipgloss.NewStyle().Foreground(colorText).PaddingLeft(2)
	listSelectedStyle   = lipgloss.NewStyle().Foreground(colorLavender).Bold(true).PaddingLeft(1)
	typeBadgeStyle      = lipgloss.NewStyle().Foreground(colorPeach).Bold(true)
	idStyle             = lipgloss.NewStyle().Foreground(colorBlue)
	timestampStyle      = lipgloss.NewStyle().Foreground(colorSubtext).Italic(true)
	contentPreviewStyle = lipgloss.NewStyle().Foreground(colorSubtext).PaddingLeft(4)
)

// ─── Detail View Styles ──────────────────────────────────────────────────────
var (
	sectionHeadingStyle = lipgloss.NewStyle().Bold(true).Foreground(colorMauve).MarginTop(1).MarginBottom(1)
	detailContentStyle  = lipgloss.NewStyle().Foreground(colorText).PaddingLeft(2)
	detailLabelStyle    = lipgloss.NewStyle().Foreground(colorSubtext).Width(14).Align(lipgloss.Right).PaddingRight(1)
	detailValueStyle    = lipgloss.NewStyle().Foreground(colorText)
)

// ─── Search Styles ───────────────────────────────────────────────────────────
var (
	searchInputStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(colorLavender).Foreground(colorText).Padding(0, 1).MarginBottom(1)
	noResultsStyle   = lipgloss.NewStyle().Foreground(colorSubtext).Italic(true).PaddingLeft(2).MarginTop(1)
)

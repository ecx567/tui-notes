package config

import "github.com/charmbracelet/lipgloss"

// Palette holds all colors used by the TUI.
type Palette struct {
	Base     lipgloss.Color
	Surface  lipgloss.Color
	Overlay  lipgloss.Color
	Text     lipgloss.Color
	Subtext  lipgloss.Color
	Lavender lipgloss.Color
	Green    lipgloss.Color
	Peach    lipgloss.Color
	Red      lipgloss.Color
	Blue     lipgloss.Color
	Mauve    lipgloss.Color
	Yellow   lipgloss.Color
	Teal     lipgloss.Color
}

// GetPalette returns the color palette for the given theme name.
func GetPalette(theme string) Palette {
	switch theme {
	case "dracula":
		return Palette{
			Base:     lipgloss.Color("#282a36"),
			Surface:  lipgloss.Color("#44475a"),
			Overlay:  lipgloss.Color("#6272a4"),
			Text:     lipgloss.Color("#f8f8f2"),
			Subtext:  lipgloss.Color("#bfbfbf"),
			Lavender: lipgloss.Color("#bd93f9"),
			Green:    lipgloss.Color("#50fa7b"),
			Peach:    lipgloss.Color("#ffb86c"),
			Red:      lipgloss.Color("#ff5555"),
			Blue:     lipgloss.Color("#8be9fd"),
			Mauve:    lipgloss.Color("#ff79c6"),
			Yellow:   lipgloss.Color("#f1fa8c"),
			Teal:     lipgloss.Color("#8be9fd"),
		}
	case "monokai":
		return Palette{
			Base:     lipgloss.Color("#272822"),
			Surface:  lipgloss.Color("#3e3d32"),
			Overlay:  lipgloss.Color("#75715e"),
			Text:     lipgloss.Color("#f8f8f2"),
			Subtext:  lipgloss.Color("#a59f85"),
			Lavender: lipgloss.Color("#ae81ff"),
			Green:    lipgloss.Color("#a6e22e"),
			Peach:    lipgloss.Color("#fd971f"),
			Red:      lipgloss.Color("#f92672"),
			Blue:     lipgloss.Color("#66d9ef"),
			Mauve:    lipgloss.Color("#ae81ff"),
			Yellow:   lipgloss.Color("#e6db74"),
			Teal:     lipgloss.Color("#66d9ef"),
		}
	default: // catppuccin mocha
		return Palette{
			Base:     lipgloss.Color("#1e1e2e"),
			Surface:  lipgloss.Color("#313244"),
			Overlay:  lipgloss.Color("#585b70"),
			Text:     lipgloss.Color("#cdd6f4"),
			Subtext:  lipgloss.Color("#a6adc8"),
			Lavender: lipgloss.Color("#b4befe"),
			Green:    lipgloss.Color("#a6e3a1"),
			Peach:    lipgloss.Color("#fab387"),
			Red:      lipgloss.Color("#f38ba8"),
			Blue:     lipgloss.Color("#89b4fa"),
			Mauve:    lipgloss.Color("#cba6f7"),
			Yellow:   lipgloss.Color("#f9e2af"),
			Teal:     lipgloss.Color("#94e2d5"),
		}
	}
}

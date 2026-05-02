package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds all user-configurable settings for tui-notes.
type Config struct {
	DefaultEditor     string `json:"default_editor"`      // Editor command (e.g. "nvim", "code")
	ExplorerPath      string `json:"explorer_path"`       // Default directory for file explorer
	Theme             string `json:"theme"`               // Color theme: "catppuccin", "dracula", "monokai"
	ObsidianVaultPath string `json:"obsidian_vault_path"` // Path to local Obsidian vault
	NotionToken       string `json:"notion_token"`        // Notion integration token
}

// DefaultConfig returns the configuration with sensible defaults.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	if home == "" {
		home = "."
	}

	return Config{
		DefaultEditor: "",
		ExplorerPath:  home,
		Theme:         "catppuccin",
	}
}

// Path returns the full path to the config file.
func Path() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		home = "."
	}
	return filepath.Join(home, ".notas-cli", "config.json")
}

// Load reads config from disk. Returns defaults if file doesn't exist.
func Load() Config {
	cfg := DefaultConfig()

	data, err := os.ReadFile(Path())
	if err != nil {
		return cfg
	}

	// Unmarshal on top of defaults so missing fields keep their default value
	_ = json.Unmarshal(data, &cfg)

	// Validate theme
	switch cfg.Theme {
	case "catppuccin", "dracula", "monokai":
		// valid
	default:
		cfg.Theme = "catppuccin"
	}

	return cfg
}

// Save writes the config to disk, creating the directory if needed.
func Save(cfg Config) error {
	p := Path()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(p, data, 0644)
}

// EnsureExists creates the config file with defaults if it doesn't exist yet.
func EnsureExists() error {
	if _, err := os.Stat(Path()); os.IsNotExist(err) {
		return Save(DefaultConfig())
	}
	return nil
}

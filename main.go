package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"notas/config"
	"notas/store"
	"notas/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dbPath := filepath.Join(home, ".notas-cli", "notas.db")
	
	// Ensure config exists with defaults
	config.EnsureExists()
	cfg := config.Load()

	s, err := store.New(dbPath)
	if err != nil {
		fmt.Printf("Error al inicializar la base de datos: %v\n", err)
		os.Exit(1)
	}

	// Integración CLI pura
	if len(os.Args) > 1 {
		handleCLI(s, cfg, os.Args[1:])
		return
	}

	m := tui.New(s, cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func handleCLI(s *store.Store, cfg config.Config, args []string) {
	command := args[0]
	
	switch command {
	case "add":
		if len(args) < 2 {
			fmt.Println("Uso: notas add \"Contenido de la nota con #tags\"")
			return
		}
		// Unimos todo lo que viene después de "add" como el contenido
		content := strings.Join(args[1:], " ")
		
		// Usamos el contenido como título si es corto, o un extracto
		title := "Nota rápida"
		if len(content) < 40 {
			title = content
		} else {
			title = content[:37] + "..."
		}
		
		note, err := s.SaveNote(title, content, "saved")
		if err != nil {
			fmt.Printf("Error guardando la nota: %v\n", err)
			return
		}
		fmt.Printf("✅ Nota #%d guardada correctamente.\n", note.ID)
		if len(note.Tags) > 0 {
			var formatted []string
			for _, t := range note.Tags {
				formatted = append(formatted, "#"+t)
			}
			fmt.Printf("🏷️  Etiquetas detectadas: %s\n", strings.Join(formatted, ", "))
		}
		
	case "list":
		notes := s.GetRecentNotes()
		if len(notes) == 0 {
			fmt.Println("No hay notas guardadas.")
			return
		}
		fmt.Printf("📝 Últimas notas:\n")
		for i, n := range notes {
			if i >= 10 { // límite de 10 en la consola
				break
			}
			tags := ""
			if len(n.Tags) > 0 {
				tags = fmt.Sprintf(" %v", n.Tags)
			}
			fmt.Printf("  [#%d] %s%s\n", n.ID, n.Title, tags)
		}
		
	case "export":
		exportDir := "./export"
		if len(args) > 1 {
			exportDir = args[1]
		}
		
		if err := os.MkdirAll(exportDir, 0755); err != nil {
			fmt.Printf("Error creando directorio de exportación: %v\n", err)
			return
		}
		
		count, err := s.ExportNotes(nil, exportDir)
		if err != nil {
			fmt.Printf("Error durante la exportación: %v\n", err)
			return
		}
		
		fmt.Printf("✅ %d notas exportadas exitosamente a '%s'\n", count, exportDir)

	case "config":
		handleConfigCommand(cfg, args[1:])
		
	case "help", "--help", "-h":
		printHelp()
		
	default:
		fmt.Printf("Comando desconocido: %s\n\n", command)
		printHelp()
	}
}

func handleConfigCommand(cfg config.Config, args []string) {
	cfgPath := config.Path()

	if len(args) == 0 {
		// Show current config
		data, _ := json.MarshalIndent(cfg, "", "  ")
		fmt.Printf("⚙️  Configuración actual (%s):\n\n%s\n", cfgPath, string(data))
		fmt.Println("\nUso:")
		fmt.Println("  notas config edit           Abre el archivo en tu editor")
		fmt.Println("  notas config set <key> <v>  Cambia un valor")
		fmt.Println("  notas config reset          Restaura valores por defecto")
		return
	}

	switch args[0] {
	case "edit":
		editor := cfg.DefaultEditor
		if editor == "" {
			editor = os.Getenv("EDITOR")
		}
		if editor == "" {
			if runtime.GOOS == "windows" {
				editor = "notepad"
			} else {
				editor = "nano"
			}
		}
		cmd := exec.Command(editor, cfgPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error abriendo editor: %v\n", err)
		}

	case "set":
		if len(args) < 3 {
			fmt.Println("Uso: notas config set <key> <valor>")
			fmt.Println("Claves: default_editor, explorer_path, theme")
			return
		}
		key, value := args[1], args[2]
		switch key {
		case "default_editor":
			cfg.DefaultEditor = value
		case "explorer_path":
			cfg.ExplorerPath = value
		case "theme":
			switch value {
			case "catppuccin", "dracula", "monokai":
				cfg.Theme = value
			default:
				fmt.Printf("Tema no válido: %s (opciones: catppuccin, dracula, monokai)\n", value)
				return
			}
		default:
			fmt.Printf("Clave desconocida: %s\n", key)
			return
		}
		if err := config.Save(cfg); err != nil {
			fmt.Printf("Error guardando: %v\n", err)
			return
		}
		fmt.Printf("✅ %s = %s\n", key, value)

	case "reset":
		if err := config.Save(config.DefaultConfig()); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println("✅ Configuración restaurada a valores por defecto")

	default:
		fmt.Printf("Subcomando desconocido: %s\n", args[0])
	}
}

func printHelp() {
	fmt.Println("📝 NOTAS CLI - Modo Consola")
	fmt.Println()
	fmt.Println("Uso:")
	fmt.Println("  notas                     Inicia la interfaz gráfica interactiva (TUI)")
	fmt.Println("  notas add <texto>         Guarda una nota rápida directamente")
	fmt.Println("  notas list                Muestra las 10 notas más recientes")
	fmt.Println("  notas export [ruta]       Exporta todas tus notas a archivos Markdown (.md)")
	fmt.Println("  notas config              Muestra la configuración actual")
	fmt.Println("  notas config edit         Abre config.json en tu editor")
	fmt.Println("  notas config set <k> <v>  Cambia un valor de configuración")
	fmt.Println("  notas config reset        Restaura valores por defecto")
	fmt.Println("  notas help                Muestra este mensaje de ayuda")
}

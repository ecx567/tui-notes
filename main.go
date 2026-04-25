package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"notas/store"
	"notas/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dbPath := filepath.Join(home, ".notas-cli", "notas.json")
	
	s, err := store.New(dbPath)
	if err != nil {
		fmt.Printf("Error al inicializar la base de datos: %v\n", err)
		os.Exit(1)
	}

	// Integración CLI pura
	if len(os.Args) > 1 {
		handleCLI(s, os.Args[1:])
		return
	}

	m := tui.New(s)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func handleCLI(s *store.Store, args []string) {
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
		
	case "help", "--help", "-h":
		printHelp()
		
	default:
		fmt.Printf("Comando desconocido: %s\n\n", command)
		printHelp()
	}
}

func printHelp() {
	fmt.Println("📝 NOTAS CLI - Modo Consola")
	fmt.Println()
	fmt.Println("Uso:")
	fmt.Println("  notas                Inicia la interfaz gráfica interactiva (TUI)")
	fmt.Println("  notas add <texto>    Guarda una nota rápida directamente")
	fmt.Println("  notas list           Muestra las 10 notas más recientes")
	fmt.Println("  notas export [ruta]  Exporta todas tus notas a archivos Markdown (.md)")
	fmt.Println("  notas help           Muestra este mensaje de ayuda")
}

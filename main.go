package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wingitman/listicles/internal/app"
	"github.com/wingitman/listicles/internal/config"
)

func main() {
	cdFile := flag.String("cd-file", "", "file path to write the chosen directory to on exit (used by shell wrapper)")
	openFile := flag.String("open-file", "", "file path to write the selected file path to on exit (used by editor integrations)")
	startDir := flag.String("dir", "", "starting directory (defaults to current working directory)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
	}

	model, err := app.New(cfg, *startDir, *cdFile, *openFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running listicles: %v\n", err)
		os.Exit(1)
	}
}

package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/listicle/listicle/internal/app"
	"github.com/listicle/listicle/internal/config"
)

func main() {
	cdFile := flag.String("cd-file", "", "file path to write the chosen directory to on exit (used by shell wrapper)")
	startDir := flag.String("dir", "", "starting directory (defaults to current working directory)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
	}

	model, err := app.New(cfg, *startDir, *cdFile)
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
		fmt.Fprintf(os.Stderr, "Error running listicle: %v\n", err)
		os.Exit(1)
	}
}

// Package main implements a terminal-based JSON inspector with image preview.
//
// It renders a split-view TUI using Bubble Tea and Lip Gloss, showing
// formatted JSON on the left and an embedded image preview on the right.
// Press ctrl+p for a full-screen high-quality image view using kitty or
// sixel graphics protocols.
//
// Usage:
//
//	glimpse <json-file>
package main

import (
	"log"
	"os"

	tea "charm.land/bubbletea/v2"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: glimpse <json-file>")
	}

	gfx := detectGraphics()

	m, err := initialModel(os.Args[1], gfx)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}

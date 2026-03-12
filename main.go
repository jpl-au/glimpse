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
//	cat data.json | glimpse -
//	psql -t -c "SELECT data FROM docs" mydb | glimpse -
package main

import (
	"io"
	"log"
	"os"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/term"
)

func main() {
	name, raw, err := readInput()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	gfx := detectGraphics()

	m, err := initialModel(name, raw, gfx)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}

// readInput returns a display name and the raw JSON bytes. It reads from
// a file when a path is given, or from stdin when the argument is "-" or
// stdin is a pipe.
func readInput() (string, []byte, error) {
	piped := !term.IsTerminal(int(os.Stdin.Fd()))

	if len(os.Args) >= 2 && os.Args[1] != "-" {
		raw, err := os.ReadFile(os.Args[1])
		return os.Args[1], raw, err
	}

	if len(os.Args) >= 2 && os.Args[1] == "-" || piped {
		raw, err := io.ReadAll(os.Stdin)
		return "stdin", raw, err
	}

	log.Fatal("Usage: glimpse <json-file>  or  ... | glimpse -")
	return "", nil, nil
}

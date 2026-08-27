package main

import (
	"fmt"
	"os"

	"github.com/arshnah/replay/internal/player"
	"github.com/arshnah/replay/internal/record"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "record":
		runRecord(os.Args[2:])
	case "play":
		runPlay(os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: replay record <file> -- <command> [args...]")
	fmt.Fprintln(os.Stderr, "       replay play <file>")
	os.Exit(1)
}

func runRecord(args []string) {
	if len(args) < 1 {
		usage()
	}
	logPath := args[0]
	rest := args[1:]
	if len(rest) > 0 && rest[0] == "--" {
		rest = rest[1:]
	}
	if len(rest) == 0 {
		usage()
	}

	cols, rows := 80, 24
	if w, h, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		cols, rows = w, h
	}

	if err := record.Record(logPath, cols, rows, rest); err != nil {
		fmt.Fprintf(os.Stderr, "replay: %v\n", err)
		os.Exit(1)
	}
}

func runPlay(args []string) {
	if len(args) < 1 {
		usage()
	}

	session, err := player.Load(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "replay: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(player.New(session), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "replay: %v\n", err)
		os.Exit(1)
	}
}

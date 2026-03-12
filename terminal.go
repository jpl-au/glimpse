package main

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// graphics records which terminal image protocols are available.
type graphics struct {
	kitty bool
	sixel bool
}

// detectGraphics probes the terminal for graphics protocol support.
// It must be called before bubbletea takes over stdin. Environment
// variables are checked first as a fast path; if inconclusive a runtime
// query is sent to the terminal with a short timeout.
func detectGraphics() graphics {
	g := graphics{kitty: probeKittyEnv() || probeKittyGraphics()}
	if !g.kitty {
		g.sixel = probeSixelEnv() || probeSixelDA()
	}
	return g
}

// probeKittyEnv checks environment variables for known kitty-protocol terminals.
func probeKittyEnv() bool {
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	termProgram := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	switch termProgram {
	case "kitty", "ghostty", "wezterm":
		return true
	}
	t := strings.ToLower(os.Getenv("TERM"))
	return strings.Contains(t, "kitty") || strings.Contains(t, "ghostty")
}

// probeSixelEnv checks environment variables for known sixel-capable terminals.
func probeSixelEnv() bool {
	if os.Getenv("WT_SESSION") != "" {
		return true
	}
	termProgram := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	switch termProgram {
	case "iterm.app", "wezterm", "konsole", "tabby", "rio":
		return true
	}
	t := strings.ToLower(os.Getenv("TERM"))
	return t == "foot" || strings.HasPrefix(t, "foot-") ||
		t == "xterm-sixel" || strings.Contains(t, "sixel")
}

// probeKittyGraphics sends a tiny kitty graphics query image and checks
// for an OK response. Returns false on timeout or error.
func probeKittyGraphics() bool {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return false
	}

	old, err := term.MakeRaw(fd)
	if err != nil {
		return false
	}
	defer term.Restore(fd, old)

	// Send a 1x1 query image (1 pixel, RGB, direct data).
	// The terminal responds with \x1b_Gi=31;OK\x1b\\ on success.
	query := "\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\"
	if _, err := os.Stdout.WriteString(query); err != nil {
		slog.Debug("kitty probe write failed", "err", err)
		return false
	}

	return readResponse(fd, "OK", 150*time.Millisecond)
}

// probeSixelDA sends a Primary Device Attributes request and checks
// whether the response advertises sixel support (attribute 4).
func probeSixelDA() bool {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return false
	}

	old, err := term.MakeRaw(fd)
	if err != nil {
		return false
	}
	defer term.Restore(fd, old)

	// Primary DA: terminal responds with \x1b[?...c
	// Attribute 4 means sixel graphics support.
	if _, err := os.Stdout.WriteString("\x1b[c"); err != nil {
		slog.Debug("sixel DA probe write failed", "err", err)
		return false
	}

	return readResponse(fd, ";4", 150*time.Millisecond)
}

// readResponse reads bytes from fd until timeout, returning true if the
// accumulated response contains needle.
func readResponse(fd int, needle string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	buf := make([]byte, 128)
	var resp []byte

	for time.Now().Before(deadline) {
		// Poll with a short read timeout via SetReadDeadline-like behaviour.
		// Since os.File doesn't support deadlines on raw fds, we use a
		// non-blocking approach: set a short OS-level timeout and retry.
		n, err := readWithTimeout(fd, buf, time.Until(deadline))
		if n > 0 {
			resp = append(resp, buf[:n]...)
			if strings.Contains(string(resp), needle) {
				return true
			}
		}
		if err != nil {
			break
		}
	}
	return false
}

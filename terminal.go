package main

import (
	"os"
	"strings"
)

// supportsKittyGraphics reports whether the terminal likely supports the kitty
// graphics protocol by checking environment variables for known compatible
// terminals (kitty, ghostty, wezterm).
func supportsKittyGraphics() bool {
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	termProgram := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	switch termProgram {
	case "kitty", "ghostty", "wezterm":
		return true
	}
	term := strings.ToLower(os.Getenv("TERM"))
	return strings.Contains(term, "kitty") || strings.Contains(term, "ghostty")
}

// supportsSixelGraphics reports whether the terminal likely supports sixel
// graphics by checking environment variables for known compatible terminals
// (iTerm2, wezterm, konsole, foot, etc.).
func supportsSixelGraphics() bool {
	if os.Getenv("WT_SESSION") != "" {
		return true
	}
	termProgram := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	switch termProgram {
	case "iterm.app", "wezterm", "konsole", "tabby", "rio":
		return true
	}
	term := strings.ToLower(os.Getenv("TERM"))
	return term == "foot" || strings.HasPrefix(term, "foot-") ||
		term == "xterm-sixel" || strings.Contains(term, "sixel")
}

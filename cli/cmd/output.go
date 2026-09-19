package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Where user-facing output goes. Status lines are for people and go to stderr,
// results are for scripts and go to stdout. Tests swap these to capture both
var (
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr
)

const (
	verbWidth        = 8  // "Archived"
	previewVerbWidth = 11 // "Would clone"

	ansiGreen  = "32"
	ansiYellow = "33"
	ansiDim    = "2"
)

// isTerminal reports whether w is a terminal. Anything that isn't an *os.File,
// like a test buffer, is not
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// styled reports whether status lines get symbols and color
func styled() bool {
	return isTerminal(stderr) && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb" && ansiSupported(stderr)
}

// statusLine formats one status line, like "✓ Cloned   ~/src/...". The symbol
// and its color only show up when styled
func statusLine(fancy bool, symbol, ansi, verb string, width int, detail string) string {
	if !fancy {
		return fmt.Sprintf("%-*s %s", width, verb, detail)
	}
	if ansi != "" {
		symbol = "\x1b[" + ansi + "m" + symbol + "\x1b[0m"
	}
	return fmt.Sprintf("%s %-*s %s", symbol, width, verb, detail)
}

// status prints one status line to stderr
func status(symbol, ansi, verb string, width int, detail string) {
	fmt.Fprintln(stderr, statusLine(styled(), symbol, ansi, verb, width, detail))
}

// done reports something that happened
func done(verb, detail string) { status("✓", ansiGreen, verb, verbWidth, detail) }

// note reports something that didn't need doing
func note(verb, detail string) { status("•", ansiDim, verb, verbWidth, detail) }

// warn reports something that was skipped
func warn(verb, detail string) { status("!", ansiYellow, verb, verbWidth, detail) }

// preview reports what a dry run would have done
func preview(verb, detail string) { status(" ", "", verb, previewVerbWidth, detail) }

// emitPath prints the absolute path of a command's result to stdout, but only
// when stdout isn't a terminal. On a terminal the status line already says
// where it went. Piped, it makes cd "$(sourceseedy clone <url>)" work
func emitPath(p string) {
	if isTerminal(stdout) {
		return
	}
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	fmt.Fprintln(stdout, p)
}

// tildePath shortens p to start with ~ when it is inside the home directory
func tildePath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if p == home {
		return "~"
	}
	if rest, ok := strings.CutPrefix(p, home+string(filepath.Separator)); ok {
		return "~" + string(filepath.Separator) + rest
	}
	return p
}

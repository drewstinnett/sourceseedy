//go:build !windows

package cmd

import "io"

// ansiSupported reports whether w understands escape sequences. Anything that
// is a terminal at all does, off Windows
func ansiSupported(io.Writer) bool { return true }

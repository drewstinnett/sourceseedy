// Package fakeexe lets a test binary stand in for another program, like fzf or
// sourceseedy itself. Shell scripts can't do this on Windows, so instead the
// test binary copies itself in to a temp directory on PATH under the program's
// name, and its TestMain checks Is to see when it has been started that way
package fakeexe

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Is reports whether this process was started as the program called name
func Is(name string) bool {
	base := filepath.Base(os.Args[0])
	if runtime.GOOS == "windows" {
		base = strings.TrimSuffix(strings.ToLower(base), ".exe")
		name = strings.ToLower(name)
	}
	return base == name
}

// Install puts a copy of the running test binary called name first on PATH for
// the rest of the test, and returns the directory it is in
func Install(tb testing.TB, name string) string {
	tb.Helper()
	self, err := os.Executable()
	if err != nil {
		tb.Fatal(err)
	}
	dir := tb.TempDir()
	dst := filepath.Join(dir, name)
	if runtime.GOOS == "windows" {
		dst += ".exe"
	}
	if err := os.Link(self, dst); err != nil {
		// Different filesystem, or links aren't allowed
		if err := copyExecutable(self, dst); err != nil {
			tb.Fatal(err)
		}
	}
	tb.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return dir
}

func copyExecutable(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

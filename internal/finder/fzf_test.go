package finder_test

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/finder"
)

// fakeFzf puts an fzf script on PATH that records its args and stdin in dir,
// prints out, and exits with code
func fakeFzf(t *testing.T, out string, code int) (argsFile, stdinFile string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake fzf is a shell script")
	}
	dir := t.TempDir()
	argsFile = filepath.Join(dir, "args")
	stdinFile = filepath.Join(dir, "stdin")
	script := "#!/bin/sh\n" +
		"for a in \"$@\"; do printf '%s\\n' \"$a\" >> '" + argsFile + "'; done\n" +
		"cat > '" + stdinFile + "'\n" +
		"printf '%s\\n' '" + out + "'\n" +
		"exit " + strconv.Itoa(code) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "fzf"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return argsFile, stdinFile
}

func newBase(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "github.com/a/one/.git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return base
}

func TestStreamFzfProjectsSelection(t *testing.T) {
	_, stdinFile := fakeFzf(t, "github.com/a/one", 0)
	got, err := finder.StreamFzfProjects(newBase(t), "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "github.com/a/one" {
		t.Errorf("got %q", got)
	}
	in, err := os.ReadFile(stdinFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(in) != "github.com/a/one\n" {
		t.Errorf("fzf stdin = %q", in)
	}
}

func TestStreamFzfProjectsCancelled(t *testing.T) {
	for _, code := range []int{1, 130} {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			fakeFzf(t, "", code)
			_, err := finder.StreamFzfProjects(newBase(t), "")
			if !errors.Is(err, finder.ErrNoSelection) {
				t.Errorf("got %v, want ErrNoSelection", err)
			}
		})
	}
}

func TestStreamFzfProjectsFilterIsLiteral(t *testing.T) {
	argsFile, _ := fakeFzf(t, "github.com/a/one", 0)
	marker := filepath.Join(t.TempDir(), "injected")
	filter := `a"b$(touch ` + marker + `)` + "`touch " + marker + "`"
	if _, err := finder.StreamFzfProjects(newBase(t), filter); err != nil {
		t.Fatal(err)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if want := "+m\n-q\n" + filter + "\n"; string(args) != want {
		t.Errorf("args = %q, want %q", args, want)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Error("filter was executed by a shell")
	}
}

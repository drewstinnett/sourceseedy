package finder_test

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/fakeexe"
	"github.com/drewstinnett/sourceseedy/internal/finder"
)

func TestMain(m *testing.M) {
	if fakeexe.Is("fzf") {
		fakeFzfMain()
	}
	os.Exit(m.Run())
}

// fakeFzfMain is what the fake fzf does when TestMain finds itself running as
// fzf. The test says what to do through environment variables
func fakeFzfMain() {
	for _, a := range os.Args[1:] {
		f, err := os.OpenFile(os.Getenv("FAKE_FZF_ARGS_FILE"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			os.Exit(99)
		}
		fmt.Fprintln(f, a)
		_ = f.Close()
	}
	in, _ := io.ReadAll(os.Stdin)
	if err := os.WriteFile(os.Getenv("FAKE_FZF_STDIN_FILE"), in, 0o600); err != nil {
		os.Exit(99)
	}
	fmt.Println(os.Getenv("FAKE_FZF_OUT"))
	code, _ := strconv.Atoi(os.Getenv("FAKE_FZF_CODE"))
	os.Exit(code)
}

// fakeFzf puts an fzf on PATH that records its args and stdin, prints out, and
// exits with code. It returns the files the args and stdin end up in
func fakeFzf(t *testing.T, out string, code int) (argsFile, stdinFile string) {
	t.Helper()
	dir := t.TempDir()
	argsFile = filepath.Join(dir, "args")
	stdinFile = filepath.Join(dir, "stdin")
	t.Setenv("FAKE_FZF_ARGS_FILE", argsFile)
	t.Setenv("FAKE_FZF_STDIN_FILE", stdinFile)
	t.Setenv("FAKE_FZF_OUT", out)
	t.Setenv("FAKE_FZF_CODE", strconv.Itoa(code))
	fakeexe.Install(t, "fzf")
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

func TestStreamFzfProjectsTrimsCRLF(t *testing.T) {
	// fzf on Windows may end its lines with \r\n
	fakeFzf(t, "github.com/a/one\r", 0)
	got, err := finder.StreamFzfProjects(newBase(t), "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "github.com/a/one" {
		t.Errorf("got %q", got)
	}
}

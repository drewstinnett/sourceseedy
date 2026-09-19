package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/git"
)

// capture swaps stdout and stderr for buffers until the test ends. Tests that
// use it can't run in parallel
func capture(t *testing.T) (out, errOut *bytes.Buffer) {
	t.Helper()
	out, errOut = &bytes.Buffer{}, &bytes.Buffer{}
	// Status lines shorten paths under the home directory to ~. On Windows the
	// temp directory is under it, so give these tests a home that isn't
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	oldOut, oldErr := stdout, stderr
	stdout, stderr = out, errOut
	t.Cleanup(func() { stdout, stderr = oldOut, oldErr })
	return out, errOut
}

// newLocalRepo makes a git repo with remote as its origin and returns its path
func newLocalRepo(t *testing.T, remote string) string {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "myrepo")
	if err := git.SysGit(nil, "init", "-q", repo); err != nil {
		t.Fatal(err)
	}
	if remote != "" {
		if err := git.SysGit(&git.SysGitConfig{Directory: repo}, "remote", "add", "origin", remote); err != nil {
			t.Fatal(err)
		}
	}
	return repo
}

// noNoise fails if s looks like a raw slog line or has terminal escapes
func noNoise(t *testing.T, s string) {
	t.Helper()
	for _, bad := range []string{"level=", "time=", "\x1b"} {
		if strings.Contains(s, bad) {
			t.Errorf("output should not contain %q:\n%s", bad, s)
		}
	}
}

func TestTildePath(t *testing.T) {
	// Paths are written with slashes and made native below, so the same table
	// covers Windows. os.UserHomeDir reads USERPROFILE there
	home := filepath.FromSlash("/home/drew")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	for in, want := range map[string]string{
		"/home/drew":                   "~",
		"/home/drew/src/github.com/a":  "~/src/github.com/a",
		"/home/drewbie/src":            "/home/drewbie/src",
		"/home/dre":                    "/home/dre",
		"/opt/src":                     "/opt/src",
		"./relative/path":              "./relative/path",
		"":                             "",
		"/home/drew-not-quite/../drew": "/home/drew-not-quite/../drew",
	} {
		in, want := filepath.FromSlash(in), filepath.FromSlash(want)
		if got := tildePath(in); got != want {
			t.Errorf("tildePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStatusLine(t *testing.T) {
	for _, tt := range []struct {
		name   string
		styled bool
		symbol string
		ansi   string
		verb   string
		width  int
		want   string
	}{
		{"plain drops symbol", false, "✓", ansiGreen, "Cloned", verbWidth, "Cloned   ~/src/a"},
		{"styled colors symbol", true, "✓", ansiGreen, "Cloned", verbWidth, "\x1b[32m✓\x1b[0m Cloned   ~/src/a"},
		{"styled aligns longest verb", true, "•", ansiDim, "Archived", verbWidth, "\x1b[2m•\x1b[0m Archived ~/src/a"},
		{"preview has no color", true, " ", "", "Would clone", previewVerbWidth, "  Would clone ~/src/a"},
		{"plain preview", false, " ", "", "Would move", previewVerbWidth, "Would move  ~/src/a"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := statusLine(tt.styled, tt.symbol, tt.ansi, tt.verb, tt.width, "~/src/a"); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsTerminal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no /dev/null")
	}
	if isTerminal(&bytes.Buffer{}) {
		t.Error("a buffer is not a terminal")
	}
	f, err := os.Create(filepath.Join(t.TempDir(), "out"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	if isTerminal(f) {
		t.Error("a regular file is not a terminal")
	}
	// A character device stands in for a tty
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = null.Close() })
	if !isTerminal(null) {
		t.Error("a character device should count as a terminal")
	}
}

func TestEmitPathWhenPiped(t *testing.T) {
	out, _ := capture(t)
	abs, err := filepath.Abs(filepath.FromSlash("/some/where"))
	if err != nil {
		t.Fatal(err)
	}
	emitPath(filepath.FromSlash("/some/where"))
	if got := out.String(); got != abs+"\n" {
		t.Errorf("piped stdout got %q, want %q", got, abs+"\n")
	}
	// Relative paths are made absolute so they still work after a cd
	out.Reset()
	emitPath(filepath.Join("rel", "dir"))
	got := strings.TrimSuffix(out.String(), "\n")
	if !filepath.IsAbs(got) || !strings.HasSuffix(got, string(filepath.Separator)+filepath.Join("rel", "dir")) {
		t.Errorf("expected an absolute path, got %q", out.String())
	}
}

func TestImportRemoteOutput(t *testing.T) {
	fakeRemotes(t, "a/b.git", "a/c.git")
	b := t.TempDir()
	out, errOut := capture(t)

	dest := filepath.Join(b, "git.example.com/a/b")
	url := "https://git.example.com/a/b.git"

	if err := run([]string{"-b", b, "import", url}); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != dest+"\n" {
		t.Errorf("stdout got %q, want %q", got, dest+"\n")
	}
	if got := errOut.String(); !strings.Contains(got, "Cloned") || !strings.Contains(got, dest) {
		t.Errorf("expected a Cloned line naming %s, got:\n%s", dest, got)
	}
	noNoise(t, errOut.String())

	// Again: already there, and the path is still handed back for cd
	out.Reset()
	errOut.Reset()
	if err := run([]string{"-b", b, "import", url}); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != dest+"\n" {
		t.Errorf("stdout got %q, want %q", got, dest+"\n")
	}
	if got := errOut.String(); !strings.Contains(got, "Exists") || strings.Contains(got, "Cloned") {
		t.Errorf("expected only an Exists line, got:\n%s", got)
	}
	noNoise(t, errOut.String())

	// Dry run says what it would do, and prints no path since there isn't one
	out.Reset()
	errOut.Reset()
	if err := run([]string{"-b", b, "import", "-d", "https://git.example.com/a/c.git"}); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Errorf("dry run should not print a path, got %q", out.String())
	}
	if got := errOut.String(); !strings.Contains(got, "Would clone") {
		t.Errorf("expected a Would clone line, got:\n%s", got)
	}
	noNoise(t, errOut.String())
}

func TestCloneOutput(t *testing.T) {
	fakeRemotes(t, "a/b.git")
	b := t.TempDir()
	out, errOut := capture(t)
	if err := run([]string{"-b", b, "clone", "https://git.example.com/a/b.git"}); err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(b, "git.example.com/a/b") + "\n"; out.String() != want {
		t.Errorf("stdout got %q, want %q", out.String(), want)
	}
	if !strings.Contains(errOut.String(), "Cloned") {
		t.Errorf("expected a Cloned line, got:\n%s", errOut.String())
	}
	noNoise(t, errOut.String())
}

func TestImportLocalOutput(t *testing.T) {
	b := t.TempDir()
	repo := newLocalRepo(t, "git@git.example.com:a/b.git")
	dest := filepath.Join(b, "git.example.com/a/b")
	out, errOut := capture(t)

	if err := run([]string{"-b", b, "import", "-d", repo}); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Errorf("dry run should not print a path, got %q", out.String())
	}
	if got := errOut.String(); !strings.Contains(got, "Would move") || !strings.Contains(got, dest) {
		t.Errorf("expected a Would move line naming %s, got:\n%s", dest, got)
	}
	noNoise(t, errOut.String())

	out.Reset()
	errOut.Reset()
	if err := run([]string{"-b", b, "import", repo}); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != dest+"\n" {
		t.Errorf("stdout got %q, want %q", got, dest+"\n")
	}
	if got := errOut.String(); !strings.Contains(got, "Moved") || !strings.Contains(got, dest) {
		t.Errorf("expected a Moved line naming %s, got:\n%s", dest, got)
	}
	noNoise(t, errOut.String())

	// The place is now taken, so importing a second copy is skipped, and
	// nothing is reported as imported
	out.Reset()
	errOut.Reset()
	dup := newLocalRepo(t, "git@git.example.com:a/b.git")
	if err := run([]string{"-b", b, "import", dup}); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Errorf("a skipped import should not print a path, got %q", out.String())
	}
	if got := errOut.String(); !strings.Contains(got, "Skipped") || !strings.Contains(got, "already exists") {
		t.Errorf("expected a Skipped line, got:\n%s", got)
	}
	if !git.IsLocalGitRepo(dup) {
		t.Error("a skipped repo should stay where it was")
	}
	noNoise(t, errOut.String())
}

func TestImportLocalWithoutRemoteIsSkipped(t *testing.T) {
	b := t.TempDir()
	repo := newLocalRepo(t, "")
	out, errOut := capture(t)
	if err := run([]string{"-b", b, "import", repo}); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Errorf("expected no path, got %q", out.String())
	}
	if got := errOut.String(); !strings.Contains(got, "Skipped") {
		t.Errorf("expected a Skipped line, got:\n%s", got)
	}
	if !git.IsLocalGitRepo(repo) {
		t.Error("a skipped repo should stay where it was")
	}
	noNoise(t, errOut.String())
}

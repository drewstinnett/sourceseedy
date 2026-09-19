// Package gittest builds real git repos for tests
package gittest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Isolate keeps the developer's own git config out of the rest of the test,
// so that things like commit.gpgsign or a different default branch name can't
// break it, and gives commits an author
func Isolate(tb testing.TB) {
	tb.Helper()
	empty := filepath.Join(tb.TempDir(), "gitconfig")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		tb.Fatal(err)
	}
	tb.Setenv("GIT_CONFIG_GLOBAL", empty)
	tb.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, who := range []string{"AUTHOR", "COMMITTER"} {
		tb.Setenv("GIT_"+who+"_NAME", "Test")
		tb.Setenv("GIT_"+who+"_EMAIL", "test@example.com")
	}
}

// Git runs git in dir and returns its output, failing the test if it fails
func Git(tb testing.TB, dir string, args ...string) string {
	tb.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		tb.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// Commit writes content to the file name in repo, and commits it
func Commit(tb testing.TB, repo, name, content string) {
	tb.Helper()
	if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0o600); err != nil {
		tb.Fatal(err)
	}
	Git(tb, repo, "add", name)
	Git(tb, repo, "commit", "-q", "-m", "add "+name)
}

// Remote returns the path of a bare repo with one commit, on main. Call
// Isolate first
func Remote(tb testing.TB) string {
	tb.Helper()
	root := tb.TempDir()
	bare := filepath.Join(root, "remote.git")
	Git(tb, root, "init", "-q", "--bare", "-b", "main", bare)
	seed := filepath.Join(root, "seed")
	Git(tb, root, "init", "-q", "-b", "main", seed)
	Commit(tb, seed, "README.md", "hello\n")
	Git(tb, seed, "push", "-q", bare, "main")
	return bare
}

// Clone clones remote in to dest, creating any missing directories, and
// returns dest
func Clone(tb testing.TB, remote, dest string) string {
	tb.Helper()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		tb.Fatal(err)
	}
	Git(tb, filepath.Dir(dest), "clone", "-q", remote, dest)
	return dest
}

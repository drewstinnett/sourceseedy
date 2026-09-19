package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/git"
)

// fakeRemotes makes https://git.example.com/ clone from a local directory
// of bare repos, and returns that directory
func fakeRemotes(t *testing.T, repos ...string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	for _, r := range repos {
		if out, err := exec.Command("git", "init", "--bare", "-q", filepath.Join(dir, r)).CombinedOutput(); err != nil {
			t.Fatalf("git init: %v\n%s", err, out)
		}
	}
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "url."+fileURL(dir)+"/.insteadOf")
	t.Setenv("GIT_CONFIG_VALUE_0", "https://git.example.com/")
}

// fileURL returns the file:// URL of dir. On Windows that is file:///C:/dir,
// with a slash before the drive letter
func fileURL(dir string) string {
	slashed := filepath.ToSlash(dir)
	if !strings.HasPrefix(slashed, "/") {
		slashed = "/" + slashed
	}
	return "file://" + slashed
}

func TestCloneInPlace(t *testing.T) {
	fakeRemotes(t, "group/sub/repo.git")
	b := t.TempDir()
	url := "https://git.example.com/group/sub/repo.git"
	dest := filepath.Join(b, "git.example.com/group/sub/repo")

	if err := run([]string{"-b", b, "clone", "-d", url}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); err == nil {
		t.Fatal("dry run should not clone")
	}

	if err := run([]string{"-b", b, "clone", url}); err != nil {
		t.Fatal(err)
	}
	if !git.IsLocalGitRepo(dest) {
		t.Fatalf("expected a git repo at %s", dest)
	}

	// Cloning again is fine, and doesn't touch the existing repo
	if err := run([]string{"-b", b, "clone", url}); err != nil {
		t.Fatal(err)
	}
}

func TestCloneRefusesUnplaceable(t *testing.T) {
	b := t.TempDir()
	for _, url := range []string{
		"/tmp/some/local/repo",
		"https://evil.com/../../etc/passwd",
		"--upload-pack=touch/pwned/x",
	} {
		// The -- keeps flag parsing from claiming the last one
		if err := run([]string{"-b", b, "clone", "--", url}); err == nil {
			t.Errorf("expected error cloning %q", url)
		}
	}
	if entries, err := os.ReadDir(b); err != nil || len(entries) != 0 {
		t.Errorf("expected base to stay empty, got %v (err %v)", entries, err)
	}
	if err := run([]string{"-b", b, "clone"}); err == nil {
		t.Error("expected error with no repos")
	}
}

func TestBaseFromEnv(t *testing.T) {
	fakeRemotes(t, "a/b.git")
	b := t.TempDir()
	t.Setenv("SOURCESEEDY_BASE", b)
	if err := run([]string{"clone", "https://git.example.com/a/b.git"}); err != nil {
		t.Fatal(err)
	}
	if !git.IsLocalGitRepo(filepath.Join(b, "git.example.com/a/b")) {
		t.Error("expected clone under SOURCESEEDY_BASE")
	}

	// -b wins over the environment
	flagBase := t.TempDir()
	if err := run([]string{"-b", flagBase, "clone", "https://git.example.com/a/b.git"}); err != nil {
		t.Fatal(err)
	}
	if !git.IsLocalGitRepo(filepath.Join(flagBase, "git.example.com/a/b")) {
		t.Error("expected clone under -b")
	}
}

func TestImportRemoteClonesInPlace(t *testing.T) {
	fakeRemotes(t, "a/b.git")
	b := t.TempDir()
	if err := run([]string{"-b", b, "import", "https://git.example.com/a/b.git"}); err != nil {
		t.Fatal(err)
	}
	if !git.IsLocalGitRepo(filepath.Join(b, "git.example.com/a/b")) {
		t.Error("expected import of a URL to clone in to place")
	}
}

func TestImportLocalMovesRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	b := t.TempDir()
	repo := filepath.Join(t.TempDir(), "myrepo")
	if err := git.SysGit(nil, "init", "-q", repo); err != nil {
		t.Fatal(err)
	}
	if err := git.SysGit(&git.SysGitConfig{Directory: repo}, "remote", "add", "origin", "git@git.example.com:a/b.git"); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(b, "git.example.com/a/b")

	if err := run([]string{"-b", b, "import", "-d", repo}); err != nil {
		t.Fatal(err)
	}
	if !git.IsLocalGitRepo(repo) || git.IsLocalGitRepo(dest) {
		t.Fatal("dry run should leave the repo where it is")
	}

	if err := run([]string{"-b", b, "import", repo}); err != nil {
		t.Fatal(err)
	}
	if !git.IsLocalGitRepo(dest) || git.IsLocalGitRepo(repo) {
		t.Error("expected repo to move in to place")
	}
}

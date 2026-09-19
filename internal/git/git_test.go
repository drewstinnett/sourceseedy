package git_test

import (
	"path"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/git"
)

func TestIsLocalGitRepo(t *testing.T) {
	if !git.IsLocalGitRepo(path.Join(testBase, "badhost/somenamespace/someproject")) {
		t.Error("expected badhost/somenamespace/someproject to be a git repo")
	}
	if git.IsLocalGitRepo(path.Join(testBase, "fake.com/emptydir")) {
		t.Error("expected fake.com/emptydir to not be a git repo")
	}
	if git.IsLocalGitRepo("/some-nonexist") {
		t.Error("expected /some-nonexist to not be a git repo")
	}
}

func TestSysGit(t *testing.T) {
	c := &git.SysGitConfig{
		Directory: t.TempDir(),
	}
	if err := git.SysGit(c, "init", "."); err != nil {
		t.Fatal(err)
	}
}

func TestSysGitOutput(t *testing.T) {
	c := &git.SysGitConfig{
		Directory: t.TempDir(),
	}
	if err := git.SysGit(c, "init", "."); err != nil {
		t.Fatal(err)
	}
	if err := git.SysGit(c, "remote", "add", "origin", "git@example.com:a/b.git"); err != nil {
		t.Fatal(err)
	}
	got, err := git.SysGitOutput(c, "remote", "get-url", "origin")
	if err != nil {
		t.Fatal(err)
	}
	if want := "git@example.com:a/b.git"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

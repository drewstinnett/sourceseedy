package git_test

import (
	"path"
	"slices"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/git"
)

func TestFindGit(t *testing.T) {
	res, err := git.FindGit(testBase)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(res, "/badhost/somenamespace/someproject") {
		t.Errorf("expected /badhost/somenamespace/someproject in %v", res)
	}
	if slices.Contains(res, "/fake.com/emptydir") {
		t.Errorf("did not expect /fake.com/emptydir in %v", res)
	}
}

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

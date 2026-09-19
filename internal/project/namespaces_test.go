package project_test

import (
	"os"
	"path"
	"runtime"
	"slices"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/project"
)

func TestListProjectsFromNamespace(t *testing.T) {
	n := project.Namespace{
		Name:      "somenamespace",
		Host:      "fake.com",
		Directory: path.Join(testBase, "fake.com", "somenamespace"),
	}
	ps, err := n.ListProjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) == 0 {
		t.Error("expected projects, got none")
	}
}

func TestListProjectsMissingDir(t *testing.T) {
	n := project.Namespace{Directory: path.Join(t.TempDir(), "not-exists")}
	if _, err := n.ListProjects(); err == nil {
		t.Error("expected error listing a missing namespace directory")
	}
}

func TestListProjectsSkipsUnreadable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows doesn't stop reading a directory because of its mode")
	}
	if os.Geteuid() == 0 {
		t.Skip("root can read directories regardless of permissions")
	}
	dir := t.TempDir()
	if err := os.MkdirAll(path.Join(dir, "ok", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	locked := path.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	// Restore permissions so t.TempDir cleanup can remove it
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	ps, err := project.Namespace{Directory: dir}.ListProjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 1 || ps[0].Name != "ok" {
		t.Errorf("expected only project ok, got %+v", ps)
	}
}

func TestListProjectsDoesNotDescendIntoRepos(t *testing.T) {
	dir := t.TempDir()
	for _, item := range []string{
		"proj/.git/modules/inner/.git",
		"proj/vendor/nested/.git",
		"group/sub/other/.git",
	} {
		if err := os.MkdirAll(path.Join(dir, item), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	ps, err := project.Namespace{Directory: dir}.ListProjects()
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range ps {
		got = append(got, p.Name)
	}
	slices.Sort(got)
	if want := []string{"group/sub/other", "proj"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestListProjectsAcceptsGitFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(path.Join(dir, "worktree"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path.Join(dir, "worktree", ".git"), []byte("gitdir: elsewhere\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ps, err := project.Namespace{Directory: dir}.ListProjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 1 || ps[0].Name != "worktree" {
		t.Errorf("expected only project worktree, got %+v", ps)
	}
}

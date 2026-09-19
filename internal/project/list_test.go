package project_test

import (
	"os"
	"path"
	"path/filepath"
	"slices"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/project"
)

func TestListAllProjectFullIDsSorted(t *testing.T) {
	base := t.TempDir()
	for _, item := range []string{
		"gitlab.com/zed/last/.git",
		"github.com/b/two/.git",
		"github.com/a/one/.git",
		"github.com/a/nested/deeper/.git",
	} {
		if err := os.MkdirAll(path.Join(base, item), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{
		"github.com/a/nested/deeper",
		"github.com/a/one",
		"github.com/b/two",
		"gitlab.com/zed/last",
	}
	// Goroutine completion order varies, so check repeatedly
	for range 20 {
		got, err := project.ListAllProjectFullIDs(base)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestListAllProjectFullIDsMissingBase(t *testing.T) {
	if _, err := project.ListAllProjectFullIDs(path.Join(t.TempDir(), "not-exists")); err == nil {
		t.Error("expected error listing a missing base")
	}
}

func TestListAllProjects(t *testing.T) {
	base := t.TempDir()
	for _, item := range []string{"gitlab.com/grp/sub/repo/.git", "github.com/a/one/.git"} {
		if err := os.MkdirAll(path.Join(base, item), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got, err := project.ListAllProjects(base)
	if err != nil {
		t.Fatal(err)
	}
	want := []project.Project{
		{Host: "github.com", Namespace: "a", Name: "one", Directory: filepath.Join(base, "github.com", "a", "one")},
		{Host: "gitlab.com", Namespace: "grp", Name: "sub/repo", Directory: filepath.Join(base, "gitlab.com", "grp", "sub", "repo")},
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

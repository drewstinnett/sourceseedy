package project_test

import (
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/git"
	"github.com/drewstinnett/sourceseedy/internal/project"
)

func TestProjectFullID(t *testing.T) {
	tests := []struct {
		project project.Project
		want    string
	}{
		{
			project.Project{Name: "foo", Host: "github.com", Namespace: "mygroup"},
			"github.com/mygroup/foo",
		},
	}
	for _, test := range tests {
		if got := test.project.FullID(); got != test.want {
			t.Errorf("got %q, want %q", got, test.want)
		}
	}
}

func TestDetectProperPathFromURL(t *testing.T) {
	tests := []struct {
		remote  string
		want    string
		wanterr bool
	}{
		{"git@github.com:drewstinnett/sourceseedy.git", "github.com/drewstinnett/sourceseedy", false},
		{"git@github.com:a/b", "github.com/a/b", false},
		{"https://github.com/a/b.git", "github.com/a/b", false},
		{"https://user@github.com/a/b.git", "github.com/a/b", false},
		{"ssh://git@gitlab.com:2222/grp/sub/repo.git", "gitlab.com/grp/sub/repo", false},
		{"github.com/foo/bar", "github.com/foo/bar", false},
		{"bad", "", true},
	}
	for _, tt := range tests {
		got, err := project.DetectProperPathFromURL(tt.remote)
		if got != tt.want {
			t.Errorf("%v: got %q, want %q", tt.remote, got, tt.want)
		}
		if tt.wanterr && err == nil {
			t.Errorf("%v: expected error", tt.remote)
		} else if !tt.wanterr && err != nil {
			t.Errorf("%v: unexpected error: %v", tt.remote, err)
		}
	}
}

func TestListAllProjectFullIDs(t *testing.T) {
	ps, err := project.ListAllProjectFullIDs(testBase)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(ps, "fake.com/somenamespace/someproject") {
		t.Errorf("expected fake.com/somenamespace/someproject in %v", ps)
	}
}

func TestDetectProperPath(t *testing.T) {
	fakedir := t.TempDir()
	tests := []struct {
		base    string
		remotes []string
		want    string
	}{
		{
			"test-origin",
			[]string{"origin", "github.com/foo/bar"},
			"github.com/foo/bar",
		},
		{
			"test-scp-origin",
			[]string{"origin", "git@github.com:foo/baz.git"},
			"github.com/foo/baz",
		},
	}
	for _, test := range tests {
		c := &git.SysGitConfig{
			Directory: path.Join(fakedir, test.base),
		}
		if err := git.SysGit(nil, "init", c.Directory); err != nil {
			t.Fatal(err)
		}
		if err := git.SysGit(c, "remote", "add", test.remotes[0], test.remotes[1]); err != nil {
			t.Fatal(err)
		}

		got, err := project.DetectProperPath(c.Directory)
		if err != nil {
			t.Fatal(err)
		}
		if got != test.want {
			t.Errorf("got %q, want %q", got, test.want)
		}
	}
}

func TestDetectProperPathErrors(t *testing.T) {
	fakedir := t.TempDir()
	noRemote := path.Join(fakedir, "no-remote")
	if err := git.SysGit(nil, "init", noRemote); err != nil {
		t.Fatal(err)
	}
	unplaceable := path.Join(fakedir, "unplaceable")
	if err := git.SysGit(nil, "init", unplaceable); err != nil {
		t.Fatal(err)
	}
	if err := git.SysGit(&git.SysGitConfig{Directory: unplaceable}, "remote", "add", "origin", "/some/local/path"); err != nil {
		t.Fatal(err)
	}

	for name, tt := range map[string]struct {
		dir  string
		want string
	}{
		"no origin":        {noRemote, "no origin remote"},
		"unplaceable":      {unplaceable, "cannot place"},
		"not a repository": {fakedir, "getting origin remote"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := project.DetectProperPath(tt.dir)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("got error %v, want one containing %q", err, tt.want)
			}
		})
	}
}

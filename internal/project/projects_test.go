package project_test

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/git"
	"github.com/drewstinnett/sourceseedy/internal/project"
	"github.com/stretchr/testify/require"
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
		got := test.project.FullID()
		require.Equal(t, test.want, got)
	}
}

func TestDetectProperPathFromURL(t *testing.T) {
	tests := []struct {
		remote  string
		want    string
		wanterr bool
	}{
		{"git@github.com:drewstinnett/sourceseedy.git", "github.com/drewstinnett/sourceseedy", false},
		{"bad", "", true},
	}
	for _, tt := range tests {
		got, err := project.DetectProperPathFromURL(tt.remote)
		require.Equal(t, tt.want, got)
		if tt.wanterr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
}

func TestListAllProjectFullIDs(t *testing.T) {
	ps, err := project.ListAllProjectFullIDs(testBase)
	require.NoError(t, err)
	require.Greater(t, len(ps), 0)
	require.Subset(t, ps, []string{"fake.com/somenamespace/someproject"})
}

func TestDetectProperPath(t *testing.T) {
	fakedir, err := ioutil.TempDir("", "sourceseedy-tests-path")
	defer os.RemoveAll(fakedir)
	if err != nil {
		panic(err)
	}
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
	}
	for _, test := range tests {
		testdir := path.Join(fakedir, test.base)
		err := os.MkdirAll(testdir, 0o755)
		require.NoError(t, err)

		c := &git.SysGitConfig{
			Directory: testdir,
		}
		err = git.SysGit(c, "init", ".")
		require.NoError(t, err)

		err = git.SysGit(c, "remote", "add", test.remotes[0], test.remotes[1])
		require.NoError(t, err)

		got, err := project.DetectProperPath(testdir)
		require.NoError(t, err)
		require.Equal(t, test.want, got)
	}
}

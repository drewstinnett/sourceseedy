package git_test

import (
	"path"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/git"
	"github.com/stretchr/testify/require"
)

func TestFindGit(t *testing.T) {
	res, err := git.FindGit(testBase)

	require.NoError(t, err)
	require.Subset(t, res, []string{"/badhost/somenamespace/someproject"})
	require.NotSubset(t, res, []string{"/fake.com/emptydir"})
	require.Greater(t, len(res), 0)
}

func TestIsLocalGitRepo(t *testing.T) {
	require.True(t, git.IsLocalGitRepo(path.Join(testBase, "badhost/somenamespace/someproject")))
	require.False(t, git.IsLocalGitRepo(path.Join(testBase, "fake.com/emptydir")))
	require.False(t, git.IsLocalGitRepo(path.Join("/some-nonexist")))
}

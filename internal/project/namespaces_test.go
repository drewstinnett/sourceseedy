package project_test

import (
	"path"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/project"
	"github.com/stretchr/testify/require"
)

func TestListProjectsFromNamespace(t *testing.T) {
	n := project.Namespace{
		Name:      "somenamespace",
		Host:      "fake.com",
		Directory: path.Join(testBase, "fake.com", "somenamespace"),
	}
	ps, err := n.ListProjects()
	require.NoError(t, err)
	require.Greater(t, len(ps), 0)
}

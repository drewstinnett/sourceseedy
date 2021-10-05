package project_test

import (
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/project"
	"github.com/stretchr/testify/require"
)

func TestListProjectsFromNamespace(t *testing.T) {
	n := project.Namespace{
		Name:      "someowner",
		Host:      "fake.com",
		Directory: "./testdata/fake.com/someowner",
	}
	ps, err := n.ListProjects()
	require.NoError(t, err)
	require.Greater(t, len(ps), 0)
}

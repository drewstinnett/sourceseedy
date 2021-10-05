package project_test

import (
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/project"
	"github.com/stretchr/testify/require"
)

func TestListHosts(t *testing.T) {
	hosts, err := project.ListHosts("./testdata")
	require.NoError(t, err)

	// Make sure we got hosts back
	require.Greater(t, len(hosts), 0)

	_, err = project.ListHosts("./not-exists")
	require.Error(t, err)
}

func TestListProjects(t *testing.T) {
	h := project.Host{
		Name:      "fake.com",
		Flavor:    "github",
		Directory: "./testdata/fake.com",
	}
	projects, err := h.ListProjects()
	require.NoError(t, err)
	require.Greater(t, len(projects), 0)
}

func TestListNamespaces(t *testing.T) {
	h := project.Host{
		Name:      "fake.com",
		Flavor:    "github",
		Directory: "./testdata/fake.com",
	}
	nss, err := h.ListNamespaces()
	require.NoError(t, err)
	require.Greater(t, len(nss), 0)
}

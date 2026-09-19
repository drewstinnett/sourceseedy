package project_test

import (
	"path"
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

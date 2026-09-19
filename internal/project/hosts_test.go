package project_test

import (
	"path"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/project"
)

func TestListHosts(t *testing.T) {
	hosts, err := project.ListHosts(testBase)
	if err != nil {
		t.Fatal(err)
	}

	// Make sure we got hosts back
	if len(hosts) == 0 {
		t.Error("expected hosts, got none")
	}

	if _, err = project.ListHosts("./not-exists"); err == nil {
		t.Error("expected error listing non-existent base")
	}
}

func TestListNamespaces(t *testing.T) {
	h := project.Host{
		Name:      "fake.com",
		Directory: path.Join(testBase, "fake.com"),
	}
	nss, err := h.ListNamespaces()
	if err != nil {
		t.Fatal(err)
	}
	if len(nss) == 0 {
		t.Error("expected namespaces, got none")
	}
}

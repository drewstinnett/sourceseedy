package archive_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/archive"
)

func requireTar(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("tar"); err != nil {
		t.Skip("tar not available")
	}
}

func TestCreateArchiveCreatesDestDir(t *testing.T) {
	requireTar(t)
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "github.com/a/b"), 0o755); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(base, "archive", "a-b.tar.gz")
	if err := archive.CreateArchive(base, "github.com/a/b", dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("expected archive to exist: %v", err)
	}
}

func TestCreateArchiveReportsTarError(t *testing.T) {
	requireTar(t)
	base := t.TempDir()
	dest := filepath.Join(base, "archive", "missing.tar.gz")
	err := archive.CreateArchive(base, "github.com/no/such", dest)
	if err == nil {
		t.Fatal("expected error archiving a missing project")
	}
	if !strings.Contains(err.Error(), "no/such") {
		t.Errorf("expected tar's stderr in the error, got %q", err)
	}
	if _, statErr := os.Stat(dest); statErr == nil {
		t.Error("expected partial archive to be removed")
	}
}

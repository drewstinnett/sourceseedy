//go:build unix

package archive_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/archive"
)

func TestCreateArchiveSkipsSpecialFiles(t *testing.T) {
	base := t.TempDir()
	proj := filepath.Join(base, "github.com", "a", "b")
	write(t, filepath.Join(proj, "file"), "x")
	// Reading a fifo would block forever
	if err := syscall.Mkfifo(filepath.Join(proj, "pipe"), 0o644); err != nil {
		t.Skipf("can't make a fifo: %v", err)
	}
	dest := filepath.Join(base, "out.tar.gz")
	if err := archive.CreateArchive(base, "github.com/a/b", dest); err != nil {
		t.Fatal(err)
	}
	got := readArchive(t, dest)
	if _, ok := got["github.com/a/b/pipe"]; ok {
		t.Error("expected the fifo to be skipped")
	}
	if _, ok := got["github.com/a/b/file"]; !ok {
		t.Error("expected the regular file to be archived")
	}
	_ = os.Remove(dest)
}

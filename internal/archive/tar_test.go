package archive_test

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/archive"
)

// entry is one thing read back out of an archive
type entry struct {
	typ  byte
	body string
	link string
}

func readArchive(t *testing.T, name string) map[string]entry {
	t.Helper()
	f, err := os.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]entry{}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return entries
		}
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		entries[hdr.Name] = entry{typ: hdr.Typeflag, body: string(body), link: hdr.Linkname}
	}
}

func write(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCreateArchive(t *testing.T) {
	base := t.TempDir()
	proj := filepath.Join(base, "github.com", "a", "b")
	write(t, filepath.Join(proj, "README.md"), "hello")
	write(t, filepath.Join(proj, "sub", "dir", "file.txt"), "nested")
	write(t, filepath.Join(proj, ".git", "HEAD"), "ref: refs/heads/main\n")
	write(t, filepath.Join(base, "github.com", "a", "other", "nope.txt"), "not this one")
	if err := os.MkdirAll(filepath.Join(proj, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Symlinks need privileges on Windows, so they are best effort there
	hasLink := os.Symlink("README.md", filepath.Join(proj, "link")) == nil

	// The archive directory doesn't exist yet
	dest := filepath.Join(base, "archive", "a-b.tar.gz")
	if err := archive.CreateArchive(base, "github.com/a/b", dest); err != nil {
		t.Fatal(err)
	}

	got := readArchive(t, dest)
	for name, want := range map[string]entry{
		"github.com/a/b/":                 {typ: tar.TypeDir},
		"github.com/a/b/README.md":        {typ: tar.TypeReg, body: "hello"},
		"github.com/a/b/sub/dir/file.txt": {typ: tar.TypeReg, body: "nested"},
		"github.com/a/b/.git/HEAD":        {typ: tar.TypeReg, body: "ref: refs/heads/main\n"},
		"github.com/a/b/empty/":           {typ: tar.TypeDir},
		"github.com/a/b/sub/":             {typ: tar.TypeDir},
		"github.com/a/b/.git/":            {typ: tar.TypeDir},
		"github.com/a/b/sub/dir/":         {typ: tar.TypeDir},
	} {
		if e, ok := got[name]; !ok || e != want {
			t.Errorf("%s: got %+v (present %v), want %+v", name, e, ok, want)
		}
	}
	if hasLink {
		if e := got["github.com/a/b/link"]; e.typ != tar.TypeSymlink || e.link != "README.md" {
			t.Errorf("expected a symlink to README.md, got %+v", e)
		}
	}
	for name := range got {
		if !strings.HasPrefix(name, "github.com/a/b/") || strings.Contains(name, `\`) || strings.Contains(name, "..") {
			t.Errorf("unexpected entry name %q", name)
		}
	}
}

func TestCreateArchiveMissingProject(t *testing.T) {
	base := t.TempDir()
	dest := filepath.Join(base, "archive", "missing.tar.gz")
	err := archive.CreateArchive(base, "github.com/no/such", dest)
	if err == nil {
		t.Fatal("expected error archiving a missing project")
	}
	if !strings.Contains(err.Error(), "github.com/no/such") || !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected the project and a not-exist cause in the error, got %q", err)
	}
	if _, statErr := os.Stat(dest); statErr == nil {
		t.Error("expected partial archive to be removed")
	}
}

func TestCreateArchiveKeepsSlashNames(t *testing.T) {
	// Callers pass slash separated projects on every OS
	base := t.TempDir()
	write(t, filepath.Join(base, "gitlab.com", "grp", "sub", "repo", "f"), "x")
	dest := filepath.Join(base, "out.tar.gz")
	if err := archive.CreateArchive(base, "gitlab.com/grp/sub/repo", dest); err != nil {
		t.Fatal(err)
	}
	var names []string
	for n := range readArchive(t, dest) {
		names = append(names, n)
	}
	slices.Sort(names)
	want := []string{"gitlab.com/grp/sub/repo/", "gitlab.com/grp/sub/repo/f"}
	if !slices.Equal(names, want) {
		t.Errorf("got %v, want %v", names, want)
	}
}

// Package archive creates compressed archives of projects
package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
)

// CreateArchive writes a gzipped tarball of base/project to dest, creating
// the directory containing dest if needed. Entries are named relative to
// base, like `tar -C base project`. Symlinks are stored as links, and files
// that aren't regular files, directories or symlinks (sockets, devices) are
// skipped. This is done in Go rather than by running tar, since the tar that
// is first on PATH varies too much on Windows: GNU tar takes C:\x for a remote
// host, and bsdtar doesn't know GNU's flags
func CreateArchive(base, project, dest string) (err error) {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			// Don't leave a partial archive behind
			_ = os.Remove(dest)
		}
	}()

	gz := gzip.NewWriter(out)
	tw := tar.NewWriter(gz)
	root := filepath.Join(base, filepath.FromSlash(project))
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		return addEntry(tw, base, p, d)
	})
	if err != nil {
		return fmt.Errorf("archiving %s: %w", project, err)
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

// addEntry writes the file, directory or symlink at p to tw, named relative to base
func addEntry(tw *tar.Writer, base, p string, d fs.DirEntry) error {
	var link string
	switch {
	case d.Type()&fs.ModeSymlink != 0:
		var err error
		if link, err = os.Readlink(p); err != nil {
			return err
		}
	case d.IsDir(), d.Type().IsRegular():
	default:
		slog.Debug("Not archiving special file", "path", p, "mode", d.Type())
		return nil
	}

	info, err := d.Info()
	if err != nil {
		return err
	}
	hdr, err := tar.FileInfoHeader(info, link)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(base, p)
	if err != nil {
		return err
	}
	hdr.Name = filepath.ToSlash(rel)
	if d.IsDir() {
		hdr.Name += "/"
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if !d.Type().IsRegular() {
		return nil
	}

	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = io.Copy(tw, f)
	return err
}

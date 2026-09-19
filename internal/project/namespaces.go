package project

import (
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
)

// Namespace is an owner or group directory under a Host
type Namespace struct {
	Name string
	// Name of the Host containing this namespace
	Host      string
	Directory string
}

// ListProjects returns every git repo under the namespace. A directory
// containing .git is a project, and is not searched any deeper, so working
// trees and nested repos are never walked. Unreadable directories are skipped
func (n Namespace) ListProjects() ([]Project, error) {
	var result []Project
	err := filepath.WalkDir(n.Directory, func(spath string, d fs.DirEntry, errIn error) error {
		if errIn != nil {
			if spath == n.Directory {
				return errIn
			}
			slog.Debug("Skipping unreadable path", "path", spath, "err", errIn)
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if _, err := os.Lstat(filepath.Join(spath, ".git")); err != nil {
			return nil
		}
		item, err := filepath.Rel(n.Directory, spath)
		if err != nil {
			return err
		}
		if item == "." {
			item = ""
		}
		result = append(result, Project{
			Name:      filepath.ToSlash(item),
			Namespace: n.Name,
			Host:      n.Host,
			Directory: spath,
		})
		return filepath.SkipDir
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

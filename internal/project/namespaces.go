package project

import (
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Namespace is an owner or group directory under a Host
type Namespace struct {
	Name string
	// Github, Gitlab, etc
	Host      string
	Directory string
}

// ListProjects returns every git repo under the namespace. Unreadable
// directories are skipped
func (n Namespace) ListProjects() ([]Project, error) {
	var result []Project
	err := filepath.Walk(n.Directory, func(spath string, fi os.FileInfo, errIn error) error {
		if errIn != nil {
			if spath == n.Directory {
				return errIn
			}
			slog.Debug("Skipping unreadable path", "path", spath, "err", errIn)
			return nil
		}
		if fi.Name() == ".git" {
			item := strings.TrimSuffix(spath, "/.git")
			item = strings.TrimPrefix(item, n.Directory)
			item = strings.TrimPrefix(item, "/")

			p := Project{
				Name:      item,
				Namespace: n.Name,
				Host:      n.Host,
				Directory: path.Join(n.Directory, item),
			}
			result = append(result, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

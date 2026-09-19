// Package project discovers hosts, namespaces and projects in a source directory
package project

import (
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Host is a remote git host directory, such as ${base}/github.com
type Host struct {
	Name string
	// Github, Gitlab, etc
	Flavor    string
	Directory string
}

// ListProjects returns the paths, relative to the host directory, of every
// git repo under the host. Unreadable directories are skipped
func (h Host) ListProjects() ([]string, error) {
	var result []string
	err := filepath.Walk(h.Directory, func(path string, fi os.FileInfo, errIn error) error {
		if errIn != nil {
			if path == h.Directory {
				return errIn
			}
			slog.Debug("Skipping unreadable path", "path", path, "err", errIn)
			return nil
		}
		if fi.Name() == ".git" {
			item := strings.TrimSuffix(path, "/.git")
			item = strings.TrimPrefix(item, h.Directory)
			result = append(result, item)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ListHosts returns the host directories in dir. Only non-hidden entries with
// a . in their name (e.g. github.com) are considered hosts
func ListHosts(dir string) ([]Host, error) {
	var hosts []Host
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		// Skip .file
		if strings.HasPrefix(f.Name(), ".") {
			continue
		}
		// Only return hosts with a . in the name
		if strings.Contains(f.Name(), ".") {
			h := Host{
				Name:      f.Name(),
				Directory: path.Join(dir, f.Name()),
			}
			hosts = append(hosts, h)
		}
	}
	return hosts, nil
}

// ListNamespaces returns the non-hidden namespace directories under the host
func (h Host) ListNamespaces() ([]Namespace, error) {
	var result []Namespace
	items, err := os.ReadDir(h.Directory)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if strings.HasPrefix(item.Name(), ".") {
			continue
		}
		n := Namespace{
			Name:      item.Name(),
			Host:      h.Name,
			Directory: path.Join(h.Directory, item.Name()),
		}
		result = append(result, n)
	}
	return result, nil
}

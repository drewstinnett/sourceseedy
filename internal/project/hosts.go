// Package project discovers hosts, namespaces and projects in a source directory
package project

import (
	"os"
	"path"
	"strings"
)

// Host is a remote git host directory, such as ${base}/github.com
type Host struct {
	Name      string
	Directory string
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

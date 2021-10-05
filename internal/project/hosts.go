package project

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Host struct {
	Name string
	// Github, Gitlab, etc
	Flavor    string
	Directory string
}

func (h Host) ListProjects() ([]string, error) {
	var result []string
	err := filepath.Walk(h.Directory,
		filepath.WalkFunc(func(path string, fi os.FileInfo, errIn error) error {
			if fi.Name() == ".git" {
				// fmt.Println("Found " + path)
				item := strings.TrimSuffix(path, "/.git")
				item = strings.TrimPrefix(item, h.Directory)
				result = append(result, item)
				// return io.EOF
			}

			return nil
		}))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func ListHosts(dir string) ([]Host, error) {
	files, err := ioutil.ReadDir(dir)
	var hosts []Host
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

func (h Host) ListNamespaces() ([]Namespace, error) {
	var result []Namespace
	items, err := ioutil.ReadDir(h.Directory)
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

package project

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Namespace struct {
	Name string
	// Github, Gitlab, etc
	Host      string
	Directory string
}

func (n Namespace) ListProjects() ([]Project, error) {
	var result []Project
	err := filepath.Walk(n.Directory,
		filepath.WalkFunc(func(spath string, fi os.FileInfo, errIn error) error {
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
				// return io.EOF
			}

			return nil
		}))
	if err != nil {
		return nil, err
	}
	return result, nil
}

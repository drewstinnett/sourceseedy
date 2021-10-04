package sourceseedy

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

func FindGit(dir string) (result []string, err error) {
	err = filepath.Walk(dir,
		filepath.WalkFunc(func(path string, fi os.FileInfo, errIn error) error {
			if fi.Name() == ".git" {
				// fmt.Println("Found " + path)
				item := strings.TrimSuffix(path, "/.git")
				item = strings.TrimPrefix(item, dir)
				result = append(result, item)
				// return io.EOF
			}

			return nil
		}))

	if err == io.EOF {
		err = nil
	}

	return
}

func GetParentPath(s string) string {
	dir := path.Dir(s)
	return dir
}

func ListAllProjectFullIDs(b string) ([]string, error) {
	var namespaces []Namespace
	hs, err := ListHosts(b)
	if err != nil {
		return nil, err
	}

	for _, h := range hs {
		ns, err := h.ListNamespaces()
		if err != nil {
			return nil, err
		}
		namespaces = append(namespaces, ns...)
	}

	// c := make(chan string)
	c := make(chan []string, len(namespaces))
	var wg sync.WaitGroup
	for _, namespace := range namespaces {
		wg.Add(1)
		go func(namespace Namespace) {
			defer wg.Done()
			projects, err := namespace.ListProjects()
			cobra.CheckErr(err)
			var batch []string
			for _, project := range projects {
				batch = append(batch, project.FullID())
			}
			c <- batch
		}(namespace)
	}
	wg.Wait()
	close(c)
	var results []string
	for item := range c {
		results = append(results, item...)
	}
	return results, nil
}

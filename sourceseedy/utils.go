package sourceseedy

import (
	"io"
	"os"
	"path/filepath"
	"strings"
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

package git

import (
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

func IsLocalGitRepo(rpath string) bool {
	gitPath := path.Join(rpath, ".git")
	fileInfo, err := os.Stat(gitPath)
	if err != nil {
		return false
	}

	if fileInfo.IsDir() {
		return true
	}
	return false
}

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

type SysGitConfig struct {
	Directory string
}

// SysGit system call to git command
func SysGit(c *SysGitConfig, args ...string) error {
	cmd := exec.Command("git", args...)
	if c != nil {
		if c.Directory != "" {
			cmd.Dir = c.Directory
		}
	}
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// Package git provides helpers for finding and running git repositories
package git

import (
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// IsLocalGitRepo returns true if rpath contains a .git directory
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

// FindGit returns the paths, relative to dir, of every git repo under dir.
// Unreadable directories are skipped
func FindGit(dir string) (result []string, err error) {
	err = filepath.Walk(dir, func(path string, fi os.FileInfo, errIn error) error {
		if errIn != nil {
			if path == dir {
				return errIn
			}
			slog.Debug("Skipping unreadable path", "path", path, "err", errIn)
			return nil
		}
		if fi.Name() == ".git" {
			item := strings.TrimSuffix(path, "/.git")
			item = strings.TrimPrefix(item, dir)
			result = append(result, item)
		}
		return nil
	})
	return
}

// SysGitConfig configures how SysGit and SysGitOutput run git
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

// SysGitOutput system call to git command, returning trimmed stdout
func SysGitOutput(c *SysGitConfig, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if c != nil {
		if c.Directory != "" {
			cmd.Dir = c.Directory
		}
	}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

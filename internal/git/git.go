// Package git provides helpers for finding and running git repositories
package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsLocalGitRepo returns true if rpath contains a .git directory
func IsLocalGitRepo(rpath string) bool {
	gitPath := filepath.Join(rpath, ".git")
	fileInfo, err := os.Stat(gitPath)
	return err == nil && fileInfo.IsDir()
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

// Clone runs git clone of remote in to dest. Git's output goes to stderr, so
// stdout stays clean for callers that print a result
func Clone(remote, dest string) error {
	cmd := exec.Command("git", "clone", "--", remote, dest)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone %s: %w", remote, err)
	}
	return nil
}

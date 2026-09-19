// Package util contains small filesystem helpers
package util

import (
	"os"
	"path"
)

// GetParentPath returns the parent directory of s, which is a slash separated
// host/namespace/repo path rather than a filesystem path
func GetParentPath(s string) string {
	return path.Dir(s)
}

// IsDir returns true if rpath exists and is a directory
func IsDir(rpath string) bool {
	fileInfo, err := os.Stat(rpath)
	return err == nil && fileInfo.IsDir()
}

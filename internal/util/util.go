// Package util contains small filesystem helpers
package util

import (
	"os"
	"path"
)

// GetParentPath returns the parent directory of s
func GetParentPath(s string) string {
	dir := path.Dir(s)
	return dir
}

// IsDir returns true if rpath exists and is a directory
func IsDir(rpath string) bool {
	fileInfo, err := os.Stat(rpath)
	if err != nil {
		return false
	}

	if fileInfo.IsDir() {
		return true
	}
	return false
}

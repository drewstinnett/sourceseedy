package util

import (
	"os"
	"path"
)

func GetParentPath(s string) string {
	dir := path.Dir(s)
	return dir
}

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

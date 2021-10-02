package sourceseedy

import (
	"errors"
	"os"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/go-git/go-git/v5"
	giturls "github.com/whilp/git-urls"
)

func IsGitRepo(rpath string) bool {
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

func DetectProperPath(fpath string) (string, error) {
	g, err := git.PlainOpen(fpath)
	if err != nil {
		return "", err
	}
	r, err := g.Remotes()
	if err != nil {
		return "", err
	}

	for _, remote := range r {
		// Only look at origins
		if remote.Config().Name != "origin" {
			continue
		}
		for _, url := range remote.Config().URLs {
			u, err := DetectProperPathFromURL(url)
			if err != nil {
				log.Warning(err)
				continue
			}
			return path.Join(u), nil
		}
	}
	return "", errors.New("CouldNotDetecProperPath")
}

func DetectProperPathFromURL(url string) (string, error) {
	if !strings.Contains(url, "/") {
		return "", errors.New("Missing / in URL")
	}
	u, err := giturls.Parse(url)
	if err != nil {
		return "", err
	}
	upath := strings.TrimSuffix(u.Path, ".git")
	return path.Join(u.Host, upath), nil
}

package project

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	ggit "github.com/go-git/go-git/v5"
	giturls "github.com/whilp/git-urls"
)

type Project struct {
	Name      string
	Host      string
	Namespace string
	Directory string
}

func (p Project) FullID() string {
	return fmt.Sprintf("%v/%v/%v", p.Host, p.Namespace, p.Name)
}

func DetectProperPath(fpath string) (string, error) {
	g, err := ggit.PlainOpen(fpath)
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

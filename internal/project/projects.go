package project

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/drewstinnett/sourceseedy/internal/git"
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
	out, err := git.SysGitOutput(&git.SysGitConfig{Directory: fpath}, "remote", "get-url", "--all", "origin")
	if err != nil {
		return "", err
	}

	for _, remote := range strings.Split(out, "\n") {
		u, err := DetectProperPathFromURL(strings.TrimSpace(remote))
		if err != nil {
			slog.Error("Error detecting path", "err", err)
			continue
		}
		return u, nil
	}
	return "", errors.New("CouldNotDetecProperPath")
}

// DetectProperPathFromURL converts a git remote URL (e.g. https://host/ns/repo.git
// or git@host:ns/repo.git) in to a host/namespace/repo path
func DetectProperPathFromURL(remote string) (string, error) {
	if !strings.Contains(remote, "/") {
		return "", errors.New("Missing / in URL")
	}
	var host, upath string
	switch {
	case strings.Contains(remote, "://"):
		u, err := url.Parse(remote)
		if err != nil {
			return "", err
		}
		host, upath = u.Hostname(), u.Path
	case strings.Contains(remote, ":"):
		// scp-like syntax: [user@]host:path
		host, upath, _ = strings.Cut(remote, ":")
		if _, h, ok := strings.Cut(host, "@"); ok {
			host = h
		}
	default:
		upath = remote
	}
	upath = strings.TrimSuffix(upath, ".git")
	return path.Join(host, upath), nil
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

	c := make(chan []string, len(namespaces))
	errc := make(chan error, len(namespaces))
	var wg sync.WaitGroup
	for _, namespace := range namespaces {
		wg.Add(1)
		go func(namespace Namespace) {
			defer wg.Done()
			projects, err := namespace.ListProjects()
			if err != nil {
				errc <- err
				return
			}
			var batch []string
			for _, project := range projects {
				batch = append(batch, project.FullID())
			}
			c <- batch
		}(namespace)
	}
	wg.Wait()
	close(c)
	close(errc)
	if err := <-errc; err != nil {
		return nil, err
	}
	var results []string
	for item := range c {
		results = append(results, item...)
	}
	return results, nil
}

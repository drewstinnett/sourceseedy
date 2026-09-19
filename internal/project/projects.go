package project

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os/exec"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/drewstinnett/sourceseedy/internal/git"
)

// Project is a single git repo within a Namespace
type Project struct {
	Name      string
	Host      string
	Namespace string
	Directory string
}

// FullID returns the project as host/namespace/name
func (p Project) FullID() string {
	return fmt.Sprintf("%v/%v/%v", p.Host, p.Namespace, p.Name)
}

// DetectProperPath returns the host/namespace/repo path a local git repo
// belongs at, based on its origin remote
func DetectProperPath(fpath string) (string, error) {
	out, err := git.SysGitOutput(&git.SysGitConfig{Directory: fpath}, "remote", "get-url", "--all", "origin")
	if err != nil {
		// git exits 2 when there is no such remote
		if exit := (*exec.ExitError)(nil); errors.As(err, &exit) && exit.ExitCode() == 2 {
			return "", errors.New("no origin remote to place it by")
		}
		return "", fmt.Errorf("getting origin remote: %w", err)
	}

	var lastErr error
	for remote := range strings.SplitSeq(out, "\n") {
		u, err := TargetFromRemote(strings.TrimSpace(remote))
		if err != nil {
			slog.Debug("Origin remote can't be placed", "err", err)
			lastErr = err
			continue
		}
		return u, nil
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", errors.New("no origin remote to place it by")
}

// DetectProperPathFromURL converts a git remote URL (e.g. https://host/ns/repo.git
// or git@host:ns/repo.git) in to a host/namespace/repo path
func DetectProperPathFromURL(remote string) (string, error) {
	if !strings.Contains(remote, "/") {
		return "", errors.New("missing / in URL")
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
	// URL paths always use /, whatever the OS, so this is path not filepath
	return path.Join(host, upath), nil
}

// ListAllNamespaces returns every namespace of every host under base b
func ListAllNamespaces(b string) ([]Namespace, error) {
	hs, err := ListHosts(b)
	if err != nil {
		return nil, err
	}
	var namespaces []Namespace
	for _, h := range hs {
		ns, err := h.ListNamespaces()
		if err != nil {
			return nil, err
		}
		namespaces = append(namespaces, ns...)
	}
	return namespaces, nil
}

// ListAllProjectFullIDs returns the sorted FullID of every project under base b
func ListAllProjectFullIDs(b string) ([]string, error) {
	namespaces, err := ListAllNamespaces(b)
	if err != nil {
		return nil, err
	}

	batches := make([][]string, len(namespaces))
	errs := make([]error, len(namespaces))
	var wg sync.WaitGroup
	for i, namespace := range namespaces {
		wg.Go(func() {
			projects, err := namespace.ListProjects()
			if err != nil {
				errs[i] = err
				return
			}
			for _, project := range projects {
				batches[i] = append(batches[i], project.FullID())
			}
		})
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	results := slices.Concat(batches...)
	slices.Sort(results)
	return results, nil
}

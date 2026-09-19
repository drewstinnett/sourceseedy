package project

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateTarget checks that target, a host/namespace/repo path relative to
// the base directory, stays inside the base and is somewhere sourceseedy will
// find it again
func ValidateTarget(target string) error {
	if !filepath.IsLocal(target) {
		return fmt.Errorf("%q is not a relative path inside the base directory", target)
	}
	parts := strings.Split(target, "/")
	if len(parts) < 3 {
		return fmt.Errorf("%q is not in host/namespace/repo form", target)
	}
	// Same rules ListHosts and Host.ListNamespaces use to find things
	if !strings.Contains(parts[0], ".") || strings.HasPrefix(parts[0], ".") {
		return fmt.Errorf("%q does not start with a host name like github.com", target)
	}
	if strings.HasPrefix(parts[1], ".") {
		return fmt.Errorf("%q has a hidden namespace", target)
	}
	return nil
}

// TargetFromRemote returns the validated host/namespace/repo path that a git
// remote URL belongs at
func TargetFromRemote(remote string) (string, error) {
	target, err := DetectProperPathFromURL(remote)
	if err != nil {
		return "", err
	}
	if err := ValidateTarget(target); err != nil {
		return "", fmt.Errorf("cannot place %q: %w", remote, err)
	}
	return target, nil
}

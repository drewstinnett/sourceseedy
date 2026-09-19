// Package archive creates compressed archives of projects
package archive

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CreateArchive writes a gzipped tarball of base/project to dest, creating
// the directory containing dest if needed. Calls the external tar program.
// All the built in golang ones seem a bit janky 😭
func CreateArchive(base, project, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	cmd := exec.Command("tar", "czf", dest, "-C", base, project)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Don't leave a partial archive behind
		_ = os.Remove(dest)
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("tar: %w: %s", err, msg)
		}
		return fmt.Errorf("tar: %w", err)
	}
	return nil
}

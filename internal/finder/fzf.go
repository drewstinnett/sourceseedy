// Package finder selects projects interactively using fzf
package finder

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/drewstinnett/sourceseedy/internal/project"
)

// ErrNoSelection is returned when fzf exits without a selection, such as when
// the user presses Esc or Ctrl-C
var ErrNoSelection = errors.New("no selection made")

// runFzf runs fzf with args, feeds it lines from feed, and returns the
// selected line
func runFzf(args []string, feed func(w io.Writer)) (string, error) {
	cmd := exec.Command("fzf", args...)
	cmd.Stderr = os.Stderr
	var out strings.Builder
	cmd.Stdout = &out
	in, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}
	go func() {
		feed(in)
		_ = in.Close()
	}()
	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		// 1 is no match, 130 is interrupted (Esc, Ctrl-C)
		if errors.As(err, &exitErr) && (exitErr.ExitCode() == 1 || exitErr.ExitCode() == 130) {
			return "", ErrNoSelection
		}
		return "", fmt.Errorf("running fzf: %w", err)
	}
	return strings.TrimSuffix(out.String(), "\n"), nil
}

// StreamFzfProjects streams projects under base in to fzf as they are found,
// optionally pre-filtered with filter, and returns the selection
func StreamFzfProjects(base, filter string) (string, error) {
	namespaces, err := project.ListAllNamespaces(base)
	if err != nil {
		return "", err
	}
	args := []string{"+m"}
	if filter != "" {
		args = append(args, "-q", filter)
	}
	return runFzf(args, func(w io.Writer) {
		var mu sync.Mutex
		var wg sync.WaitGroup
		for _, namespace := range namespaces {
			wg.Add(1)
			go func() {
				defer wg.Done()
				projects, err := namespace.ListProjects()
				if err != nil {
					slog.Error("Error listing projects", "err", err)
					return
				}
				for _, p := range projects {
					mu.Lock()
					_, err := fmt.Fprintln(w, p.FullID())
					mu.Unlock()
					if err != nil {
						// fzf has already exited
						return
					}
				}
			}()
		}
		wg.Wait()
	})
}

package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Status is the state of one repo's branch and working tree
type Status struct {
	// Branch is empty when HEAD is detached
	Branch   string
	Detached bool
	// Initial is true when the repo has no commits yet
	Initial  bool
	Upstream string
	// UpstreamGone is true when the branch has an upstream that no longer exists
	UpstreamGone bool
	// Ahead and Behind are only meaningful when there is an upstream, and are
	// as of the last fetch
	Ahead, Behind int
	// Changed counts tracked files with staged or unstaged changes
	Changed    int
	Untracked  int
	Conflicted int
}

// ParseStatus reads the output of git status --porcelain=v2 --branch
func ParseStatus(out string) Status {
	var s Status
	var sawAheadBehind bool
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "# branch.oid "):
			s.Initial = strings.TrimPrefix(line, "# branch.oid ") == "(initial)"
		case strings.HasPrefix(line, "# branch.head "):
			if head := strings.TrimPrefix(line, "# branch.head "); head == "(detached)" {
				s.Detached = true
			} else {
				s.Branch = head
			}
		case strings.HasPrefix(line, "# branch.upstream "):
			s.Upstream = strings.TrimPrefix(line, "# branch.upstream ")
		case strings.HasPrefix(line, "# branch.ab "):
			if _, err := fmt.Sscanf(strings.TrimPrefix(line, "# branch.ab "), "+%d -%d", &s.Ahead, &s.Behind); err == nil {
				sawAheadBehind = true
			}
		case strings.HasPrefix(line, "1 "), strings.HasPrefix(line, "2 "):
			s.Changed++
		case strings.HasPrefix(line, "u "):
			s.Conflicted++
		case strings.HasPrefix(line, "? "):
			s.Untracked++
		}
	}
	// git leaves out the ahead/behind line when the upstream is gone
	s.UpstreamGone = s.Upstream != "" && !sawAheadBehind
	return s
}

// Attention returns short descriptions of everything about the repo that
// someone might want to deal with, and nothing when it is clean, pushed, and
// on a branch tracking something
func (s Status) Attention() []string {
	var out []string
	add := func(format string, args ...any) { out = append(out, fmt.Sprintf(format, args...)) }
	if s.Conflicted > 0 {
		add("%d conflicted", s.Conflicted)
	}
	if s.Changed > 0 {
		add("%d changed", s.Changed)
	}
	if s.Untracked > 0 {
		add("%d untracked", s.Untracked)
	}
	if s.Ahead > 0 {
		add("ahead %d", s.Ahead)
	}
	if s.Behind > 0 {
		add("behind %d", s.Behind)
	}
	switch {
	case s.Initial:
		add("no commits")
	case s.Detached:
		add("detached")
	case s.Upstream == "":
		add("no upstream")
	case s.UpstreamGone:
		add("upstream gone")
	}
	return out
}

// RepoStatus runs git status in dir. It doesn't fetch, and doesn't take the
// optional lock on the index, so it is safe to run while someone is using git
func RepoStatus(dir string) (Status, error) {
	// Ask for untracked files explicitly, in case the repo is configured to hide them
	cmd := exec.Command("git", "status", "--porcelain=v2", "--branch", "--untracked-files=normal")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if msg := strings.TrimSpace(string(exitErr.Stderr)); msg != "" {
				return Status{}, fmt.Errorf("git status: %s", msg)
			}
		}
		return Status{}, fmt.Errorf("git status: %w", err)
	}
	return ParseStatus(string(out)), nil
}

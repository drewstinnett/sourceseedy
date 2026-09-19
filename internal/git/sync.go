package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"
)

// SyncAction is what sync can do for a repo, going by its status after a fetch
type SyncAction int

const (
	// SyncUpToDate means the branch has nothing to pick up
	SyncUpToDate SyncAction = iota
	// SyncNoBranch means there is no branch tracking anything to move, because
	// HEAD is detached, there are no commits, or there is no upstream
	SyncNoBranch
	// SyncSkip means the branch is behind, but moving it isn't safe
	SyncSkip
	// SyncFastForward means the branch is behind and can move without risk
	SyncFastForward
)

// SyncAction decides what to do with the checked out branch once the repo has
// been fetched. It only ever picks a fast-forward for a branch that is behind,
// has nothing of its own that isn't pushed, and has a clean working tree, so
// that nothing can be lost or need merging. The reason says why when the
// answer is not to move it
func (s Status) SyncAction() (SyncAction, string) {
	switch {
	case s.Initial:
		return SyncNoBranch, "no commits"
	case s.Detached:
		return SyncNoBranch, "detached"
	case s.Upstream == "":
		return SyncNoBranch, "no upstream"
	case s.UpstreamGone:
		return SyncNoBranch, "upstream gone"
	case s.Behind == 0:
		return SyncUpToDate, ""
	case s.Ahead > 0:
		return SyncSkip, fmt.Sprintf("diverged (ahead %d, behind %d)", s.Ahead, s.Behind)
	case s.Changed > 0 || s.Conflicted > 0:
		return SyncSkip, "uncommitted changes"
	}
	return SyncFastForward, ""
}

// SyncTimeout is how long one git command gets when syncing, so that a repo
// waiting on a network that never answers can't hold up all the others
const SyncTimeout = 2 * time.Minute

// ErrTimeout is what Fetch and FastForward wrap when git didn't finish in time
var ErrTimeout = errors.New("timed out")

// Fetch fetches every remote of the repo in dir, and forgets remote branches
// that were deleted
func Fetch(ctx context.Context, dir string) error {
	return runQuiet(ctx, dir, "fetch", "--all", "--prune", "--quiet")
}

// FastForward moves the current branch of the repo in dir up to its upstream,
// and fails rather than make a merge commit if it can't do that
func FastForward(ctx context.Context, dir string) error {
	return runQuiet(ctx, dir, "merge", "--ff-only", "--quiet", "@{u}")
}

// runQuiet runs git in dir, giving up after SyncTimeout. Git is never allowed
// to ask for a username or password, since many of these run together and
// nobody could answer them. The error carries git's own message
func runQuiet(ctx context.Context, dir string, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, SyncTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	// Killing git leaves a helper like ssh holding its output open, so don't
	// wait on that forever
	cmd.WaitDelay = 5 * time.Second
	_, err := cmd.Output()
	if err == nil {
		return nil
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("git %s: %w after %s", args[0], ErrTimeout, SyncTimeout)
	}
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		if msg := errorSummary(string(exitErr.Stderr)); msg != "" {
			return fmt.Errorf("git %s: %s", args[0], msg)
		}
	}
	return fmt.Errorf("git %s: %w", args[0], err)
}

// boilerplate is what git adds to the end of most network errors, which says
// nothing that the line above it didn't
var boilerplate = []string{
	"Could not read from remote repository.",
	"Please make sure you have the correct access rights",
	"and the repository exists.",
	"could not fetch ",
}

// errorSummary flattens git's stderr to one line, leaving out the lines that
// only repeat that something went wrong, and the "fatal: " in front of the rest
func errorSummary(stderr string) string {
	var keep []string
	for line := range strings.SplitSeq(stderr, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(strings.TrimPrefix(line, "fatal: "), "error: ")
		if line == "" || slices.ContainsFunc(boilerplate, func(b string) bool { return strings.Contains(line, b) }) {
			continue
		}
		keep = append(keep, line)
	}
	return strings.Join(keep, "; ")
}

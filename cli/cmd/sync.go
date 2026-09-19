package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/drewstinnett/sourceseedy/internal/git"
	"github.com/drewstinnett/sourceseedy/internal/project"
)

// What happened to one project in sync, as reported in sync --json
const (
	syncUpdated     = "updated"
	syncWouldUpdate = "would-update"
	syncFetched     = "fetched"
	syncUnchanged   = "unchanged"
	syncSkipped     = "skipped"
	syncFailed      = "failed"
)

// syncReport is one project in sync output, and sync --json
type syncReport struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Result string `json:"result"`
	Branch string `json:"branch"`
	// Commits is how many commits the branch was behind by, which is how many
	// it took in when Result is updated
	Commits int    `json:"commits"`
	Reason  string `json:"reason,omitempty"`
	Error   string `json:"error,omitempty"`
}

// syncMode is what sync is allowed to do to a project after fetching it
type syncMode struct {
	// fetchOnly never moves a branch
	fetchOnly bool
	// dryRun says what would be moved, without moving it
	dryRun bool
}

// fetch is how a project is fetched. Tests swap it
var fetch = git.Fetch

// downAfter is how many projects on a host have to time out, without a single
// one being fetched, before the rest of that host's projects aren't tried. One
// timeout isn't enough, a big repo can take that long on a host that is fine
const downAfter = 3

// hostHealth tracks how fetching is going for each host, so that a host that
// isn't answering doesn't make every one of its projects wait out the timeout
type hostHealth struct {
	mu    sync.Mutex
	hosts map[string]*hostCount
}

type hostCount struct{ fetched, timedOut int }

func (h *hostHealth) count(host string) *hostCount {
	if h.hosts == nil {
		h.hosts = map[string]*hostCount{}
	}
	if h.hosts[host] == nil {
		h.hosts[host] = &hostCount{}
	}
	return h.hosts[host]
}

// down reports whether host has timed out downAfter times and never answered
func (h *hostHealth) down(host string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	c := h.count(host)
	return c.fetched == 0 && c.timedOut >= downAfter
}

func (h *hostHealth) fetched(host string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.count(host).fetched++
}

func (h *hostHealth) timedOut(host string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.count(host).timedOut++
}

func init() {
	var all, asJSON bool
	var mode syncMode
	commands = append(commands, &command{
		name:  "sync",
		usage: "sync [flags]",
		short: "Fetch every project, and fast-forward the ones that are safe to",
		long: `Fetch every project, several at once, and forget remote branches that were
deleted. Then move the branch you have checked out up to its upstream, but only
when that is a fast-forward and the working tree is clean. Nothing is rebased,
merged, or stashed, so a project with local changes or commits that haven't been
pushed is left alone, and reported.

Only projects that were updated, skipped, or failed are listed. Pass --all to
include the rest. The exit status is non-zero when a project failed, and not
when one was skipped.

--dry-run still fetches, since that only updates what git knows about the
remote, but doesn't move any branch. --fetch-only does the same, and says which
projects are behind. Set up ssh-agent first if your remotes ask for a
passphrase, sync can't answer git's prompts`,
		flags: func(fs *flag.FlagSet) {
			for _, name := range []string{"all", "a"} {
				fs.BoolVar(&all, name, false, "Include projects that had nothing to do too")
			}
			for _, name := range []string{"fetch-only", "f"} {
				fs.BoolVar(&mode.fetchOnly, name, false, "Only fetch, never move a branch")
			}
			for _, name := range []string{"dry-run", "d"} {
				fs.BoolVar(&mode.dryRun, name, false, "Fetch, and say what would be updated without updating it")
			}
			fs.BoolVar(&asJSON, "json", false, "Print a JSON array instead of status lines")
		},
		run: func(args []string) error {
			if len(args) > 0 {
				return errors.New("sync accepts no args")
			}
			projects, err := project.ListAllProjects(base)
			if err != nil {
				return err
			}
			reports, err := collectSync(context.Background(), projects, mode)
			if err != nil {
				return err
			}

			shown := make([]syncReport, 0, len(reports))
			for _, r := range reports {
				if all || r.notable() {
					shown = append(shown, r)
				}
			}
			if asJSON {
				if err := writeJSON(shown); err != nil {
					return err
				}
			} else {
				for _, r := range shown {
					r.print()
				}
			}
			return summarizeSync(reports, mode)
		},
	})
}

// notable reports whether r says something without --all: a project that was
// changed, or needs someone to look at it, or is behind after a fetch-only run
func (r syncReport) notable() bool {
	switch r.Result {
	case syncUpdated, syncWouldUpdate, syncSkipped, syncFailed:
		return true
	}
	return r.Commits > 0
}

// print reports r as a status line
func (r syncReport) print() {
	branch := r.Branch
	if branch == "" {
		branch = "-"
	}
	switch r.Result {
	case syncUpdated:
		done("Updated", r.ID+"  "+branch+"  "+plural(r.Commits, "commit"))
	case syncWouldUpdate:
		preview("Would update", r.ID+"  "+branch+"  "+plural(r.Commits, "commit"))
	case syncSkipped:
		warn("Skipped", r.ID+": "+r.Reason)
	case syncFailed:
		warn("Failed", r.ID+": "+r.Error)
	case syncFetched:
		detail := r.ID + "  " + branch
		if r.Reason != "" {
			detail += "  " + r.Reason
		}
		note("Fetched", detail)
	default:
		note("Current", r.ID+"  "+branch)
	}
}

// groupMin is how many projects on one host have to fail for the same reason
// before they are reported as one line
const groupMin = 3

// printSyncReports prints a status line for each of reports. When a host has
// broken for a lot of projects at once, because it is offline or wants a key
// that isn't loaded, that is said once instead of on every line
func printSyncReports(reports []syncReport) {
	type key struct{ host, err string }
	counts := map[key]int{}
	for _, r := range reports {
		if r.Result == syncFailed {
			counts[key{hostOf(r.ID), r.Error}]++
		}
	}
	printed := map[key]bool{}
	for _, r := range reports {
		k := key{hostOf(r.ID), r.Error}
		if r.Result != syncFailed || counts[k] < groupMin {
			r.print()
			continue
		}
		if !printed[k] {
			printed[k] = true
			warn("Failed", fmt.Sprintf("%s: %s (%d projects)", k.host, k.err, counts[k]))
		}
	}
}

// hostOf is the host part of a project id like host/namespace/name
func hostOf(id string) string {
	host, _, _ := strings.Cut(id, "/")
	return host
}

// collectSync syncs every project, a few at a time, and returns the results in
// the same order as projects
func collectSync(ctx context.Context, projects []project.Project, mode syncMode) ([]syncReport, error) {
	var hosts hostHealth
	reports := eachProject(projects, func(p project.Project) syncReport {
		return syncProject(ctx, p, mode, &hosts)
	})
	for i, p := range projects {
		abs, err := filepath.Abs(p.Directory)
		if err != nil {
			return nil, err
		}
		reports[i].Path = abs
	}
	return reports, nil
}

// syncProject fetches p and moves its branch if mode and the state of the repo
// allow it. A repo git can't fetch or move is reported with its error rather
// than stopping the whole run
func syncProject(ctx context.Context, p project.Project, mode syncMode, hosts *hostHealth) syncReport {
	r := syncReport{ID: p.FullID(), Result: syncUnchanged}
	fail := func(err error) syncReport {
		r.Result, r.Error = syncFailed, err.Error()
		return r
	}
	if hosts.down(p.Host) {
		return fail(errors.New(p.Host + " isn't answering, so it wasn't tried"))
	}
	if err := fetch(ctx, p.Directory); err != nil {
		if errors.Is(err, git.ErrTimeout) {
			hosts.timedOut(p.Host)
		}
		return fail(err)
	}
	hosts.fetched(p.Host)
	s, err := git.RepoStatus(p.Directory)
	if err != nil {
		return fail(err)
	}
	r.Branch = s.Branch

	action, reason := s.SyncAction()
	switch action {
	case git.SyncUpToDate:
		return r
	case git.SyncNoBranch:
		r.Result, r.Reason = syncFetched, reason
		return r
	}

	r.Commits = s.Behind
	switch {
	case mode.fetchOnly:
		r.Result, r.Reason = syncFetched, fmt.Sprintf("behind %d", s.Behind)
	case action == git.SyncSkip:
		r.Result, r.Reason = syncSkipped, reason
	case mode.dryRun:
		r.Result = syncWouldUpdate
	default:
		if err := git.FastForward(ctx, p.Directory); err != nil {
			r.Commits = 0
			return fail(err)
		}
		r.Result = syncUpdated
	}
	return r
}

// summarizeSync says how it went overall, and returns an error if any project
// failed, so that scripts can tell
func summarizeSync(reports []syncReport, mode syncMode) error {
	counts := map[string]int{}
	for _, r := range reports {
		counts[r.Result]++
	}

	verb := "Synced"
	switch {
	case mode.dryRun:
		verb = "Checked"
	case mode.fetchOnly:
		verb = "Fetched"
	}
	detail := plural(len(reports), "project")

	var parts []string
	for _, c := range []struct{ result, label string }{
		{syncUpdated, "updated"},
		{syncWouldUpdate, "would update"},
		{syncSkipped, "skipped"},
		{syncFailed, "failed"},
	} {
		if n := counts[c.result]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, c.label))
		}
	}
	if behind := behindCount(reports); behind > 0 && mode.fetchOnly {
		parts = append(parts, fmt.Sprintf("%d behind", behind))
	}
	if len(parts) == 0 {
		done(verb, detail+", all up to date")
		return nil
	}
	detail += ", " + strings.Join(parts, ", ")
	if counts[syncSkipped]+counts[syncFailed] > 0 {
		warn(verb, detail)
	} else {
		done(verb, detail)
	}
	if n := counts[syncFailed]; n > 0 {
		return errors.New(plural(n, "project") + " failed")
	}
	return nil
}

// behindCount is how many projects a fetch-only run found to be behind
func behindCount(reports []syncReport) int {
	n := 0
	for _, r := range reports {
		if r.Result == syncFetched && r.Commits > 0 {
			n++
		}
	}
	return n
}

// plural formats n and noun, like "1 commit" or "3 commits"
func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

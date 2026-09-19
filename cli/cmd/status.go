package cmd

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/drewstinnett/sourceseedy/internal/git"
	"github.com/drewstinnett/sourceseedy/internal/project"
)

// statusWorkers is how many repos are checked at once. Each check runs git, so
// this follows the number of CPUs, but stays within limits so that a big tree
// can't start hundreds of processes together, or crawl on a small machine
func statusWorkers() int {
	return min(max(runtime.NumCPU(), 4), 16)
}

// statusReport is one project in status output, and status --json
type statusReport struct {
	ID         string   `json:"id"`
	Path       string   `json:"path"`
	Branch     string   `json:"branch"`
	Detached   bool     `json:"detached"`
	Upstream   string   `json:"upstream"`
	Ahead      int      `json:"ahead"`
	Behind     int      `json:"behind"`
	Changed    int      `json:"changed"`
	Untracked  int      `json:"untracked"`
	Conflicted int      `json:"conflicted"`
	Clean      bool     `json:"clean"`
	Problems   []string `json:"problems"`
	Error      string   `json:"error,omitempty"`
}

func init() {
	var all, asJSON bool
	commands = append(commands, &command{
		name:  "status",
		usage: "status [flags]",
		short: "Show projects with uncommitted or unpushed work",
		long: `Check every project, and list the ones that need attention: uncommitted or
untracked files, commits that aren't pushed, a branch with no upstream, a
detached HEAD. Clean projects are left out unless you pass --all.

This doesn't fetch, so "behind" is as of the last time you fetched`,
		flags: func(fs *flag.FlagSet) {
			for _, name := range []string{"all", "a"} {
				fs.BoolVar(&all, name, false, "Include clean projects too")
			}
			fs.BoolVar(&asJSON, "json", false, "Print a JSON array instead of a table")
		},
		run: func(args []string) error {
			if len(args) > 0 {
				return errors.New("status accepts no args")
			}
			projects, err := project.ListAllProjects(base)
			if err != nil {
				return err
			}
			reports, err := collectStatus(projects)
			if err != nil {
				return err
			}

			shown := make([]statusReport, 0, len(reports))
			for _, r := range reports {
				if all || !r.Clean {
					shown = append(shown, r)
				}
			}
			if asJSON {
				err = writeJSON(shown)
			} else {
				err = writeStatusTable(shown)
			}
			if err != nil {
				return err
			}

			noun := "projects"
			if len(reports) == 1 {
				noun = "project"
			}
			if n := len(reports) - countClean(reports); n > 0 {
				warn("Checked", fmt.Sprintf("%d %s, %d need attention", len(reports), noun, n))
			} else {
				done("Checked", fmt.Sprintf("%d %s, all clean", len(reports), noun))
			}
			return nil
		},
	})
}

func countClean(reports []statusReport) int {
	n := 0
	for _, r := range reports {
		if r.Clean {
			n++
		}
	}
	return n
}

// collectStatus checks every project, a few at a time, and returns the results
// in the same order as projects
func collectStatus(projects []project.Project) ([]statusReport, error) {
	reports := make([]statusReport, len(projects))
	sem := make(chan struct{}, statusWorkers())
	var wg sync.WaitGroup
	for i, p := range projects {
		sem <- struct{}{}
		wg.Go(func() {
			defer func() { <-sem }()
			reports[i] = checkProject(p)
		})
	}
	wg.Wait()
	for i, p := range projects {
		abs, err := filepath.Abs(p.Directory)
		if err != nil {
			return nil, err
		}
		reports[i].Path = abs
	}
	return reports, nil
}

// checkProject runs git status for p. A repo git can't read is reported with
// its error rather than stopping the whole run
func checkProject(p project.Project) statusReport {
	r := statusReport{ID: p.FullID(), Problems: []string{}}
	s, err := git.RepoStatus(p.Directory)
	if err != nil {
		r.Error = err.Error()
		return r
	}
	r.Branch, r.Detached, r.Upstream = s.Branch, s.Detached, s.Upstream
	r.Ahead, r.Behind = s.Ahead, s.Behind
	r.Changed, r.Untracked, r.Conflicted = s.Changed, s.Untracked, s.Conflicted
	r.Problems = append(r.Problems, s.Attention()...)
	r.Clean = len(r.Problems) == 0
	return r
}

// writeStatusTable prints one aligned line per project: id, branch, and what is
// wrong with it
func writeStatusTable(reports []statusReport) error {
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	for _, r := range reports {
		branch := r.Branch
		switch {
		case r.Error != "":
			branch = "-"
		case r.Detached:
			branch = "(detached)"
		case branch == "":
			branch = "-"
		}
		state := "clean"
		switch {
		case r.Error != "":
			state = "error: " + r.Error
		case len(r.Problems) > 0:
			state = strings.Join(r.Problems, ", ")
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", r.ID, branch, state)
	}
	return tw.Flush()
}

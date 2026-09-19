package cmd

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/drewstinnett/sourceseedy/internal/git"
	"github.com/drewstinnett/sourceseedy/internal/project"
	"github.com/drewstinnett/sourceseedy/internal/util"
)

func init() {
	var dr bool
	commands = append(commands, &command{
		name:  "import",
		usage: "import [flags] <repo>...",
		short: "Import a new git repo in to your structure",
		long: `Got a repo somewhere outside of the standard structure? Use this command to
import and move it over. Use the git remote URL to decide where it should go.
Remote URLs are cloned straight in to place, same as the clone command. The path
of each imported repo is printed to stdout when it is not a terminal, so this
works: cd "$(sourceseedy import https://github.com/a/b.git)"`,
		flags: dryRunFlag(&dr),
		run: func(args []string) error {
			if len(args) < 1 {
				return errors.New("import requires at least 1 arg")
			}
			for _, item := range args {
				if !git.IsLocalGitRepo(item) {
					slog.Debug("Not found on host, attempting to clone it", "gitrepo", item)
					dest, err := cloneRemote(item, dr)
					if err != nil {
						return err
					}
					if !dr {
						emitPath(dest)
					}
					continue
				}
				if err := importLocal(item, dr); err != nil {
					return err
				}
			}
			return nil
		},
	})
}

// importLocal moves the local git repo at item in to its proper place under
// base and reports what happened. Nothing is moved when dryRun is set. A repo
// that can't be placed, or whose place is taken, is skipped rather than failed
func importLocal(item string, dryRun bool) error {
	target, err := project.DetectProperPath(item)
	if err != nil {
		warn("Skipped", tildePath(item)+": "+err.Error())
		return nil
	}
	fullTarget := filepath.Join(base, target)
	if util.IsDir(fullTarget) {
		warn("Skipped", tildePath(item)+": "+tildePath(fullTarget)+" already exists")
		return nil
	}
	if dryRun {
		preview("Would move", tildePath(item)+" → "+tildePath(fullTarget))
		return nil
	}
	fullPpath := filepath.Join(base, util.GetParentPath(target))
	if !util.IsDir(fullPpath) {
		slog.Debug("Creating parent path", "path", fullPpath)
		if err := os.MkdirAll(fullPpath, os.ModePerm); err != nil {
			return err
		}
	}
	if err := os.Rename(item, fullTarget); err != nil {
		return err
	}
	done("Moved", tildePath(item)+" → "+tildePath(fullTarget))
	emitPath(fullTarget)
	return nil
}

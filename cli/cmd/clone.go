package cmd

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/drewstinnett/sourceseedy/internal/git"
	"github.com/drewstinnett/sourceseedy/internal/project"
	"github.com/drewstinnett/sourceseedy/internal/util"
)

func init() {
	var dr bool
	commands = append(commands, &command{
		name:  "clone",
		usage: "clone [flags] <repo>...",
		short: "Clone a git repo straight in to your structure",
		long: `Clone a remote git repo in to ${base}/${remote-host}/${namespace}/${repo}, using
the git remote URL to decide where it goes. The path of each repo is printed to
stdout, so this works: cd "$(sourceseedy clone git@github.com:a/b.git)"`,
		flags: dryRunFlag(&dr),
		run: func(args []string) error {
			if len(args) < 1 {
				return errors.New("clone requires at least 1 arg")
			}
			if dr {
				slog.Info("Running in dry-run mode!")
			}
			for _, remote := range args {
				dest, err := cloneRemote(remote, dr)
				if err != nil {
					return err
				}
				if !dr {
					fmt.Println(dest)
				}
			}
			return nil
		},
	})
}

// dryRunFlag returns a flags func registering -d/--dry-run in to dr
func dryRunFlag(dr *bool) func(fs *flag.FlagSet) {
	return func(fs *flag.FlagSet) {
		for _, name := range []string{"dry-run", "d"} {
			fs.BoolVar(dr, name, false, "Just do a dry run, don't actually clone or import")
		}
	}
}

// cloneRemote clones remote in to its proper place under base and returns
// that path. Nothing is cloned when dryRun is set, or the path already exists
func cloneRemote(remote string, dryRun bool) (string, error) {
	target, err := project.TargetFromRemote(remote)
	if err != nil {
		return "", err
	}
	dest := filepath.Join(base, filepath.FromSlash(target))
	if util.IsDir(dest) {
		slog.Info("Target dir already exists", "path", dest)
		return dest, nil
	}
	slog.Info("Cloning", "repo", remote, "path", dest)
	if dryRun {
		return dest, nil
	}
	if err := git.Clone(remote, dest); err != nil {
		return "", err
	}
	return dest, nil
}

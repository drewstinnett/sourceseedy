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
Remote URLs are cloned straight in to place, same as the clone command`,
		flags: dryRunFlag(&dr),
		run: func(args []string) error {
			if len(args) < 1 {
				return errors.New("import requires at least 1 arg")
			}
			if dr {
				slog.Info("Running in dry-run mode!")
			}
			for _, item := range args {
				if !git.IsLocalGitRepo(item) {
					slog.Debug("Not found on host, attempting to clone it", "gitrepo", item)
					if _, err := cloneRemote(item, dr); err != nil {
						return err
					}
					continue
				}
				target, err := project.DetectProperPath(item)
				if err != nil {
					slog.Warn("Could not detect path", "repo", item, "err", err)
					continue
				}
				ppath := util.GetParentPath(target)
				fullPpath := filepath.Join(base, ppath)
				slog.Info("Importing", "repo", item)
				if !util.IsDir(fullPpath) {
					slog.Info("Creating parent path", "path", fullPpath)
					if !dr {
						if err := os.MkdirAll(fullPpath, os.ModePerm); err != nil {
							return err
						}
					}
				}
				fullTarget := filepath.Join(base, target)
				if util.IsDir(fullTarget) {
					slog.Info("Target dir already exists", "path", fullTarget)
					continue
				}

				if !dr {
					if err := os.Rename(item, fullTarget); err != nil {
						return err
					}
					slog.Info("Imported source", "directory", fullTarget)
				}
			}
			return nil
		},
	})
}

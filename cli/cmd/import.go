package cmd

import (
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/exec"
	"path"

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
import and move it over. Use the git remote URL to decide where it should go`,
		flags: func(fs *flag.FlagSet) {
			for _, name := range []string{"dry-run", "d"} {
				fs.BoolVar(&dr, name, false, "Just do a dry run, don't actually import")
			}
		},
		run: func(args []string) error {
			if len(args) < 1 {
				return errors.New("import requires at least 1 arg")
			}
			if dr {
				slog.Info("Running in dry-run mode!")
			}
			for _, item := range args {
				origItem := item
				if !git.IsLocalGitRepo(item) {
					slog.Debug("Not found on host, attempting to clone it", "gitrepo", item)
					dir, err := os.MkdirTemp("", "sourceseedy")
					if err != nil {
						return err
					}
					defer func() {
						if err := os.RemoveAll(dir); err != nil {
							slog.Warn("Could not clean up temp dir", "dir", dir, "err", err)
						}
					}()
					if err := exec.Command("git", "clone", item, dir).Run(); err != nil {
						return err
					}
					item = dir
				}
				target, err := project.DetectProperPath(item)
				if err != nil {
					slog.Warn("Could not detect path")
					continue
				}
				ppath := util.GetParentPath(target)
				fullPpath := path.Join(base, ppath)
				slog.Info("Importing", "repo", origItem)
				if !util.IsDir(fullPpath) {
					slog.Info("Creating parent path", "path", fullPpath)
					if !dr {
						if err := os.MkdirAll(fullPpath, os.ModePerm); err != nil {
							return err
						}
					}
				}
				fullTarget := path.Join(base, target)
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

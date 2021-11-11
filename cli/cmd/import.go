package cmd

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"

	"github.com/apex/log"
	"github.com/drewstinnett/sourceseedy/internal/git"
	"github.com/drewstinnett/sourceseedy/internal/project"
	"github.com/drewstinnett/sourceseedy/internal/util"
	"github.com/spf13/cobra"
)

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import a new git repo in to your structure",
	Long: `Got a repo somewhere outside of the standard structure? Use this command to
import and move it over. Use the git remote URL to decide where it should go`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dr, err := cmd.PersistentFlags().GetBool("dry-run")
		cobra.CheckErr(err)

		if dr {
			log.Info("Running in dry-run mode!")
		}
		for _, item := range args {
			origItem := item
			if !git.IsLocalGitRepo(item) {
				log.WithFields(log.Fields{
					"gitrepo": item,
				}).Debug("Not found on host, attempting to clone it")
				dir, err := ioutil.TempDir("", "sourceseedy")
				if err != nil {
					cobra.CheckErr(err)
				}
				cmd := exec.Command("git", "clone", item, dir)
				err = cmd.Run()
				cobra.CheckErr(err)
				item = dir
				defer os.RemoveAll(dir)
			}
			target, err := project.DetectProperPath(item)
			if err != nil {
				log.WithFields(log.Fields{}).Warn("Could not detect path")
				continue
			}
			ppath := util.GetParentPath(target)
			fullPpath := path.Join(base, ppath)
			log.WithFields(log.Fields{
				"repo": origItem,
			}).Info("Importing")
			if !util.IsDir(fullPpath) {
				log.WithFields(log.Fields{
					"path": fullPpath,
				}).Info("Creating parent path")
				if !dr {
					err := os.MkdirAll(fullPpath, os.ModePerm)
					cobra.CheckErr(err)
				}
			}
			fullTarget := path.Join(base, target)
			if util.IsDir(fullTarget) {
				log.WithFields(log.Fields{
					"path": fullTarget,
				}).Info("Target dir already exists")
				continue
			}

			if !dr {
				err := os.Rename(item, fullTarget)
				cobra.CheckErr(err)
				log.WithFields(log.Fields{
					"directory": fullTarget,
				}).Info("Imported source")
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(importCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// importCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	importCmd.PersistentFlags().BoolP("dry-run", "d", false, "Just do a dry run, don't actually import")
}

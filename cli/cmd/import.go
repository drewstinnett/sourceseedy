package cmd

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/drewstinnett/sourceseedy/internal/git"
	"github.com/drewstinnett/sourceseedy/internal/project"
	"github.com/drewstinnett/sourceseedy/internal/util"
	ggit "github.com/go-git/go-git/v5"
	log "github.com/sirupsen/logrus"
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
			log.Warning("Running in dry-run mode!")
		}
		for _, item := range args {
			if !git.IsLocalGitRepo(item) {
				log.Debugf("%v is not a git repo, attempting to clone it", item)
				dir, err := ioutil.TempDir("", "sourceseedy")
				if err != nil {
					cobra.CheckErr(err)
				}
				log.Println(dir)
				_, err = ggit.PlainClone(dir, false, &ggit.CloneOptions{
					URL:      item,
					Progress: os.Stdout,
				})
				cobra.CheckErr(err)
				item = dir
				defer os.RemoveAll(dir)
			}
			target, err := project.DetectProperPath(item)
			if err != nil {
				log.Warning("Could not detect path of", item)
				continue
			}
			ppath := util.GetParentPath(target)
			fullPpath := path.Join(base, ppath)
			log.Printf("Importing %v → %v", item, target)
			if !util.IsDir(fullPpath) {
				log.Println("Creating: ", fullPpath)
				if !dr {
					err := os.MkdirAll(fullPpath, os.ModePerm)
					cobra.CheckErr(err)
				}
			}
			fullTarget := path.Join(base, target)
			if util.IsDir(fullTarget) {
				log.Warningf("Target dir %v already exists", fullTarget)
				continue
			}

			if !dr {
				err := os.Rename(item, fullTarget)
				cobra.CheckErr(err)
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

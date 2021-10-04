package cmd

import (
	"os"
	"path"

	log "github.com/sirupsen/logrus"

	"github.com/drewstinnett/sourceseedy/sourceseedy"
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
			if !sourceseedy.IsGitRepo(item) {
				log.Warningf("%v is not a git repo", item)
				continue
			}
			target, err := sourceseedy.DetectProperPath(item)
			if err != nil {
				log.Warning("Could not detect path of", item)
				continue
			}
			ppath := sourceseedy.GetParentPath(target)
			fullPpath := path.Join(base, ppath)
			log.Printf("Importing %v → %v", item, target)
			if !sourceseedy.IsDir(fullPpath) {
				log.Println("Creating: ", fullPpath)
				if !dr {
					err := os.MkdirAll(fullPpath, os.ModePerm)
					cobra.CheckErr(err)
				}
			}
			fullTarget := path.Join(base, target)
			if sourceseedy.IsDir(fullTarget) {
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

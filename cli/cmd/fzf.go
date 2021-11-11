package cmd

import (
	"fmt"
	"path"

	"github.com/drewstinnett/sourceseedy/internal/finder"
	"github.com/spf13/cobra"
)

// fzfCmd represents the fzf command
var fzfCmd = &cobra.Command{
	Use:   "fzf [initial filter]",
	Short: "Use Fzf to jump in to a source directory",
	Long: `Quick method of jumping around source directories, using Fzf. Throw something
like this in your .zshrc for easier usage:

scd() {
  target=$(/usr/local/bin/sourceseedy fzf)
  cd $target
}

If given a filter arg, the fzf command will pass that in as an initial string to
match`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var thing string
		var err error
		if len(args) > 0 {
			thing, err = finder.StreamFzfProjects(base, args[0])
		} else {
			thing, err = finder.StreamFzfProjects(base, "")
		}
		cobra.CheckErr(err)
		fmt.Println(path.Join(base, thing))
	},
}

func init() {
	rootCmd.AddCommand(fzfCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// fzfCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// fzfCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

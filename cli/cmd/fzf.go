/*
Copyright © 2021 Drew Stinnett <drew@drewlink.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"path"

	"github.com/drewstinnett/sourceseedy/sourceseedy"
	"github.com/spf13/cobra"
)

// fzfCmd represents the fzf command
var fzfCmd = &cobra.Command{
	Use:   "fzf [filter]",
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
			thing, err = sourceseedy.StreamFzfProjects(base, args[0])
		} else {
			thing, err = sourceseedy.StreamFzfProjects(base, "")
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

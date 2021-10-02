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
	"bytes"
	"fmt"
	"path"

	"github.com/drewstinnett/sourceseedy/sourceseedy"
	"github.com/spf13/cobra"
)

// fzfCmd represents the fzf command
var fzfCmd = &cobra.Command{
	Use:   "fzf",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		projects, err := sourceseedy.ListAllProjectFullIDs(base)
		cobra.CheckErr(err)
		r := new(bytes.Buffer)
		var thing string
		for _, p := range projects {
			line := fmt.Sprintf(p + "\n")
			r.Write([]byte(line))

		}
		thing, err = sourceseedy.Fzf(r)
		cobra.CheckErr(err)
		fmt.Println(path.Join(base, thing))
		// r := strings.NewReader("")
		/*
			r := new(bytes.Buffer)
			var thing string
			for _, h := range hs {
				projects, err := h.ListProjects()
				cobra.CheckErr(err)
				for _, d := range projects {
					line := fmt.Sprintf(path.Join(h.Name, d) + "\n")
					r.Write([]byte(line))
				}
			}
			thing, err = sourceseedy.Fzf(r)
			cobra.CheckErr(err)
			fmt.Println(path.Join(base, thing))
		*/
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

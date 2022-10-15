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
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/drewstinnett/sourceseedy/internal/archive"
	"github.com/drewstinnett/sourceseedy/internal/finder"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// archiveCmd represents the archive command
var archiveCmd = &cobra.Command{
	Use:   "archive [directory]",
	Short: "Compress and copy a repo in to the 'archive' directory",
	Long: `Use this if you are gonna make a big scary change, and wanna make sure you have
a copy of everything stashed in ti ${base}/archived. If no directory is specified, an fzf
chooser will pop up`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var project, repo string
		var err error
		if len(args) == 0 {
			project, err = finder.FzfProjects(base)
			cobra.CheckErr(err)
			// repo = path.Join(base, project)
		} else {
			repo, err = filepath.Abs(args[0])
			cobra.CheckErr(err)
			project = strings.TrimPrefix(repo, base+"/")
		}

		archiveFilename := path.Join(base, "archive", strings.ReplaceAll(project, "/", "-")+"-"+time.Now().Format("20060102150405")+".tar")
		gzName := archiveFilename + ".gz"
		err = archive.CreateArchive(base, project, gzName)
		cobra.CheckErr(err)
		log.Info().Str("archive", gzName).Msg("Created archive")
	},
}

func init() {
	rootCmd.AddCommand(archiveCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// archiveCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// archiveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

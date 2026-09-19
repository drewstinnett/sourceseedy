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
	"errors"
	"log/slog"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/drewstinnett/sourceseedy/internal/archive"
	"github.com/drewstinnett/sourceseedy/internal/finder"
)

func init() {
	commands = append(commands, &command{
		name:  "archive",
		usage: "archive [directory]",
		short: "Compress and copy a repo in to the 'archive' directory",
		long: `Use this if you are gonna make a big scary change, and wanna make sure you have
a copy of everything stashed in ti ${base}/archived. If no directory is specified, an fzf
chooser will pop up`,
		run: func(args []string) error {
			if len(args) > 1 {
				return errors.New("archive accepts at most 1 arg")
			}
			var project string
			if len(args) == 0 {
				var err error
				project, err = finder.FzfProjects(base)
				if err != nil {
					return err
				}
			} else {
				repo, err := filepath.Abs(args[0])
				if err != nil {
					return err
				}
				project = strings.TrimPrefix(repo, base+"/")
			}

			archiveFilename := path.Join(base, "archive", strings.ReplaceAll(project, "/", "-")+"-"+time.Now().Format("20060102150405")+".tar")
			gzName := archiveFilename + ".gz"
			if err := archive.CreateArchive(base, project, gzName); err != nil {
				return err
			}
			slog.Info("Created archive", "archive", gzName)
			return nil
		},
	})
}

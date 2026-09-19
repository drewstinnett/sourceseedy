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
	"flag"
	"fmt"
	"path/filepath"

	"github.com/drewstinnett/sourceseedy/internal/project"
)

// listEntry is one project in list --json
type listEntry struct {
	ID        string `json:"id"`
	Host      string `json:"host"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Path      string `json:"path"`
}

func init() {
	var asJSON, fullPath bool
	commands = append(commands, &command{
		name:  "list",
		usage: "list [flags]",
		short: "List projects in your source directory",
		long: `List every project, one per line, as ${remote-host}/${namespace}/${repo}. Use
--full-path to get absolute directories instead, or --json for scripts`,
		flags: func(fs *flag.FlagSet) {
			fs.BoolVar(&fullPath, "full-path", false, "Print the absolute path of each project")
			fs.BoolVar(&asJSON, "json", false, "Print a JSON array of projects, with id, host, namespace, name and path")
		},
		run: func(args []string) error {
			if len(args) > 0 {
				return errors.New("list accepts no args")
			}
			projects, err := project.ListAllProjects(base)
			if err != nil {
				return err
			}
			entries := make([]listEntry, len(projects))
			for i, p := range projects {
				abs, err := filepath.Abs(p.Directory)
				if err != nil {
					return err
				}
				entries[i] = listEntry{ID: p.FullID(), Host: p.Host, Namespace: p.Namespace, Name: p.Name, Path: abs}
			}
			if asJSON {
				return writeJSON(entries)
			}
			for _, e := range entries {
				if fullPath {
					fmt.Fprintln(stdout, e.Path)
				} else {
					fmt.Fprintln(stdout, e.ID)
				}
			}
			return nil
		},
	})
}

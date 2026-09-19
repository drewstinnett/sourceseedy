package cmd

import (
	"errors"
	"fmt"
	"path"

	"github.com/drewstinnett/sourceseedy/internal/finder"
)

func init() {
	commands = append(commands, &command{
		name:  "fzf",
		usage: "fzf [initial filter]",
		short: "Use Fzf to jump in to a source directory",
		long: `Quick method of jumping around source directories, using Fzf. Throw something
like this in your .zshrc for easier usage:

scd() {
  target=$(/usr/local/bin/sourceseedy fzf "$@") && cd "$target"
}

If given a filter arg, the fzf command will pass that in as an initial string to
match`,
		run: func(args []string) error {
			if len(args) > 1 {
				return errors.New("fzf accepts at most 1 arg")
			}
			var filter string
			if len(args) > 0 {
				filter = args[0]
			}
			thing, err := finder.StreamFzfProjects(base, filter)
			if err != nil {
				return err
			}
			fmt.Println(path.Join(base, thing))
			return nil
		},
	})
}

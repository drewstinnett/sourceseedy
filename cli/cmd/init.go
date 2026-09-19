package cmd

import (
	"errors"
	"flag"
	"fmt"
	"regexp"
	"strings"
)

const posixInit = `%[1]s() {
  local target
  target=$(command sourceseedy fzf "$@") && cd "$target"
}
`

const fishInit = `function %[1]s
    set -l target (command sourceseedy fzf $argv)
    and test -n "$target"
    and cd $target
end
`

var funcName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func init() {
	var name string
	commands = append(commands, &command{
		name:  "init",
		usage: "init [flags] <bash|zsh|fish>",
		short: "Print shell integration for jumping to projects",
		long: `Prints a shell function that uses fzf to pick a project and cd's to it. The
function is called scd unless you pick another name. Add this to your shell
config:

  bash, zsh: eval "$(sourceseedy init zsh)"
  fish:      sourceseedy init fish | source

Give the function an initial filter with: scd myproj`,
		flags: func(fs *flag.FlagSet) {
			fs.StringVar(&name, "name", "scd", "Name of the shell function to define")
		},
		run: func(args []string) error {
			if len(args) != 1 {
				return errors.New("init requires exactly 1 arg: bash, zsh or fish")
			}
			out, err := shellInit(args[0], name)
			if err != nil {
				return err
			}
			fmt.Print(out)
			return nil
		},
	})
}

// shellInit returns the shell code defining a function called name that
// cd's to a project chosen with fzf
func shellInit(shell, name string) (string, error) {
	if !funcName.MatchString(name) {
		return "", fmt.Errorf("invalid function name %q", name)
	}
	switch strings.ToLower(shell) {
	case "bash", "zsh":
		return fmt.Sprintf(posixInit, name), nil
	case "fish":
		return fmt.Sprintf(fishInit, name), nil
	}
	return "", fmt.Errorf("unsupported shell %q, use bash, zsh or fish", shell)
}

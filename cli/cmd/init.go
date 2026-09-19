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

// The encoding dance is because PowerShell decodes a program's output with the
// console code page, which mangles non-ASCII characters in a path unless it is
// UTF-8. Setting it can fail when there is no console, which is fine
const powershellInit = `function %[1]s {
    $exe = Get-Command sourceseedy -CommandType Application -ErrorAction Stop | Select-Object -First 1
    $prev = [Console]::OutputEncoding
    try { [Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false) } catch {}
    try {
        $target = & $exe.Source fzf @args
        $ok = ($LASTEXITCODE -eq 0)
    } finally {
        try { [Console]::OutputEncoding = $prev } catch {}
    }
    if ($ok -and $target) { Set-Location -LiteralPath $target }
}
`

var funcName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func init() {
	var name string
	commands = append(commands, &command{
		name:  "init",
		usage: "init [flags] <bash|zsh|fish|powershell>",
		short: "Print shell integration for jumping to projects",
		long: `Prints a shell function that uses fzf to pick a project and cd's to it. The
function is called scd unless you pick another name. Add this to your shell
config:

  bash, zsh:  eval "$(sourceseedy init zsh)"
  fish:       sourceseedy init fish | source
  powershell: sourceseedy init powershell | Out-String | Invoke-Expression

For PowerShell, put that line in your profile (run: notepad $PROFILE). pwsh is
accepted as a name for powershell.

Give the function an initial filter with: scd myproj`,
		flags: func(fs *flag.FlagSet) {
			fs.StringVar(&name, "name", "scd", "Name of the shell function to define")
		},
		run: func(args []string) error {
			if len(args) != 1 {
				return errors.New("init requires exactly 1 arg: bash, zsh, fish or powershell")
			}
			out, err := shellInit(args[0], name)
			if err != nil {
				return err
			}
			fmt.Fprint(stdout, out)
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
	case "powershell", "pwsh":
		return fmt.Sprintf(powershellInit, name), nil
	}
	return "", fmt.Errorf("unsupported shell %q, use bash, zsh, fish or powershell", shell)
}

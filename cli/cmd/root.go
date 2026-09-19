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

// Package cmd implements the sourceseedy subcommands
package cmd

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/drewstinnett/sourceseedy/internal/finder"
)

var (
	base    string
	verbose bool
	// Set at build time with -ldflags -X
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const longDescription = `Quickly move around your various source directories, assuming a standard
directory structure of:

${base}/${remote-host}/${namespace}/${repo}

${base} - Defaults to $SOURCESEEDY_BASE, or ~/src if that is not set
${remote-host} - This will be something like github.com, gitlab.com, gitlab.yourco.com
${namespace} - Namespace containing the repo. This could be just the owner, or a nested group
${repo} - The repo itself`

// command is a single sourceseedy subcommand
type command struct {
	name  string
	usage string
	short string
	long  string
	flags func(fs *flag.FlagSet)
	run   func(args []string) error
}

var commands []*command

// Execute parses the command line and runs the requested subcommand.
// This is called by main.main().
func Execute() {
	if err := run(os.Args[1:]); err != nil {
		// Cancelling fzf isn't an error worth reporting, but callers like
		// scd need the non-zero exit to know not to cd
		if errors.Is(err, finder.ErrNoSelection) {
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// defaultBase is the base directory used when -base isn't given
func defaultBase() string {
	if b := os.Getenv("SOURCESEEDY_BASE"); b != "" {
		return b
	}
	return "~/src"
}

func run(args []string) error {
	base = defaultBase()
	root := flag.NewFlagSet("sourceseedy", flag.ExitOnError)
	globalFlags(root)
	showVersion := root.Bool("version", false, "Print the version and exit")
	root.Usage = func() { rootUsage(root) }
	_ = root.Parse(args)

	if *showVersion {
		fmt.Printf("sourceseedy version %s (commit %s, built %s)\n", version, commit, date)
		return nil
	}

	args = root.Args()
	if len(args) == 0 || args[0] == "help" {
		root.Usage()
		return nil
	}

	cmd := findCommand(args[0])
	if cmd == nil {
		root.Usage()
		return fmt.Errorf("unknown command %q", args[0])
	}

	fs := flag.NewFlagSet(cmd.name, flag.ExitOnError)
	globalFlags(fs)
	if cmd.flags != nil {
		cmd.flags(fs)
	}
	fs.Usage = func() { commandUsage(cmd, fs) }
	_ = fs.Parse(args[1:])

	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	var err error
	base, err = expandHome(base)
	if err != nil {
		return err
	}
	return cmd.run(fs.Args())
}

// globalFlags registers flags that are valid both before and after the subcommand
func globalFlags(fs *flag.FlagSet) {
	for _, name := range []string{"base", "b"} {
		fs.StringVar(&base, name, base, "Base directory containing sources, defaults to $SOURCESEEDY_BASE")
	}
	for _, name := range []string{"verbose", "v"} {
		fs.BoolVar(&verbose, name, verbose, "Enable verbose logging")
	}
}

func findCommand(name string) *command {
	for _, c := range commands {
		if c.name == name {
			return c
		}
	}
	return nil
}

func rootUsage(fs *flag.FlagSet) {
	out := fs.Output()
	fmt.Fprintf(out, "%s\n\nUsage:\n  sourceseedy [flags] <command> [args]\n\nCommands:\n", longDescription)
	for _, c := range commands {
		fmt.Fprintf(out, "  %-10s %s\n", c.name, c.short)
	}
	fmt.Fprintln(out, "\nFlags:")
	fs.PrintDefaults()
}

func commandUsage(c *command, fs *flag.FlagSet) {
	out := fs.Output()
	desc := c.long
	if desc == "" {
		desc = c.short
	}
	fmt.Fprintf(out, "%s\n\nUsage:\n  sourceseedy %s\n\nFlags:\n", desc, c.usage)
	fs.PrintDefaults()
}

// expandHome replaces a leading ~ in p with the current user's home directory
func expandHome(p string) (string, error) {
	if p != "~" && !strings.HasPrefix(p, "~/") {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, p[1:]), nil
}

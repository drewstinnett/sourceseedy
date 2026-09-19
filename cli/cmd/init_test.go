package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/drewstinnett/sourceseedy/internal/fakeexe"
)

func TestMain(m *testing.M) {
	if fakeexe.Is("sourceseedy") {
		fakeSourceseedyMain()
	}
	os.Exit(m.Run())
}

// fakeSourceseedyMain is what the fake sourceseedy does when TestMain finds
// itself running as sourceseedy. It records its args, then either picks
// $FAKE_SS_TARGET or exits 1 like a cancelled fzf
func fakeSourceseedyMain() {
	f, err := os.OpenFile(os.Getenv("FAKE_SS_ARGS_FILE"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		os.Exit(99)
	}
	for _, a := range os.Args[1:] {
		fmt.Fprintln(f, a)
	}
	_ = f.Close()
	if os.Getenv("FAKE_SS_CANCEL") != "" {
		os.Exit(1)
	}
	fmt.Println(os.Getenv("FAKE_SS_TARGET"))
	os.Exit(0)
}

func TestShellInitErrors(t *testing.T) {
	if _, err := shellInit("cmd", "scd"); err == nil {
		t.Error("expected error for unsupported shell")
	}
	for _, name := range []string{"", "1scd", "a b", "scd;rm", "$(x)", "a-b"} {
		if _, err := shellInit("zsh", name); err == nil {
			t.Errorf("expected error for function name %q", name)
		}
	}
}

func TestShellInitName(t *testing.T) {
	for shell, prefix := range map[string]string{
		"zsh":        "jump() {",
		"bash":       "jump() {",
		"fish":       "function jump\n",
		"powershell": "function jump {",
		"pwsh":       "function jump {",
		"PowerShell": "function jump {",
	} {
		out, err := shellInit(shell, "jump")
		if err != nil {
			t.Fatalf("%s: %v", shell, err)
		}
		if !strings.HasPrefix(out, prefix) {
			t.Errorf("%s: unexpected output: %s", shell, out)
		}
	}
}

// shellCase says how to drive one shell from a test
type shellCase struct {
	init string   // what to pass to shellInit
	exes []string // programs that could be this shell, in order of preference
	// flags go before the script, and keep the user's own config away from PATH
	flags []string
	// load returns the script that defines the function from the file at f
	load func(f string) string
	// syntax returns the args that check that the file at f parses
	syntax func(f string) []string
	// pwd is the script that prints the current directory
	pwd string
	// unix shells are skipped on Windows, where they mangle paths
	unixOnly bool
}

func posixLoad(f string) string { return ". '" + f + "'" }

func powershellLoad(f string) string {
	return "Invoke-Expression ([IO.File]::ReadAllText('" + f + "'))"
}

var shellCases = map[string]shellCase{
	"bash": {
		init: "bash", exes: []string{"bash"}, flags: []string{"--noprofile", "--norc", "-c"},
		load: posixLoad, syntax: func(f string) []string { return []string{"-n", f} },
		pwd: "pwd -P", unixOnly: true,
	},
	"zsh": {
		init: "zsh", exes: []string{"zsh"}, flags: []string{"-f", "-c"},
		load: posixLoad, syntax: func(f string) []string { return []string{"-n", f} },
		pwd: "pwd -P", unixOnly: true,
	},
	"fish": {
		init: "fish", exes: []string{"fish"}, flags: []string{"--no-config", "-c"},
		load:   func(f string) string { return "source '" + f + "'" },
		syntax: func(f string) []string { return []string{"-n", f} },
		pwd:    "pwd -P", unixOnly: true,
	},
	"powershell": {
		init: "powershell", exes: []string{"pwsh", "powershell"},
		flags: []string{"-NoProfile", "-NonInteractive", "-Command"},
		load:  powershellLoad,
		syntax: func(f string) []string {
			return []string{"-NoProfile", "-NonInteractive", "-Command", "[void][scriptblock]::Create([IO.File]::ReadAllText('" + f + "'))"}
		},
		pwd: "(Get-Location).Path",
	},
}

// TestShellInitInRealShells defines the function in each installed shell,
// with a fake sourceseedy on PATH, and checks whether it cd's
func TestShellInitInRealShells(t *testing.T) {
	for name, sc := range shellCases {
		t.Run(name, func(t *testing.T) {
			if sc.unixOnly && runtime.GOOS == "windows" {
				t.Skip("unix shell")
			}
			var bin string
			for _, exe := range sc.exes {
				if p, err := exec.LookPath(exe); err == nil {
					bin = p
					break
				}
			}
			if bin == "" {
				t.Skipf("none of %v installed", sc.exes)
			}
			code, err := shellInit(sc.init, "scd")
			if err != nil {
				t.Fatal(err)
			}

			tmp := t.TempDir()
			codeFile := filepath.Join(tmp, "init")
			if err := os.WriteFile(codeFile, []byte(code), 0o600); err != nil {
				t.Fatal(err)
			}
			if out, err := exec.Command(bin, sc.syntax(codeFile)...).CombinedOutput(); err != nil {
				t.Fatalf("syntax check failed: %v\n%s", err, out)
			}

			fakeexe.Install(t, "sourceseedy")
			argsFile := filepath.Join(tmp, "args")
			start := filepath.Join(tmp, "start")
			target := filepath.Join(tmp, "has a space")
			for _, d := range []string{start, target} {
				if err := os.Mkdir(d, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("FAKE_SS_ARGS_FILE", argsFile)
			t.Setenv("FAKE_SS_TARGET", target)

			run := func(cancel bool) string {
				t.Helper()
				if cancel {
					t.Setenv("FAKE_SS_CANCEL", "1")
				} else {
					t.Setenv("FAKE_SS_CANCEL", "")
				}
				ctx, cancelCtx := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancelCtx()
				script := sc.load(codeFile) + "; scd myfilter; " + sc.pwd
				cmd := exec.CommandContext(ctx, bin, append(append([]string{}, sc.flags...), script)...)
				cmd.Dir = start
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("running %s: %v\n%s", name, err, out)
				}
				lines := strings.Split(strings.TrimSpace(string(out)), "\n")
				return strings.TrimSpace(lines[len(lines)-1])
			}
			resolve := func(p string) string {
				t.Helper()
				r, err := filepath.EvalSymlinks(p)
				if err != nil {
					t.Fatalf("%q: %v", p, err)
				}
				return r
			}

			if got, want := resolve(run(false)), resolve(target); got != want {
				t.Errorf("after selecting: pwd = %q, want %q", got, want)
			}
			args, err := os.ReadFile(argsFile)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.ReplaceAll(string(args), "\r\n", "\n"); got != "fzf\nmyfilter\n" {
				t.Errorf("sourceseedy args = %q", args)
			}
			if got, want := resolve(run(true)), resolve(start); got != want {
				t.Errorf("after cancelling: pwd = %q, want to stay in %q", got, want)
			}
		})
	}
}

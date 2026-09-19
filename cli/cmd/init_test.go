package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestShellInitErrors(t *testing.T) {
	if _, err := shellInit("powershell", "scd"); err == nil {
		t.Error("expected error for unsupported shell")
	}
	for _, name := range []string{"", "1scd", "a b", "scd;rm", "$(x)", "a-b"} {
		if _, err := shellInit("zsh", name); err == nil {
			t.Errorf("expected error for function name %q", name)
		}
	}
}

func TestShellInitName(t *testing.T) {
	out, err := shellInit("zsh", "jump")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "jump() {") {
		t.Errorf("unexpected output: %s", out)
	}
	out, err = shellInit("fish", "jump")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "function jump\n") {
		t.Errorf("unexpected output: %s", out)
	}
}

// TestShellInitInRealShells defines the function in each installed shell,
// with a fake sourceseedy on PATH, and checks whether it cd's
func TestShellInitInRealShells(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs a posix environment")
	}
	// Flags to keep the user's own shell config from touching PATH
	shells := map[string][]string{
		"bash": {"--noprofile", "--norc"},
		"zsh":  {"-f"},
		"fish": {"--no-config"},
	}
	for shell, noConfig := range shells {
		t.Run(shell, func(t *testing.T) {
			bin, err := exec.LookPath(shell)
			if err != nil {
				t.Skipf("%s not installed", shell)
			}
			code, err := shellInit(shell, "scd")
			if err != nil {
				t.Fatal(err)
			}

			tmp := t.TempDir()
			codeFile := filepath.Join(tmp, "init")
			if err := os.WriteFile(codeFile, []byte(code), 0o600); err != nil {
				t.Fatal(err)
			}
			if out, err := exec.Command(bin, "-n", codeFile).CombinedOutput(); err != nil {
				t.Fatalf("syntax check failed: %v\n%s", err, out)
			}

			// Fake sourceseedy: records args, then either picks $SCD_TARGET or
			// exits 1 like a cancelled fzf
			fakeBin := filepath.Join(tmp, "bin")
			if err := os.Mkdir(fakeBin, 0o755); err != nil {
				t.Fatal(err)
			}
			argsFile := filepath.Join(tmp, "args")
			script := "#!/bin/sh\n" +
				"printf '%s\\n' \"$@\" > '" + argsFile + "'\n" +
				"[ -n \"$SCD_CANCEL\" ] && exit 1\n" +
				"printf '%s\\n' \"$SCD_TARGET\"\n"
			if err := os.WriteFile(filepath.Join(fakeBin, "sourceseedy"), []byte(script), 0o755); err != nil { //nolint:gosec // must be executable
				t.Fatal(err)
			}

			start := filepath.Join(tmp, "start")
			target := filepath.Join(tmp, "has a space")
			for _, d := range []string{start, target} {
				if err := os.Mkdir(d, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			run := func(cancel bool) string {
				t.Helper()
				source := "source " + codeFile
				if shell != "fish" {
					source = ". " + codeFile
				}
				ctx, cancelCtx := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancelCtx()
				cmd := exec.CommandContext(ctx, bin, append(noConfig, "-c", source+"; scd myfilter; pwd -P")...)
				cmd.Dir = start
				cmd.Env = append(os.Environ(),
					"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
					"SCD_TARGET="+target,
				)
				if cancel {
					cmd.Env = append(cmd.Env, "SCD_CANCEL=1")
				}
				out, err := cmd.Output()
				if err != nil {
					t.Fatalf("running %s: %v", shell, err)
				}
				return strings.TrimSpace(string(out))
			}
			resolve := func(p string) string {
				t.Helper()
				r, err := filepath.EvalSymlinks(p)
				if err != nil {
					t.Fatal(err)
				}
				return r
			}

			if got, want := run(false), resolve(target); got != want {
				t.Errorf("after selecting: pwd = %q, want %q", got, want)
			}
			args, err := os.ReadFile(argsFile)
			if err != nil {
				t.Fatal(err)
			}
			if string(args) != "fzf\nmyfilter\n" {
				t.Errorf("sourceseedy args = %q", args)
			}
			if got, want := run(true), resolve(start); got != want {
				t.Errorf("after cancelling: pwd = %q, want to stay in %q", got, want)
			}
		})
	}
}

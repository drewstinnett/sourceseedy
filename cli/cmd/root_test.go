package cmd

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/git"
)

func TestExpandHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	sep := string(filepath.Separator)
	for in, want := range map[string]string{
		"~":              home,
		"~/src":          filepath.Join(home, "src"),
		"~" + sep + "a":  filepath.Join(home, "a"),
		"~other/src":     "~other/src",
		"src/~":          "src/~",
		"":               "",
		"relative/path":  "relative/path",
		"/absolute/path": "/absolute/path",
	} {
		got, err := expandHome(in)
		if err != nil {
			t.Fatalf("expandHome(%q): %v", in, err)
		}
		if got != want {
			t.Errorf("expandHome(%q) = %q, want %q", in, got, want)
		}
	}
}

// A flag after an argument used to be taken as another argument, so
// `import ./repo -d` did the import for real
func TestFlagsMayFollowArguments(t *testing.T) {
	b := t.TempDir()
	repo := newLocalRepo(t, "git@git.example.com:a/b.git")
	dest := filepath.Join(b, "git.example.com", "a", "b")
	_, errOut := capture(t)

	if err := run([]string{"-b", b, "import", repo, "-d"}); err != nil {
		t.Fatal(err)
	}
	if !git.IsLocalGitRepo(repo) || git.IsLocalGitRepo(dest) {
		t.Fatal("a dry run moved the repo")
	}
	if got := errOut.String(); !strings.Contains(got, "Would move") {
		t.Errorf("expected a Would move line, got:\n%s", got)
	}
}

func TestFlagsMayFollowRemoteArguments(t *testing.T) {
	fakeRemotes(t, "a/b.git")
	b := t.TempDir()
	capture(t)
	if err := run([]string{"-b", b, "clone", "https://git.example.com/a/b.git", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if entries, err := os.ReadDir(b); err != nil || len(entries) != 0 {
		t.Errorf("a dry run cloned something: %v (err %v)", entries, err)
	}
}

func TestParseFlags(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		want []string
		dry  bool
		base string
	}{
		{name: "no args"},
		{name: "arguments only", args: []string{"a", "b"}, want: []string{"a", "b"}},
		{name: "flag first", args: []string{"-d", "a"}, want: []string{"a"}, dry: true},
		{name: "flag last", args: []string{"a", "-d"}, want: []string{"a"}, dry: true},
		{name: "flag between", args: []string{"a", "-d", "b"}, want: []string{"a", "b"}, dry: true},
		{name: "flag with a value after an argument", args: []string{"a", "-b", "x", "c"}, want: []string{"a", "c"}, base: "x"},
		{name: "long form with equals", args: []string{"a", "--b=x"}, want: []string{"a"}, base: "x"},
		{name: "double dash ends flags", args: []string{"a", "--", "-d", "b"}, want: []string{"a", "-d", "b"}},
		{name: "double dash first", args: []string{"--", "-d"}, want: []string{"-d"}},
		{name: "flag before double dash", args: []string{"-d", "--", "-x"}, want: []string{"-x"}, dry: true},
		{name: "a lone dash is an argument", args: []string{"-", "a"}, want: []string{"-", "a"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			var dry bool
			var base string
			fs.BoolVar(&dry, "d", false, "")
			fs.StringVar(&base, "b", "", "")
			fs.StringVar(&base, "base", "", "")
			got, err := parseFlags(fs, tt.args)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("arguments = %q, want %q", got, tt.want)
			}
			if dry != tt.dry || base != tt.base {
				t.Errorf("flags: dry=%v base=%q, want dry=%v base=%q", dry, base, tt.dry, tt.base)
			}
		})
	}
}

func TestParseFlagsUnknownFlag(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if _, err := parseFlags(fs, []string{"a", "--nope"}); err == nil {
		t.Error("expected an error for an unknown flag after an argument")
	}
}

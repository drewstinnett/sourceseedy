package cmd

import (
	"path/filepath"
	"testing"
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

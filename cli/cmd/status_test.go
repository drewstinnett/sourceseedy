package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/gittest"
)

// statusFixture makes a base with four projects under git.example.com/a: clean,
// dirty (an edited file), ahead (an unpushed commit) and broken (a .git that
// isn't a repo). It returns the base and a function giving a project's path
func statusFixture(t *testing.T) (string, func(name string) string) {
	t.Helper()
	gittest.Isolate(t)
	remote := gittest.Remote(t)
	base := t.TempDir()
	at := func(name string) string { return filepath.Join(base, "git.example.com", "a", name) }

	gittest.Clone(t, remote, at("clean"))
	dirty := gittest.Clone(t, remote, at("dirty"))
	if err := os.WriteFile(filepath.Join(dirty, "README.md"), []byte("edited\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ahead := gittest.Clone(t, remote, at("ahead"))
	gittest.Commit(t, ahead, "local.txt", "x")
	if err := os.MkdirAll(filepath.Join(at("broken"), ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return base, at
}

func TestStatusTable(t *testing.T) {
	base, _ := statusFixture(t)
	out, errOut := capture(t)
	if err := run([]string{"-b", base, "status"}); err != nil {
		t.Fatal(err)
	}

	// One line per project that needs attention, in order, and not the clean one
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), out)
	}
	for i, want := range []struct{ id, rest string }{
		{"git.example.com/a/ahead", "main"},
		{"git.example.com/a/broken", "error:"},
		{"git.example.com/a/dirty", "1 changed"},
	} {
		if !strings.HasPrefix(lines[i], want.id+" ") || !strings.Contains(lines[i], want.rest) {
			t.Errorf("line %d = %q, want %s ... %s", i, lines[i], want.id, want.rest)
		}
	}
	if !strings.Contains(lines[0], "ahead 1") {
		t.Errorf("expected the ahead project to say ahead 1: %q", lines[0])
	}
	if strings.Contains(out.String(), "a/clean") {
		t.Errorf("clean project should be left out:\n%s", out)
	}
	if got := errOut.String(); !strings.Contains(got, "Checked") || !strings.Contains(got, "4 projects, 3 need attention") {
		t.Errorf("expected a summary on stderr, got:\n%s", got)
	}
	noNoise(t, errOut.String())
}

func TestStatusAll(t *testing.T) {
	base, _ := statusFixture(t)
	out, _ := capture(t)
	if err := run([]string{"-b", base, "status", "--all"}); err != nil {
		t.Fatal(err)
	}
	var clean string
	for _, line := range strings.Split(out.String(), "\n") {
		if strings.HasPrefix(line, "git.example.com/a/clean ") {
			clean = line
		}
	}
	if !strings.Contains(clean, "main") || !strings.HasSuffix(clean, "clean") {
		t.Errorf("expected a clean line for the clean project, got %q in:\n%s", clean, out)
	}
}

func TestStatusJSON(t *testing.T) {
	base, at := statusFixture(t)
	out, _ := capture(t)
	if err := run([]string{"-b", base, "status", "--json"}); err != nil {
		t.Fatal(err)
	}
	var got []statusReport
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 reports, got %d:\n%s", len(got), out)
	}

	ahead, broken, dirty := got[0], got[1], got[2]
	if ahead.ID != "git.example.com/a/ahead" || ahead.Path != at("ahead") ||
		ahead.Branch != "main" || ahead.Upstream != "origin/main" || ahead.Ahead != 1 ||
		ahead.Clean || !slices.Equal(ahead.Problems, []string{"ahead 1"}) {
		t.Errorf("unexpected ahead report: %+v", ahead)
	}
	if dirty.Changed != 1 || !slices.Equal(dirty.Problems, []string{"1 changed"}) {
		t.Errorf("unexpected dirty report: %+v", dirty)
	}
	if broken.Error == "" || broken.Clean {
		t.Errorf("expected an error and not clean for the broken repo: %+v", broken)
	}
	// Scripts should get [] rather than null
	if strings.Contains(out.String(), `"problems": null`) || !strings.Contains(out.String(), `"problems": []`) {
		t.Errorf("problems should be an empty list, not null:\n%s", out)
	}

	out.Reset()
	if err := run([]string{"-b", base, "status", "--json", "--all"}); err != nil {
		t.Fatal(err)
	}
	got = nil
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 || got[2].ID != "git.example.com/a/clean" || !got[2].Clean || len(got[2].Problems) != 0 {
		t.Errorf("expected the clean project too, got %+v", got)
	}
}

func TestStatusAllClean(t *testing.T) {
	gittest.Isolate(t)
	base := t.TempDir()
	gittest.Clone(t, gittest.Remote(t), filepath.Join(base, "git.example.com", "a", "clean"))
	out, errOut := capture(t)

	if err := run([]string{"-b", base, "status"}); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Errorf("nothing to report, but stdout got %q", out)
	}
	if got := errOut.String(); !strings.Contains(got, "1 project, all clean") {
		t.Errorf("expected an all clean summary, got:\n%s", got)
	}

	out.Reset()
	if err := run([]string{"-b", base, "status", "--json"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.String()); got != "[]" {
		t.Errorf("expected an empty JSON list, got %q", got)
	}
}

func TestStatusArgs(t *testing.T) {
	capture(t)
	if err := run([]string{"-b", t.TempDir(), "status", "extra"}); err == nil {
		t.Error("expected an error for an unexpected arg")
	}
}

func TestStatusWorkersWithinLimits(t *testing.T) {
	if n := statusWorkers(); n < 4 || n > 16 {
		t.Errorf("statusWorkers() = %d, want 4 to 16", n)
	}
}

func TestListFlags(t *testing.T) {
	base := t.TempDir()
	for _, d := range []string{"gitlab.com/grp/sub/repo", "github.com/b/two", "github.com/a/one"} {
		if err := os.MkdirAll(filepath.Join(base, filepath.FromSlash(d), ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	out, _ := capture(t)

	if err := run([]string{"-b", base, "list"}); err != nil {
		t.Fatal(err)
	}
	if want := "github.com/a/one\ngithub.com/b/two\ngitlab.com/grp/sub/repo\n"; out.String() != want {
		t.Errorf("list got %q, want %q", out, want)
	}

	out.Reset()
	if err := run([]string{"-b", base, "list", "--full-path"}); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(base, "github.com", "a", "one") + "\n" +
		filepath.Join(base, "github.com", "b", "two") + "\n" +
		filepath.Join(base, "gitlab.com", "grp", "sub", "repo") + "\n"
	if out.String() != want {
		t.Errorf("list --full-path got %q, want %q", out, want)
	}

	out.Reset()
	if err := run([]string{"-b", base, "list", "--json"}); err != nil {
		t.Fatal(err)
	}
	var got []listEntry
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	wantEntries := []listEntry{
		{ID: "github.com/a/one", Host: "github.com", Namespace: "a", Name: "one", Path: filepath.Join(base, "github.com", "a", "one")},
		{ID: "github.com/b/two", Host: "github.com", Namespace: "b", Name: "two", Path: filepath.Join(base, "github.com", "b", "two")},
		{ID: "gitlab.com/grp/sub/repo", Host: "gitlab.com", Namespace: "grp", Name: "sub/repo", Path: filepath.Join(base, "gitlab.com", "grp", "sub", "repo")},
	}
	if !slices.Equal(got, wantEntries) {
		t.Errorf("list --json got %+v, want %+v", got, wantEntries)
	}
}

func TestListJSONEmptyAndRelativeBase(t *testing.T) {
	base := t.TempDir()
	out, _ := capture(t)
	if err := run([]string{"-b", base, "list", "--json"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.String()); got != "[]" {
		t.Errorf("expected an empty JSON list, got %q", got)
	}

	// A relative base still gives absolute paths, so they work after a cd
	if err := os.MkdirAll(filepath.Join(base, "src", "github.com", "a", "one", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(base)
	out.Reset()
	if err := run([]string{"-b", "src", "list", "--full-path"}); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(out.String())
	if !filepath.IsAbs(got) || !strings.HasSuffix(got, filepath.Join("src", "github.com", "a", "one")) {
		t.Errorf("expected an absolute path, got %q", got)
	}
}

func TestListRejectsArgs(t *testing.T) {
	capture(t)
	if err := run([]string{"-b", t.TempDir(), "list", "extra"}); err == nil {
		t.Error("expected an error for an unexpected arg")
	}
}

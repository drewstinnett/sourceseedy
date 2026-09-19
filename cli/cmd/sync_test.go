package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/git"
	"github.com/drewstinnett/sourceseedy/internal/gittest"
	"github.com/drewstinnett/sourceseedy/internal/project"
)

// advanceRemote pushes n new commits to remote's main, from a clone of its own
// that is thrown away, and returns the new head
func advanceRemote(t *testing.T, remote string, n int) string {
	t.Helper()
	work := gittest.Clone(t, remote, filepath.Join(t.TempDir(), "work"))
	for i := range n {
		gittest.Commit(t, work, "remote"+string(rune('a'+i))+".txt", "x")
	}
	gittest.Git(t, work, "push", "-q", "origin", "main")
	return head(t, work)
}

func head(t *testing.T, repo string) string {
	t.Helper()
	return strings.TrimSpace(gittest.Git(t, repo, "rev-parse", "HEAD"))
}

// syncFixture makes a base with six projects under git.example.com/a, and then
// pushes 2 commits to the remote three of them share. It returns the base, a
// function giving a project's path, and the remote's new head. The projects are:
//
//	behind    clean, so it can be fast-forwarded
//	broken    a .git that isn't a repo
//	current   nothing new upstream
//	dirty     behind, with an edited file
//	diverged  has a commit of its own as well as being behind
//	noup      up to date, but its branch tracks nothing
func syncFixture(t *testing.T) (string, func(name string) string, string) {
	t.Helper()
	gittest.Isolate(t)
	shared, other := gittest.Remote(t), gittest.Remote(t)
	base := t.TempDir()
	at := func(name string) string { return filepath.Join(base, "git.example.com", "a", name) }

	gittest.Clone(t, shared, at("behind"))
	diverged := gittest.Clone(t, shared, at("diverged"))
	gittest.Commit(t, diverged, "local.txt", "x")
	dirty := gittest.Clone(t, shared, at("dirty"))
	if err := os.WriteFile(filepath.Join(dirty, "README.md"), []byte("edited\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gittest.Clone(t, other, at("current"))
	noup := gittest.Clone(t, other, at("noup"))
	gittest.Git(t, noup, "branch", "--unset-upstream")
	if err := os.MkdirAll(filepath.Join(at("broken"), ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	return base, at, advanceRemote(t, shared, 2)
}

func TestSyncFastForwards(t *testing.T) {
	base, at, remoteHead := syncFixture(t)
	divergedHead, dirtyHead := head(t, at("diverged")), head(t, at("dirty"))
	out, errOut := capture(t)

	err := run([]string{"-b", base, "sync"})
	if err == nil || !strings.Contains(err.Error(), "1 project failed") {
		t.Fatalf("expected an error for the broken project, got %v", err)
	}

	// The clean one moved, and only it
	if got := head(t, at("behind")); got != remoteHead {
		t.Errorf("behind is at %s, want the remote's %s", got, remoteHead)
	}
	if got := head(t, at("diverged")); got != divergedHead {
		t.Errorf("diverged moved from %s to %s", divergedHead, got)
	}
	if got := head(t, at("dirty")); got != dirtyHead {
		t.Errorf("dirty moved from %s to %s", dirtyHead, got)
	}
	if got, _ := os.ReadFile(filepath.Join(at("dirty"), "README.md")); string(got) != "edited\n" {
		t.Errorf("dirty's edit was lost, README.md is now %q", got)
	}

	if out.Len() != 0 {
		t.Errorf("no --json, but stdout got %q", out)
	}
	got := errOut.String()
	for _, want := range []string{
		"Updated  git.example.com/a/behind  main  2 commits",
		"Skipped  git.example.com/a/diverged: diverged (ahead 1, behind 2)",
		"Skipped  git.example.com/a/dirty: uncommitted changes",
		"Failed   git.example.com/a/broken: git fetch:",
		"Synced   6 projects, 1 updated, 2 skipped, 1 failed",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("stderr is missing %q:\n%s", want, got)
		}
	}
	// Nothing to say about these two, unless asked
	for _, quiet := range []string{"a/current", "a/noup"} {
		if strings.Contains(got, quiet) {
			t.Errorf("%s had nothing to do, but was reported:\n%s", quiet, got)
		}
	}
	noNoise(t, got)
}

func TestSyncFetchOnly(t *testing.T) {
	base, at, remoteHead := syncFixture(t)
	before := head(t, at("behind"))
	out, errOut := capture(t)

	// The broken project fails the run, but that is not what this is about
	_ = run([]string{"-b", base, "sync", "--fetch-only"})

	if got := head(t, at("behind")); got != before {
		t.Errorf("--fetch-only moved a branch, %s to %s", before, got)
	}
	// It did fetch, so git knows where the remote is now
	if got := strings.TrimSpace(gittest.Git(t, at("behind"), "rev-parse", "origin/main")); got != remoteHead {
		t.Errorf("origin/main is %s, want %s, so nothing was fetched", got, remoteHead)
	}
	if out.Len() != 0 {
		t.Errorf("stdout got %q", out)
	}
	got := errOut.String()
	for _, want := range []string{
		"Fetched  git.example.com/a/behind  main  behind 2",
		"Fetched  git.example.com/a/dirty  main  behind 2",
		"Fetched  git.example.com/a/diverged  main  behind 2",
		"Fetched  6 projects, 1 failed, 3 behind",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("stderr is missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Updated") || strings.Contains(got, "Skipped") {
		t.Errorf("nothing is moved or skipped with --fetch-only:\n%s", got)
	}
}

func TestSyncDryRun(t *testing.T) {
	base, at, _ := syncFixture(t)
	before := head(t, at("behind"))
	_, errOut := capture(t)

	_ = run([]string{"-b", base, "sync", "-d"})

	if got := head(t, at("behind")); got != before {
		t.Errorf("a dry run moved a branch, %s to %s", before, got)
	}
	got := errOut.String()
	for _, want := range []string{
		"Would update git.example.com/a/behind  main  2 commits",
		"Skipped  git.example.com/a/dirty: uncommitted changes",
		"Checked  6 projects, 1 would update, 2 skipped, 1 failed",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("stderr is missing %q:\n%s", want, got)
		}
	}
}

func TestSyncJSON(t *testing.T) {
	base, at, _ := syncFixture(t)
	out, _ := capture(t)

	_ = run([]string{"-b", base, "sync", "--json", "--all"})

	var got []syncReport
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	type want struct {
		id, result string
		commits    int
	}
	wants := []want{
		{"git.example.com/a/behind", "updated", 2},
		{"git.example.com/a/broken", "failed", 0},
		{"git.example.com/a/current", "unchanged", 0},
		{"git.example.com/a/dirty", "skipped", 2},
		{"git.example.com/a/diverged", "skipped", 2},
		{"git.example.com/a/noup", "fetched", 0},
	}
	if len(got) != len(wants) {
		t.Fatalf("expected %d reports, got %d:\n%s", len(wants), len(got), out)
	}
	for i, w := range wants {
		r := got[i]
		if r.ID != w.id || r.Result != w.result || r.Commits != w.commits {
			t.Errorf("report %d = %s %s %d, want %s %s %d", i, r.ID, r.Result, r.Commits, w.id, w.result, w.commits)
		}
	}
	if got[0].Path != at("behind") || got[0].Branch != "main" {
		t.Errorf("unexpected path or branch in %+v", got[0])
	}
	if got[1].Error == "" || got[4].Reason == "" || got[5].Reason != "no upstream" {
		t.Errorf("expected an error for broken, and reasons for diverged and noup: %+v", got)
	}

	// Without --all, only what needs saying
	out.Reset()
	if err := run([]string{"-b", base, "sync", "--json"}); err == nil {
		t.Error("expected the broken project to fail the run")
	}
	got = nil
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].ID != "git.example.com/a/broken" {
		t.Errorf("expected broken, dirty and diverged, got %+v", got)
	}
}

func TestSyncAllShowsCurrent(t *testing.T) {
	base, _, _ := syncFixture(t)
	_, errOut := capture(t)

	_ = run([]string{"-b", base, "sync", "-a"})

	got := errOut.String()
	for _, want := range []string{
		"Current  git.example.com/a/current  main",
		"Fetched  git.example.com/a/noup  main  no upstream",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("stderr is missing %q:\n%s", want, got)
		}
	}
}

func TestSyncAllUpToDate(t *testing.T) {
	gittest.Isolate(t)
	base := t.TempDir()
	gittest.Clone(t, gittest.Remote(t), filepath.Join(base, "git.example.com", "a", "current"))
	out, errOut := capture(t)

	if err := run([]string{"-b", base, "sync"}); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Errorf("nothing to report, but stdout got %q", out)
	}
	if got := errOut.String(); got != "Synced   1 project, all up to date\n" {
		t.Errorf("expected an all up to date summary, got %q", got)
	}

	out.Reset()
	if err := run([]string{"-b", base, "sync", "--json"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.String()); got != "[]" {
		t.Errorf("expected an empty JSON list, got %q", got)
	}
}

// A branch deleted on the remote is forgotten by the fetch, which leaves it
// with an upstream that is gone. That is worth saying, and isn't a failure
func TestSyncPrunesDeletedBranch(t *testing.T) {
	gittest.Isolate(t)
	remote := gittest.Remote(t)
	base := t.TempDir()
	repo := gittest.Clone(t, remote, filepath.Join(base, "git.example.com", "a", "gone"))
	gittest.Git(t, repo, "checkout", "-q", "-b", "feature")
	gittest.Git(t, repo, "push", "-q", "-u", "origin", "feature")
	gittest.Git(t, remote, "branch", "-q", "-D", "feature")
	out, errOut := capture(t)

	if err := run([]string{"-b", base, "sync", "--json", "--all"}); err != nil {
		t.Fatal(err)
	}
	var got []syncReport
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Result != "fetched" || got[0].Reason != "upstream gone" || got[0].Branch != "feature" {
		t.Errorf("expected feature to be fetched with its upstream gone, got %+v", got)
	}
	if refs := gittest.Git(t, repo, "branch", "-r"); strings.Contains(refs, "origin/feature") {
		t.Errorf("origin/feature should have been pruned, remote branches:\n%s", refs)
	}
	noNoise(t, errOut.String())
}

// A host that times out over and over without ever answering is given up on,
// but one slow fetch on a host that is fine, like a big repo, must not do that
func TestSyncGivesUpOnHostThatIsNotAnswering(t *testing.T) {
	var fetched []string
	old := fetch
	t.Cleanup(func() { fetch = old })
	fetch = func(_ context.Context, dir string) error {
		fetched = append(fetched, dir)
		if strings.HasPrefix(dir, "ok-") {
			return errors.New("git fetch: not a git repository")
		}
		return fmt.Errorf("git fetch: %w after 2m0s", git.ErrTimeout)
	}
	mk := func(host, name string) project.Project {
		return project.Project{Host: host, Namespace: "a", Name: name, Directory: name}
	}
	try := func(hosts *hostHealth, host, name string) syncReport {
		return syncProject(context.Background(), mk(host, name), syncMode{}, hosts)
	}

	var hosts hostHealth
	for _, name := range []string{"one", "two", "three"} {
		if r := try(&hosts, "dead.example.com", name); r.Result != "failed" || !strings.Contains(r.Error, "timed out after") {
			t.Errorf("%s: unexpected report %+v", name, r)
		}
	}
	four := try(&hosts, "dead.example.com", "four")
	if four.Result != "failed" || four.Error != "dead.example.com isn't answering, so it wasn't tried" {
		t.Errorf("four: expected it to be given up on, got %+v", four)
	}
	if !slices.Equal(fetched, []string{"one", "two", "three"}) {
		t.Errorf("fetched %v, want just one, two and three", fetched)
	}
	// Another host is not affected
	if r := try(&hosts, "other.example.com", "five"); !strings.Contains(r.Error, "timed out after") {
		t.Errorf("other host should still be tried, got %+v", r)
	}

	// A host that has answered before is never given up on, however many time out
	var busy hostHealth
	busy.fetched("big.example.com")
	fetched = nil
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		try(&busy, "big.example.com", name)
	}
	if len(fetched) != 5 {
		t.Errorf("a host that answered was given up on, only %v were tried", fetched)
	}
}

func TestPrintSyncReportsGroupsRepeatedFailures(t *testing.T) {
	_, errOut := capture(t)
	const dns = "git fetch: ssh: Could not resolve hostname a.example.com"
	failed := func(id, err string) syncReport { return syncReport{ID: id, Result: "failed", Error: err} }
	printSyncReports([]syncReport{
		failed("a.example.com/x/one", dns),
		failed("a.example.com/x/two", dns),
		{ID: "b.example.com/x/ok", Result: "updated", Branch: "main", Commits: 1},
		failed("a.example.com/x/three", dns),
		failed("b.example.com/x/odd", "git fetch: Repository not found"),
		failed("b.example.com/x/odder", "git fetch: Repository not found"),
	})

	want := "Failed   a.example.com: " + dns + " (3 projects)\n" +
		"Updated  b.example.com/x/ok  main  1 commit\n" +
		"Failed   b.example.com/x/odd: git fetch: Repository not found\n" +
		"Failed   b.example.com/x/odder: git fetch: Repository not found\n"
	if got := errOut.String(); got != want {
		t.Errorf("got:\n%swant:\n%s", got, want)
	}
}

func TestSyncArgs(t *testing.T) {
	capture(t)
	if err := run([]string{"-b", t.TempDir(), "sync", "extra"}); err == nil {
		t.Error("expected an error for an unexpected arg")
	}
}

func TestPlural(t *testing.T) {
	for n, want := range map[int]string{0: "0 commits", 1: "1 commit", 2: "2 commits"} {
		if got := plural(n, "commit"); got != want {
			t.Errorf("plural(%d) = %q, want %q", n, got, want)
		}
	}
}

package git_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/git"
	"github.com/drewstinnett/sourceseedy/internal/gittest"
)

func TestSyncAction(t *testing.T) {
	tracking := git.Status{Branch: "main", Upstream: "origin/main"}
	with := func(edit func(*git.Status)) git.Status {
		s := tracking
		edit(&s)
		return s
	}
	for _, tt := range []struct {
		name   string
		status git.Status
		want   git.SyncAction
		reason string
	}{
		{"up to date", tracking, git.SyncUpToDate, ""},
		{"ahead only", with(func(s *git.Status) { s.Ahead = 2 }), git.SyncUpToDate, ""},
		{"behind and dirty but nothing to pick up", with(func(s *git.Status) { s.Changed = 1 }), git.SyncUpToDate, ""},
		{"behind", with(func(s *git.Status) { s.Behind = 3 }), git.SyncFastForward, ""},
		{"behind with untracked files", with(func(s *git.Status) { s.Behind, s.Untracked = 3, 2 }), git.SyncFastForward, ""},
		{"behind and changed", with(func(s *git.Status) { s.Behind, s.Changed = 1, 1 }), git.SyncSkip, "uncommitted changes"},
		{"behind and conflicted", with(func(s *git.Status) { s.Behind, s.Conflicted = 1, 1 }), git.SyncSkip, "uncommitted changes"},
		{"diverged", with(func(s *git.Status) { s.Ahead, s.Behind = 1, 2 }), git.SyncSkip, "diverged (ahead 1, behind 2)"},
		{"diverged and dirty", with(func(s *git.Status) { s.Ahead, s.Behind, s.Changed = 1, 2, 1 }), git.SyncSkip, "diverged (ahead 1, behind 2)"},
		{"no commits", git.Status{Initial: true}, git.SyncNoBranch, "no commits"},
		{"detached", git.Status{Detached: true, Behind: 1}, git.SyncNoBranch, "detached"},
		{"no upstream", git.Status{Branch: "main"}, git.SyncNoBranch, "no upstream"},
		{"upstream gone", with(func(s *git.Status) { s.UpstreamGone = true }), git.SyncNoBranch, "upstream gone"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := tt.status.SyncAction()
			if got != tt.want || reason != tt.reason {
				t.Errorf("SyncAction() = %v, %q, want %v, %q", got, reason, tt.want, tt.reason)
			}
		})
	}
}

func TestFetchAndFastForward(t *testing.T) {
	gittest.Isolate(t)
	remote := gittest.Remote(t)
	repo := gittest.Clone(t, remote, filepath.Join(t.TempDir(), "repo"))
	other := gittest.Clone(t, remote, filepath.Join(t.TempDir(), "other"))
	gittest.Commit(t, other, "new.txt", "x")
	gittest.Git(t, other, "push", "-q", "origin", "main")

	ctx := context.Background()
	if err := git.Fetch(ctx, repo); err != nil {
		t.Fatal(err)
	}
	// Fetching only learns about the commit, it doesn't take it in
	if _, err := os.Stat(filepath.Join(repo, "new.txt")); err == nil {
		t.Fatal("Fetch changed the working tree")
	}
	if err := git.FastForward(ctx, repo); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repo, "new.txt")); err != nil {
		t.Errorf("FastForward didn't bring in new.txt: %v", err)
	}
}

func TestFastForwardRefusesDiverged(t *testing.T) {
	gittest.Isolate(t)
	remote := gittest.Remote(t)
	repo := gittest.Clone(t, remote, filepath.Join(t.TempDir(), "repo"))
	other := gittest.Clone(t, remote, filepath.Join(t.TempDir(), "other"))
	gittest.Commit(t, other, "theirs.txt", "x")
	gittest.Git(t, other, "push", "-q", "origin", "main")
	gittest.Commit(t, repo, "ours.txt", "x")

	ctx := context.Background()
	if err := git.Fetch(ctx, repo); err != nil {
		t.Fatal(err)
	}
	before := gittest.Git(t, repo, "rev-parse", "HEAD")
	if err := git.FastForward(ctx, repo); err == nil {
		t.Error("expected an error rather than a merge")
	}
	if after := gittest.Git(t, repo, "rev-parse", "HEAD"); after != before {
		t.Errorf("a refused fast-forward still moved HEAD, %s to %s", before, after)
	}
}

func TestFetchError(t *testing.T) {
	gittest.Isolate(t)
	repo := gittest.Clone(t, gittest.Remote(t), filepath.Join(t.TempDir(), "repo"))
	gittest.Git(t, repo, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing.git"))

	err := git.Fetch(context.Background(), repo)
	if err == nil || !strings.HasPrefix(err.Error(), "git fetch: ") || strings.Contains(err.Error(), "\n") {
		t.Fatalf("expected a one line error starting with git fetch:, got %v", err)
	}
	if !strings.Contains(err.Error(), "does not appear to be a git repository") {
		t.Errorf("expected what git said was wrong, got %v", err)
	}
	for _, noise := range []string{"Please make sure", "Could not read from remote", "fatal:"} {
		if strings.Contains(err.Error(), noise) {
			t.Errorf("error should leave out %q: %v", noise, err)
		}
	}
}

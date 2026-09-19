package git_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/drewstinnett/sourceseedy/internal/git"
	"github.com/drewstinnett/sourceseedy/internal/gittest"
)

func TestParseStatus(t *testing.T) {
	for _, tt := range []struct {
		name string
		out  string
		want git.Status
		att  []string
	}{
		{
			name: "clean and pushed",
			out:  "# branch.oid abc\n# branch.head main\n# branch.upstream origin/main\n# branch.ab +0 -0\n",
			want: git.Status{Branch: "main", Upstream: "origin/main"},
		},
		{
			name: "ahead and behind",
			out:  "# branch.oid abc\n# branch.head main\n# branch.upstream origin/main\n# branch.ab +2 -3\n",
			want: git.Status{Branch: "main", Upstream: "origin/main", Ahead: 2, Behind: 3},
			att:  []string{"ahead 2", "behind 3"},
		},
		{
			name: "changes of every kind",
			out: "# branch.oid abc\n# branch.head main\n# branch.upstream origin/main\n# branch.ab +0 -0\n" +
				"1 .M N... 100644 100644 100644 aaa bbb file.go\n" +
				"1 A. N... 000000 100644 100644 aaa bbb new.go\n" +
				"2 R. N... 100644 100644 100644 aaa bbb R100 renamed.go\told.go\n" +
				"u UU N... 100644 100644 100644 100644 aaa bbb ccc conflict.go\n" +
				"? untracked.txt\n? other.txt\n! ignored.log\n",
			want: git.Status{Branch: "main", Upstream: "origin/main", Changed: 3, Conflicted: 1, Untracked: 2},
			att:  []string{"1 conflicted", "3 changed", "2 untracked"},
		},
		{
			name: "no upstream",
			out:  "# branch.oid abc\n# branch.head feature\n",
			want: git.Status{Branch: "feature"},
			att:  []string{"no upstream"},
		},
		{
			name: "upstream gone",
			out:  "# branch.oid abc\n# branch.head topic\n# branch.upstream origin/topic\n",
			want: git.Status{Branch: "topic", Upstream: "origin/topic", UpstreamGone: true},
			att:  []string{"upstream gone"},
		},
		{
			name: "detached",
			out:  "# branch.oid abc\n# branch.head (detached)\n",
			want: git.Status{Detached: true},
			att:  []string{"detached"},
		},
		{
			name: "no commits yet",
			out:  "# branch.oid (initial)\n# branch.head main\n",
			want: git.Status{Branch: "main", Initial: true},
			att:  []string{"no commits"},
		},
		{
			name: "windows line endings",
			out:  "# branch.oid abc\r\n# branch.head main\r\n# branch.upstream origin/main\r\n# branch.ab +1 -0\r\n? f\r\n",
			want: git.Status{Branch: "main", Upstream: "origin/main", Ahead: 1, Untracked: 1},
			att:  []string{"1 untracked", "ahead 1"},
		},
		{
			// A file can't fake a header, since entries start with their own marker
			name: "odd file names",
			out:  "# branch.oid abc\n# branch.head main\n# branch.upstream origin/main\n# branch.ab +0 -0\n? # branch.head evil\n",
			want: git.Status{Branch: "main", Upstream: "origin/main", Untracked: 1},
			att:  []string{"1 untracked"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := git.ParseStatus(tt.out)
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
			if att := got.Attention(); !slices.Equal(att, tt.att) {
				t.Errorf("Attention() = %v, want %v", att, tt.att)
			}
		})
	}
}

func TestRepoStatus(t *testing.T) {
	gittest.Isolate(t)
	remote := gittest.Remote(t)
	// clone starts each scenario from a clean, pushed checkout
	clone := func(t *testing.T) string {
		t.Helper()
		return gittest.Clone(t, remote, filepath.Join(t.TempDir(), "work"))
	}

	for _, tt := range []struct {
		name  string
		setup func(t *testing.T, repo string)
		want  git.Status
		att   []string
	}{
		{name: "clean", want: git.Status{Branch: "main", Upstream: "origin/main"}},
		{
			name: "changed",
			setup: func(t *testing.T, repo string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("edited\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
			want: git.Status{Branch: "main", Upstream: "origin/main", Changed: 1},
			att:  []string{"1 changed"},
		},
		{
			name: "untracked",
			setup: func(t *testing.T, repo string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(repo, "new.txt"), []byte("x"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
			want: git.Status{Branch: "main", Upstream: "origin/main", Untracked: 1},
			att:  []string{"1 untracked"},
		},
		{
			name: "ahead",
			setup: func(t *testing.T, repo string) {
				t.Helper()
				gittest.Commit(t, repo, "local.txt", "x")
			},
			want: git.Status{Branch: "main", Upstream: "origin/main", Ahead: 1},
			att:  []string{"ahead 1"},
		},
		{
			name: "behind after a fetch",
			setup: func(t *testing.T, repo string) {
				t.Helper()
				other := clone(t)
				gittest.Commit(t, other, "elsewhere.txt", "x")
				gittest.Git(t, other, "push", "-q")
				gittest.Git(t, repo, "fetch", "-q")
			},
			want: git.Status{Branch: "main", Upstream: "origin/main", Behind: 1},
			att:  []string{"behind 1"},
		},
		{
			name: "no upstream",
			setup: func(t *testing.T, repo string) {
				t.Helper()
				gittest.Git(t, repo, "checkout", "-q", "-b", "feature")
			},
			want: git.Status{Branch: "feature"},
			att:  []string{"no upstream"},
		},
		{
			name: "detached",
			setup: func(t *testing.T, repo string) {
				t.Helper()
				gittest.Git(t, repo, "checkout", "-q", "--detach")
			},
			want: git.Status{Detached: true},
			att:  []string{"detached"},
		},
		{
			name: "upstream gone",
			setup: func(t *testing.T, repo string) {
				t.Helper()
				gittest.Git(t, repo, "checkout", "-q", "-b", "topic")
				gittest.Git(t, repo, "push", "-q", "-u", "origin", "topic")
				gittest.Git(t, repo, "push", "-q", "origin", "--delete", "topic")
			},
			want: git.Status{Branch: "topic", Upstream: "origin/topic", UpstreamGone: true},
			att:  []string{"upstream gone"},
		},
		{
			name: "commits, edits and untracked together",
			setup: func(t *testing.T, repo string) {
				t.Helper()
				gittest.Commit(t, repo, "local.txt", "x")
				if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("edited\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(repo, "new.txt"), []byte("x"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
			want: git.Status{Branch: "main", Upstream: "origin/main", Ahead: 1, Changed: 1, Untracked: 1},
			att:  []string{"1 changed", "1 untracked", "ahead 1"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := clone(t)
			if tt.setup != nil {
				tt.setup(t, repo)
			}
			got, err := git.RepoStatus(repo)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
			if att := got.Attention(); !slices.Equal(att, tt.att) {
				t.Errorf("Attention() = %v, want %v", att, tt.att)
			}
		})
	}
}

func TestRepoStatusNoCommits(t *testing.T) {
	gittest.Isolate(t)
	repo := t.TempDir()
	gittest.Git(t, repo, "init", "-q", "-b", "main")
	got, err := git.RepoStatus(repo)
	if err != nil {
		t.Fatal(err)
	}
	if want := (git.Status{Branch: "main", Initial: true}); got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestRepoStatusNotARepo(t *testing.T) {
	gittest.Isolate(t)
	// An empty .git isn't a repo, and isn't part of a bigger one
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := git.RepoStatus(dir); err == nil {
		t.Error("expected an error for something that isn't a git repo")
	}
}

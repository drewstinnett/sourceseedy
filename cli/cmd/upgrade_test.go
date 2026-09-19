package cmd

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/drewstinnett/sourceseedy/internal/update"
)

// asVersion runs the test as a build of version v
func asVersion(t *testing.T, v string) {
	t.Helper()
	old := version
	version = v
	t.Cleanup(func() { version = old })
}

// releaseServer serves a releases page whose latest release is tag
func releaseServer(t *testing.T, tag string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/o/r/releases/tag/"+tag, http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	old := releasesURL
	releasesURL = srv.URL + "/o/r/releases"
	t.Cleanup(func() { releasesURL = old })
}

// fakeExecutable stands in for the binary that would be replaced
func fakeExecutable(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "sourceseedy")
	if err := os.WriteFile(p, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	old := executable
	executable = func() (string, error) { return p, nil }
	t.Cleanup(func() { executable = old })
	return p
}

func TestUpgradeRefusesDevelopmentBuild(t *testing.T) {
	capture(t)
	for _, v := range []string{"dev", "v0.2.6-dirty"} {
		asVersion(t, v)
		err := run([]string{"upgrade"})
		if err == nil || !strings.Contains(err.Error(), "development build") {
			t.Errorf("%s: error = %v", v, err)
		}
	}
}

func TestUpgradeTakesNoArguments(t *testing.T) {
	capture(t)
	asVersion(t, "v0.2.6")
	if err := run([]string{"upgrade", "now"}); err == nil {
		t.Error("expected an error")
	}
}

func TestUpgradeDryRun(t *testing.T) {
	_, errOut := capture(t)
	asVersion(t, "v0.2.6")
	releaseServer(t, "v0.3.0")
	exe := fakeExecutable(t)

	for _, flag := range []string{"-d", "--dry-run"} {
		errOut.Reset()
		if err := run([]string{"upgrade", flag}); err != nil {
			t.Fatal(err)
		}
		if got := errOut.String(); !strings.Contains(got, "Would upgrade") || !strings.Contains(got, "v0.2.6 → v0.3.0") {
			t.Errorf("%s: expected a Would upgrade line, got:\n%s", flag, got)
		}
		if b, _ := os.ReadFile(exe); string(b) != "old" {
			t.Errorf("%s: a dry run changed the binary to %q", flag, b)
		}
	}
}

func TestUpgradeAlreadyLatest(t *testing.T) {
	_, errOut := capture(t)
	asVersion(t, "v0.3.0")
	releaseServer(t, "v0.3.0")
	exe := fakeExecutable(t)

	if err := run([]string{"upgrade"}); err != nil {
		t.Fatal(err)
	}
	if got := errOut.String(); !strings.Contains(got, "v0.3.0 is the latest release") {
		t.Errorf("expected to be told it is the latest, got:\n%s", got)
	}
	if b, _ := os.ReadFile(exe); string(b) != "old" {
		t.Errorf("binary changed to %q", b)
	}
}

// updateFixture is a stand-in for looking up releases, that counts how often it
// was asked
type updateFixture struct {
	fetches atomic.Int32
}

// fakeUpdates replaces where the update check looks and keeps its state, and
// says there is somewhere to tell the user. The latest release is v0.3.0
func fakeUpdates(t *testing.T) *updateFixture {
	t.Helper()
	f := &updateFixture{}
	t.Setenv("CI", "") // set on GitHub Actions
	t.Setenv("SOURCESEEDY_NO_UPDATE_CHECK", "")
	statePath := filepath.Join(t.TempDir(), "update.json")
	oldOpts, oldNotify := updateOptions, canNotify
	updateOptions = func() (update.Options, error) {
		return update.Options{
			Path:    statePath,
			Current: version,
			Fetch: func(context.Context) (string, error) {
				f.fetches.Add(1)
				return "v0.3.0", nil
			},
		}, nil
	}
	canNotify = func() bool { return true }
	t.Cleanup(func() { updateOptions, canNotify = oldOpts, oldNotify })
	return f
}

func TestUpdateNotice(t *testing.T) {
	_, errOut := capture(t)
	asVersion(t, "v0.2.6")
	f := fakeUpdates(t)
	b := t.TempDir()

	if err := run([]string{"-b", b, "list"}); err != nil {
		t.Fatal(err)
	}
	got := errOut.String()
	for _, want := range []string{"Update", "v0.3.0 is available", "you have v0.2.6", "sourceseedy upgrade"} {
		if !strings.Contains(got, want) {
			t.Errorf("notice is missing %q, got:\n%s", want, got)
		}
	}

	// Not on every run
	errOut.Reset()
	if err := run([]string{"-b", b, "list"}); err != nil {
		t.Fatal(err)
	}
	if got := errOut.String(); got != "" {
		t.Errorf("second run said:\n%s", got)
	}
	if n := f.fetches.Load(); n != 1 {
		t.Errorf("looked %d times, want 1", n)
	}
}

func TestUpdateNoticeUpToDate(t *testing.T) {
	_, errOut := capture(t)
	asVersion(t, "v0.3.0")
	fakeUpdates(t)
	if err := run([]string{"-b", t.TempDir(), "list"}); err != nil {
		t.Fatal(err)
	}
	if got := errOut.String(); got != "" {
		t.Errorf("said:\n%s", got)
	}
}

// The notice is for people, so it isn't added to the output of scripts
func TestUpdateNoticeNeedsATerminal(t *testing.T) {
	out, errOut := capture(t)
	asVersion(t, "v0.2.6")
	f := fakeUpdates(t)
	canNotify = func() bool { return false }
	if err := run([]string{"-b", t.TempDir(), "list"}); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 || errOut.Len() != 0 {
		t.Errorf("printed stdout %q, stderr %q", out, errOut)
	}
	if n := f.fetches.Load(); n != 1 {
		t.Errorf("looked %d times, want 1: the state should still be kept fresh", n)
	}
}

func TestUpdateCheckSkipped(t *testing.T) {
	for _, tt := range []struct {
		name    string
		version string
		env     map[string]string
		args    []string
	}{
		{name: "development build", version: "dev", args: []string{"list"}},
		{name: "built from a commit", version: "v0.2.6-3-gabcdef", args: []string{"list"}},
		{name: "opted out", version: "v0.2.6", env: map[string]string{"SOURCESEEDY_NO_UPDATE_CHECK": "1"}, args: []string{"list"}},
		{name: "in CI", version: "v0.2.6", env: map[string]string{"CI": "true"}, args: []string{"list"}},
		// init runs whenever a shell starts
		{name: "init", version: "v0.2.6", args: []string{"init", "zsh"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, errOut := capture(t)
			asVersion(t, tt.version)
			f := fakeUpdates(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if err := run(append([]string{"-b", t.TempDir()}, tt.args...)); err != nil {
				t.Fatal(err)
			}
			if n := f.fetches.Load(); n != 0 {
				t.Errorf("looked %d times, want 0", n)
			}
			if got := errOut.String(); got != "" {
				t.Errorf("said:\n%s", got)
			}
		})
	}
}

func TestUpdateCheckNoStateDirectory(t *testing.T) {
	_, errOut := capture(t)
	asVersion(t, "v0.2.6")
	fakeUpdates(t)
	updateOptions = func() (update.Options, error) { return update.Options{}, errors.New("no home") }
	if err := run([]string{"-b", t.TempDir(), "list"}); err != nil {
		t.Fatal(err)
	}
	if got := errOut.String(); got != "" {
		t.Errorf("said:\n%s", got)
	}
}

// A check that has nothing to say can't make a command wait long
func TestUpdateCheckDoesNotWaitLong(t *testing.T) {
	capture(t)
	asVersion(t, "v0.2.6")
	fakeUpdates(t)
	start := time.Now()
	if err := run([]string{"-b", t.TempDir(), "list"}); err != nil {
		t.Fatal(err)
	}
	if took := time.Since(start); took > 2*time.Second {
		t.Errorf("took %v", took)
	}
}

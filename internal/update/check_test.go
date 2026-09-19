package update

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

var epoch = time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)

// checkFixture is a state file and a clock, with a fake fetch that counts how
// many times it was asked
type checkFixture struct {
	t       *testing.T
	path    string
	now     time.Time
	latest  string
	err     error
	fetches atomic.Int32
}

func newCheckFixture(t *testing.T, latest string) *checkFixture {
	t.Helper()
	return &checkFixture{
		t:      t,
		path:   filepath.Join(t.TempDir(), "sub", "update.json"),
		now:    epoch,
		latest: latest,
	}
}

func (f *checkFixture) start(current string) *Check {
	f.t.Helper()
	return Start(context.Background(), Options{
		Path:    f.path,
		Current: current,
		Now:     func() time.Time { return f.now },
		Fetch: func(context.Context) (string, error) {
			f.fetches.Add(1)
			return f.latest, f.err
		},
	})
}

func (f *checkFixture) state() state {
	f.t.Helper()
	b, err := os.ReadFile(f.path)
	if err != nil {
		f.t.Fatal(err)
	}
	var st state
	if err := json.Unmarshal(b, &st); err != nil {
		f.t.Fatal(err)
	}
	return st
}

func (f *checkFixture) advance(d time.Duration) { f.now = f.now.Add(d) }

func TestCheckFirstRun(t *testing.T) {
	f := newCheckFixture(t, "v0.3.0")
	latest, ok := f.start("v0.2.6").Finish(true)
	if !ok || latest != "v0.3.0" {
		t.Fatalf("Finish = %q, %v, want v0.3.0, true", latest, ok)
	}
	if got := f.fetches.Load(); got != 1 {
		t.Errorf("fetched %d times, want 1", got)
	}
	st := f.state()
	if st.Latest != "v0.3.0" || !st.CheckedAt.Equal(epoch) || st.NotifiedFor != "v0.3.0" || !st.NotifiedAt.Equal(epoch) {
		t.Errorf("state = %+v", st)
	}
}

func TestCheckUsesFreshState(t *testing.T) {
	f := newCheckFixture(t, "v0.3.0")
	f.start("v0.2.6").Finish(false)

	f.advance(Interval - time.Minute)
	f.latest = "v0.4.0"
	latest, ok := f.start("v0.2.6").Finish(true)
	if got := f.fetches.Load(); got != 1 {
		t.Errorf("fetched %d times, want 1: a fresh state shouldn't ask again", got)
	}
	if !ok || latest != "v0.3.0" {
		t.Errorf("Finish = %q, %v, want the cached v0.3.0, true", latest, ok)
	}
}

func TestCheckAsksAgainWhenStale(t *testing.T) {
	f := newCheckFixture(t, "v0.3.0")
	f.start("v0.2.6").Finish(false)

	f.advance(Interval)
	f.latest = "v0.4.0"
	latest, ok := f.start("v0.2.6").Finish(true)
	if got := f.fetches.Load(); got != 2 {
		t.Errorf("fetched %d times, want 2", got)
	}
	if !ok || latest != "v0.4.0" {
		t.Errorf("Finish = %q, %v, want v0.4.0, true", latest, ok)
	}
	if st := f.state(); st.Latest != "v0.4.0" || !st.CheckedAt.Equal(f.now) {
		t.Errorf("state = %+v", st)
	}
}

// With no network every run would wait out the timeout, unless a failed check
// counts as one
func TestCheckFailureBacksOff(t *testing.T) {
	f := newCheckFixture(t, "v0.3.0")
	f.start("v0.2.6").Finish(false)

	f.advance(Interval)
	f.err = errors.New("no network")
	f.latest = ""
	latest, ok := f.start("v0.2.6").Finish(true)
	if !ok || latest != "v0.3.0" {
		t.Errorf("Finish = %q, %v, want the earlier v0.3.0, true", latest, ok)
	}
	if st := f.state(); st.Latest != "v0.3.0" || !st.CheckedAt.Equal(f.now) {
		t.Errorf("state = %+v, want v0.3.0 kept and the check recorded", st)
	}

	f.advance(time.Minute)
	f.start("v0.2.6").Finish(false)
	if got := f.fetches.Load(); got != 2 {
		t.Errorf("fetched %d times, want 2: a failure should wait for the interval too", got)
	}
}

func TestCheckFailureWithNothingKnown(t *testing.T) {
	f := newCheckFixture(t, "")
	f.err = errors.New("no network")
	if latest, ok := f.start("v0.2.6").Finish(true); ok {
		t.Errorf("Finish = %q, true, want nothing to report", latest)
	}
}

func TestCheckNoticeCadence(t *testing.T) {
	f := newCheckFixture(t, "v0.3.0")
	if _, ok := f.start("v0.2.6").Finish(true); !ok {
		t.Fatal("first run should bring it up")
	}

	f.advance(time.Minute)
	if latest, ok := f.start("v0.2.6").Finish(true); ok {
		t.Errorf("brought up %q again straight away", latest)
	}

	f.advance(Interval)
	if _, ok := f.start("v0.2.6").Finish(true); !ok {
		t.Error("should be brought up again after the interval")
	}
}

// Having been told about one release doesn't hold back news of the next
func TestCheckNewReleaseIsBroughtUpAtOnce(t *testing.T) {
	f := newCheckFixture(t, "v0.3.0")
	f.start("v0.2.6").Finish(true)

	f.advance(time.Minute)
	f.st(t, func(s *state) { s.CheckedAt = time.Time{} }) // makes it check again
	f.latest = "v0.4.0"
	if latest, ok := f.start("v0.2.6").Finish(true); !ok || latest != "v0.4.0" {
		t.Errorf("Finish = %q, %v, want v0.4.0, true", latest, ok)
	}
}

// st rewrites the state file
func (f *checkFixture) st(t *testing.T, edit func(*state)) {
	t.Helper()
	st := f.state()
	edit(&st)
	if err := writeState(f.path, st); err != nil {
		t.Fatal(err)
	}
}

// A run whose output goes to a pipe has nowhere to say anything, and shouldn't
// use up the notice
func TestCheckNotShownIsNotUsedUp(t *testing.T) {
	f := newCheckFixture(t, "v0.3.0")
	if latest, ok := f.start("v0.2.6").Finish(false); ok {
		t.Fatalf("Finish(false) = %q, true", latest)
	}
	if st := f.state(); !st.NotifiedAt.IsZero() {
		t.Errorf("NotifiedAt = %v, want it left alone", st.NotifiedAt)
	}
	f.advance(time.Minute)
	if _, ok := f.start("v0.2.6").Finish(true); !ok {
		t.Error("the next run to a terminal should bring it up")
	}
}

func TestCheckUpToDate(t *testing.T) {
	f := newCheckFixture(t, "v0.3.0")
	if latest, ok := f.start("v0.3.0").Finish(true); ok {
		t.Errorf("Finish = %q, true, want nothing to report", latest)
	}
	if latest, ok := f.start("v0.4.0").Finish(true); ok {
		t.Errorf("Finish = %q, true for a version ahead of the latest", latest)
	}
}

func TestCheckIgnoresDevelopmentBuilds(t *testing.T) {
	for _, v := range []string{"dev", "v0.2.6-dirty", "v0.2.6-3-gabcdef", ""} {
		f := newCheckFixture(t, "v0.3.0")
		if latest, ok := f.start(v).Finish(true); ok {
			t.Errorf("%q: Finish = %q, true", v, latest)
		}
		if got := f.fetches.Load(); got != 0 {
			t.Errorf("%q: fetched %d times, want 0", v, got)
		}
		if _, err := os.Stat(f.path); err == nil {
			t.Errorf("%q: wrote a state file", v)
		}
	}
}

func TestCheckCorruptState(t *testing.T) {
	f := newCheckFixture(t, "v0.3.0")
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := f.start("v0.2.6").Finish(true); !ok {
		t.Error("a corrupt state should be treated as no state")
	}
	if got := f.state().Latest; got != "v0.3.0" {
		t.Errorf("Latest = %q, want the state to be replaced", got)
	}
}

// The check can't hold up the command for longer than its timeout
func TestCheckTimesOut(t *testing.T) {
	fetchTimeout = 50 * time.Millisecond
	t.Cleanup(func() { fetchTimeout = 2 * time.Second })
	f := newCheckFixture(t, "v0.3.0")
	c := Start(context.Background(), Options{
		Path:    f.path,
		Current: "v0.2.6",
		Now:     func() time.Time { return f.now },
		Fetch: func(ctx context.Context) (string, error) {
			<-ctx.Done()
			return "", ctx.Err()
		},
	})
	start := time.Now()
	if latest, ok := c.Finish(true); ok {
		t.Errorf("Finish = %q, true", latest)
	}
	if took := time.Since(start); took > 2*fetchTimeout {
		t.Errorf("Finish took %v", took)
	}
	if st := f.state(); !st.CheckedAt.Equal(f.now) {
		t.Errorf("state = %+v, want the check recorded", st)
	}
}

func TestCachePath(t *testing.T) {
	p, err := CachePath()
	if err != nil {
		t.Skip("no cache directory here:", err)
	}
	if filepath.Base(p) != "update.json" || filepath.Base(filepath.Dir(p)) != "sourceseedy" {
		t.Errorf("CachePath = %q", p)
	}
}

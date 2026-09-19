package update

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	// Interval is how often to look for a new release, and how often to bring
	// one up
	Interval = 6 * time.Hour
)

// fetchTimeout is how long a check can hold up the command it ran beside
var fetchTimeout = 2 * time.Second

// state is what is remembered between runs
type state struct {
	CheckedAt time.Time `json:"checked_at"`
	Latest    string    `json:"latest"`
	// NotifiedFor and NotifiedAt are the release the user was last told about,
	// and when
	NotifiedFor string    `json:"notified_for"`
	NotifiedAt  time.Time `json:"notified_at"`
}

// Options configures Start
type Options struct {
	// Path is the file the state is kept in
	Path string
	// Current is the version that is running
	Current string
	// Fetch returns the latest release tag. The default asks GitHub
	Fetch func(ctx context.Context) (string, error)
	// Now is the clock, time.Now by default
	Now func() time.Time
}

// Check is one run's look at whether there is a newer release
type Check struct {
	opts    Options
	st      state
	started bool
	done    chan struct{}
	cancel  context.CancelFunc
}

// CachePath is where the state is kept by default, in the user's cache directory
func CachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sourceseedy", "update.json"), nil
}

// Start begins a check. It reads what was found last time, and when that is more
// than Interval old, asks for the latest release in the background so the
// command being run isn't held up by it. Call Finish when the command is done.
//
// Nothing is checked, and Finish never reports anything, when Current isn't a
// release
func Start(ctx context.Context, opts Options) *Check {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Fetch == nil {
		opts.Fetch = func(ctx context.Context) (string, error) {
			return Latest(ctx, http.DefaultClient, ReleasesURL)
		}
	}
	c := &Check{opts: opts}
	if !IsRelease(opts.Current) {
		return c
	}
	c.st = c.load()
	if opts.Now().Sub(c.st.CheckedAt) < Interval {
		return c
	}

	ctx, c.cancel = context.WithTimeout(ctx, fetchTimeout)
	c.started = true
	c.done = make(chan struct{})
	go func() {
		defer close(c.done)
		latest, err := opts.Fetch(ctx)
		if err != nil {
			// Still counts as a check. Otherwise, with no network, every run
			// would wait out the timeout
			slog.Debug("checking for a new release", "err", err)
			latest = c.st.Latest
		}
		c.st.CheckedAt = opts.Now()
		c.st.Latest = latest
		c.save()
	}()
	return c
}

// Finish waits for the check to be done, up to the time it is allowed. When
// show is set and there is a newer release the user hasn't heard about lately, it
// returns that release and remembers it was brought up. show is for callers to
// say whether there is anywhere to tell the user, so a run whose output goes to a
// pipe doesn't use up the notice
func (c *Check) Finish(show bool) (latest string, ok bool) {
	if c.started {
		<-c.done
		c.cancel()
	}
	if !show || !Newer(c.opts.Current, c.st.Latest) {
		return "", false
	}
	now := c.opts.Now()
	if c.st.NotifiedFor == c.st.Latest && now.Sub(c.st.NotifiedAt) < Interval {
		return "", false
	}
	c.st.NotifiedFor, c.st.NotifiedAt = c.st.Latest, now
	c.save()
	return c.st.Latest, true
}

// load reads the state, which is empty when there isn't any or it can't be read
func (c *Check) load() state {
	var st state
	b, err := os.ReadFile(c.opts.Path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.Debug("reading update state", "err", err)
		}
		return state{}
	}
	if err := json.Unmarshal(b, &st); err != nil {
		slog.Debug("reading update state", "err", err)
		return state{}
	}
	return st
}

// save writes the state. It is written to a temporary file first, since the
// process can exit while the check is still running and a half written file
// would lose the state. Nothing depends on it, so errors are only logged
func (c *Check) save() {
	if err := writeState(c.opts.Path, c.st); err != nil {
		slog.Debug("saving update state", "err", err)
	}
}

func writeState(p string, st state) error {
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(p), ".update-*")
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(f.Name(), p)
	}
	if err != nil {
		_ = os.Remove(f.Name())
	}
	return err
}

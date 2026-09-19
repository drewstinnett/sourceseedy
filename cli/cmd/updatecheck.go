package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/drewstinnett/sourceseedy/internal/update"
)

// releasesURL is where releases are published. Tests point it somewhere else
var releasesURL = update.ReleasesURL

// updateOptions returns what checking for a new release needs. Tests swap it for
// something that doesn't go to the network
var updateOptions = func() (update.Options, error) {
	path, err := update.CachePath()
	return update.Options{Path: path, Current: version}, err
}

// canNotify reports whether there is somewhere to tell the user about a new
// release. That is only a terminal, so a script's output is never added to
var canNotify = func() bool { return isTerminal(stderr) }

// startUpdateCheck starts looking for a newer release beside the command c, and
// returns a function to call when c is done, that says so if there is one. The
// look is at most every 6 hours, and only holds the command up if it is over
// quicker than the few seconds a look is allowed. It does nothing for a
// development build, when SOURCESEEDY_NO_UPDATE_CHECK or CI is set, or for
// commands that opt out
func startUpdateCheck(c *command) (finish func()) {
	skip := func() {}
	if c.noUpdateCheck || !update.IsRelease(version) ||
		os.Getenv("SOURCESEEDY_NO_UPDATE_CHECK") != "" || os.Getenv("CI") != "" {
		return skip
	}
	opts, err := updateOptions()
	if err != nil {
		slog.Debug("no place to keep update state", "err", err)
		return skip
	}
	check := update.Start(context.Background(), opts)
	return func() {
		if latest, ok := check.Finish(canNotify()); ok {
			note("Update", fmt.Sprintf("%s is available, you have %s. Run: sourceseedy upgrade", latest, version))
		}
	}
}

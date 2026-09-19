package cmd

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"time"

	"github.com/drewstinnett/sourceseedy/internal/update"
)

// upgradeTimeout is how long a whole upgrade can take, on a slow connection
const upgradeTimeout = 2 * time.Minute

// executable finds the binary to replace. Tests swap it, so they don't replace
// the test binary
var executable = update.Executable

func init() {
	var dryRun bool
	commands = append(commands, &command{
		name:  "upgrade",
		usage: "upgrade [flags]",
		short: "Upgrade sourceseedy to the latest release",
		long: `Downloads the latest release for this machine from GitHub, checks it against
the checksums published with it, and replaces the running sourceseedy with it.

This needs to be able to write to the directory sourceseedy is installed in.
Development builds can't be upgraded.

To turn off the hint that a new release is out, set SOURCESEEDY_NO_UPDATE_CHECK=1`,
		noUpdateCheck: true,
		flags: func(fs *flag.FlagSet) {
			for _, name := range []string{"dry-run", "d"} {
				fs.BoolVar(&dryRun, name, false, "Just say what the latest release is, don't upgrade")
			}
		},
		run: func(args []string) error {
			if len(args) != 0 {
				return errors.New("upgrade doesn't take any arguments")
			}
			return upgrade(dryRun)
		},
	})
}

func upgrade(dryRun bool) error {
	if !update.IsRelease(version) {
		return errors.New("this is a development build (" + version + "), so there is no release to upgrade it from")
	}
	exe, err := executable()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), upgradeTimeout)
	defer cancel()
	latest, upgraded, err := update.Upgrade(ctx, update.UpgradeOptions{
		Current: version,
		Target:  exe,
		BaseURL: releasesURL,
		Client:  http.DefaultClient,
		DryRun:  dryRun,
	})
	if err != nil {
		return err
	}
	switch {
	case upgraded:
		done("Upgraded", version+" → "+latest+" "+tildePath(exe))
	case dryRun && update.Newer(version, latest):
		preview("Would upgrade", version+" → "+latest+" "+tildePath(exe))
	default:
		note("Current", "sourceseedy "+version+" is the latest release")
	}
	return nil
}

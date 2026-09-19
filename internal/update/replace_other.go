//go:build !windows

package update

import "os"

// replace puts the file at from in to place as target. This is fine while target
// is running, since the old file just stays around until it exits
func replace(target, from string) error {
	return os.Rename(from, target)
}

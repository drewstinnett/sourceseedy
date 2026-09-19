//go:build windows

package update

import "os"

// replace puts the file at from in to place as target. Windows won't let a
// running program be overwritten or deleted, but it can be renamed, so the old
// one is moved out of the way. It can't be removed until it has exited, so it is
// cleared up by the next upgrade
func replace(target, from string) error {
	old := target + ".old"
	_ = os.Remove(old)
	if err := os.Rename(target, old); err != nil {
		return err
	}
	if err := os.Rename(from, target); err != nil {
		_ = os.Rename(old, target)
		return err
	}
	_ = os.Remove(old)
	return nil
}

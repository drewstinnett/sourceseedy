// Package update finds newer releases of sourceseedy and upgrades the running
// binary to them
package update

import (
	"strconv"
	"strings"
)

// IsRelease reports whether v is a clean release version like v1.2.3, as
// opposed to a development build: dev, v1.2.3-dirty, v1.2.3-4-gabcdef
func IsRelease(v string) bool {
	_, ok := parseVersion(v)
	return ok
}

// Newer reports whether latest is a later release than current. Both have to be
// clean release versions, so a development build is never told it is out of date
func Newer(current, latest string) bool {
	c, ok := parseVersion(current)
	if !ok {
		return false
	}
	l, ok := parseVersion(latest)
	if !ok {
		return false
	}
	for i := range c {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

// parseVersion splits v1.2.3, with or without the v, in to its numbers
func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if len(parts) != len(out) {
		return out, false
	}
	for i, p := range parts {
		// Atoi takes a sign, which isn't part of a version
		if p == "" || strings.TrimLeft(p, "0123456789") != "" {
			return out, false
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

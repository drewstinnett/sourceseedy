package update

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// ReleasesURL is where releases are published
const ReleasesURL = "https://github.com/drewstinnett/sourceseedy/releases"

// Latest returns the tag of the newest release, like v0.3.0. It doesn't use the
// GitHub API, which limits how often an address can ask. Instead it reads where
// <base>/latest redirects to, which is <base>/tag/<tag>
func Latest(ctx context.Context, client *http.Client, base string) (string, error) {
	// Only the redirect is wanted, so don't follow it to the release page
	noFollow := *client
	noFollow.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, base+"/latest", nil)
	if err != nil {
		return "", err
	}
	resp, err := noFollow.Do(req)
	if err != nil {
		return "", err
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 300 || resp.StatusCode > 399 {
		return "", fmt.Errorf("looking up the latest release: %s", resp.Status)
	}
	loc, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return "", fmt.Errorf("looking up the latest release: %w", err)
	}
	dir, tag := path.Split(loc.Path)
	if tag == "" || !strings.HasSuffix(dir, "/tag/") {
		return "", fmt.Errorf("looking up the latest release: unexpected redirect to %q", loc)
	}
	return tag, nil
}

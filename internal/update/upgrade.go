package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// A release is a few megabytes. These are far more than that, and only there
	// so something wrong can't fill the disk
	maxDownload = 256 << 20
	maxSums     = 1 << 20
)

// UpgradeOptions configures Upgrade
type UpgradeOptions struct {
	// Current is the version that is running
	Current string
	// Target is the binary to replace
	Target string
	// BaseURL is where releases are published, ReleasesURL by default
	BaseURL string
	// Client makes the requests, http.DefaultClient by default
	Client *http.Client
	// GOOS and GOARCH pick the build to install, this machine's by default
	GOOS, GOARCH string
	// DryRun only looks up what the latest release is
	DryRun bool
}

// Executable returns the path of the running binary, with symlinks resolved so
// that the binary is replaced and not a link to it
func Executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// Upgrade replaces opts.Target with the latest release, and returns the version
// it is at now. upgraded is false when Target is already the latest, or for a
// dry run. The new binary is checked against the release's checksums before it
// is used, and Target is left alone if anything goes wrong
func Upgrade(ctx context.Context, opts UpgradeOptions) (latest string, upgraded bool, err error) {
	if !IsRelease(opts.Current) {
		return "", false, fmt.Errorf("%s is a development build, and there is no release to upgrade it from", opts.Current)
	}
	if opts.BaseURL == "" {
		opts.BaseURL = ReleasesURL
	}
	if opts.Client == nil {
		opts.Client = http.DefaultClient
	}
	if opts.GOOS == "" {
		opts.GOOS = runtime.GOOS
	}
	if opts.GOARCH == "" {
		opts.GOARCH = runtime.GOARCH
	}

	latest, err = Latest(ctx, opts.Client, opts.BaseURL)
	if err != nil {
		return "", false, err
	}
	if !Newer(opts.Current, latest) || opts.DryRun {
		return latest, false, nil
	}
	archiveName, sumsName, err := assetNames(latest, opts.GOOS, opts.GOARCH)
	if err != nil {
		return "", false, err
	}
	dl := opts.BaseURL + "/download/" + latest + "/"

	var sums strings.Builder
	if err := download(ctx, opts.Client, dl+sumsName, &sums, maxSums); err != nil {
		return "", false, err
	}
	want, err := checksumFor(sums.String(), archiveName)
	if err != nil {
		return "", false, err
	}

	// Everything is put next to Target, so that it is on the same filesystem and
	// can be renamed in to place, and so not being allowed to write there is the
	// first thing found out
	dir := filepath.Dir(opts.Target)
	tmp, err := os.CreateTemp(dir, ".sourceseedy-download-*")
	if err != nil {
		return "", false, writeError(dir, err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	got := sha256.New()
	err = download(ctx, opts.Client, dl+archiveName, io.MultiWriter(tmp, got), maxDownload)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", false, err
	}
	if sum := hex.EncodeToString(got.Sum(nil)); !strings.EqualFold(sum, want) {
		return "", false, fmt.Errorf("%s doesn't match its checksum (got %s, want %s)", archiveName, sum, want)
	}

	binary, err := os.CreateTemp(dir, ".sourceseedy-new-*")
	if err != nil {
		return "", false, writeError(dir, err)
	}
	defer func() { _ = os.Remove(binary.Name()) }()
	err = extractBinary(tmp.Name(), archiveName, binary, binaryName(opts.GOOS))
	if cerr := binary.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", false, err
	}
	if err := os.Chmod(binary.Name(), 0o755); err != nil {
		return "", false, err
	}
	if err := replace(opts.Target, binary.Name()); err != nil {
		return "", false, err
	}
	return latest, true, nil
}

// assetNames returns the names of the archive and the checksums file for a
// release, following the name templates in .goreleaser.yaml
func assetNames(tag, goos, goarch string) (archive, sums string, err error) {
	switch goos {
	case "darwin", "linux", "windows":
	default:
		return "", "", fmt.Errorf("no release is built for %s", goos)
	}
	switch goarch {
	case "amd64", "arm64":
	default:
		return "", "", fmt.Errorf("no release is built for %s/%s", goos, goarch)
	}
	version := strings.TrimPrefix(tag, "v")
	osName, ext := goos, ".tar.gz"
	if goos == "darwin" {
		osName = "macOS"
	}
	if goos == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("sourceseedy-%s_%s_%s%s", version, osName, goarch, ext),
		fmt.Sprintf("sourceseedy-%s_SHA256SUMS", version), nil
}

func binaryName(goos string) string {
	if goos == "windows" {
		return "sourceseedy.exe"
	}
	return "sourceseedy"
}

// checksumFor finds the checksum of name in the text of a checksums file, which
// has a line of "<hex>  <name>" for each file
func checksumFor(sums, name string) (string, error) {
	for line := range strings.Lines(sums) {
		fields := strings.Fields(line)
		// A * in front of the name marks a file as binary
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == name {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no checksum for %s in the release", name)
}

// download writes the body of url to w, up to limit bytes
func download(ctx context.Context, client *http.Client, url string, w io.Writer, limit int64) error {
	slog.Debug("downloading", "url", url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading %s: %s", path.Base(url), resp.Status)
	}
	n, err := io.Copy(w, io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return fmt.Errorf("downloading %s: %w", path.Base(url), err)
	}
	if n > limit {
		return fmt.Errorf("downloading %s: larger than the %d bytes expected", path.Base(url), limit)
	}
	return nil
}

// extractBinary copies the file called name out of the archive at src, and in
// to dst. archiveName says what kind of archive it is
func extractBinary(src, archiveName string, dst io.Writer, name string) error {
	var err error
	if strings.HasSuffix(archiveName, ".zip") {
		err = extractZip(src, dst, name)
	} else {
		err = extractTarGz(src, dst, name)
	}
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s isn't in the release archive", name)
	}
	if err != nil {
		return fmt.Errorf("reading %s: %w", archiveName, err)
	}
	return nil
}

func extractTarGz(src string, dst io.Writer, name string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return fs.ErrNotExist
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag == tar.TypeReg && path.Base(hdr.Name) == name {
			return copyLimited(dst, tr)
		}
	}
}

func extractZip(src string, dst io.Writer, name string) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() { _ = zr.Close() }()
	for _, zf := range zr.File {
		if zf.FileInfo().Mode().IsRegular() && path.Base(zf.Name) == name {
			r, err := zf.Open()
			if err != nil {
				return err
			}
			defer func() { _ = r.Close() }()
			return copyLimited(dst, r)
		}
	}
	return fs.ErrNotExist
}

func copyLimited(dst io.Writer, src io.Reader) error {
	n, err := io.Copy(dst, io.LimitReader(src, maxDownload+1))
	if err != nil {
		return err
	}
	if n > maxDownload {
		return errors.New("the binary in the release archive is too large")
	}
	return nil
}

// writeError explains not being able to make a file in dir
func writeError(dir string, err error) error {
	if errors.Is(err, fs.ErrPermission) {
		return fmt.Errorf("can't write to %s: %w. Install sourceseedy somewhere you own, or upgrade it with sudo", dir, err)
	}
	return err
}

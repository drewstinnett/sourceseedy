package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
)

const oldBinary = "the old binary"

// release is a fake GitHub releases page, serving files as the assets of v0.3.0,
// which is its latest release
type release struct {
	srv       *httptest.Server
	mu        sync.Mutex
	downloads []string
}

func newRelease(t *testing.T, files map[string][]byte) *release {
	t.Helper()
	const tag = "v0.3.0"
	r := &release{}
	mux := http.NewServeMux()
	mux.HandleFunc("/o/r/releases/latest", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/o/r/releases/tag/"+tag, http.StatusFound)
	})
	mux.HandleFunc("/o/r/releases/download/"+tag+"/", func(w http.ResponseWriter, req *http.Request) {
		name := req.URL.Path[strings.LastIndex(req.URL.Path, "/")+1:]
		r.mu.Lock()
		r.downloads = append(r.downloads, name)
		r.mu.Unlock()
		b, ok := files[name]
		if !ok {
			http.NotFound(w, req)
			return
		}
		_, _ = w.Write(b)
	})
	r.srv = httptest.NewServer(mux)
	t.Cleanup(r.srv.Close)
	return r
}

func (r *release) url() string { return r.srv.URL + "/o/r/releases" }

func tarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, name := range slices.Sorted(mapKeys(files)) {
		body := files[name]
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func zipFile(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range slices.Sorted(mapKeys(files)) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(files[name])); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func mapKeys[V any](m map[string]V) func(func(string) bool) {
	return func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

// sums returns a checksums file for files, the way goreleaser writes one
func sums(files map[string][]byte) []byte {
	var b strings.Builder
	for _, name := range slices.Sorted(mapKeys(files)) {
		fmt.Fprintf(&b, "%x  %s\n", sha256.Sum256(files[name]), name)
	}
	return []byte(b.String())
}

// target makes a directory with an old binary in it, and returns its path
func target(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "sourceseedy")
	if err := os.WriteFile(p, []byte(oldBinary), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// onlyTarget checks nothing but the target is left in its directory, so a failed
// upgrade doesn't leave downloads lying around
func onlyTarget(t *testing.T, target string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(target))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if want := []string{filepath.Base(target)}; !slices.Equal(names, want) {
		t.Errorf("directory has %v, want %v", names, want)
	}
}

func TestUpgrade(t *testing.T) {
	for _, tt := range []struct {
		name         string
		goos, goarch string
		archive      string
		zipped       bool
		contents     map[string]string
		binary       string
	}{
		{
			name: "linux", goos: "linux", goarch: "amd64",
			archive:  "sourceseedy-0.3.0_linux_amd64.tar.gz",
			contents: map[string]string{"sourceseedy": "new linux binary", "LICENSE": "MIT", "README.md": "hi"},
			binary:   "new linux binary",
		},
		{
			name: "macOS is named for the marketing", goos: "darwin", goarch: "arm64",
			archive:  "sourceseedy-0.3.0_macOS_arm64.tar.gz",
			contents: map[string]string{"sourceseedy": "new mac binary"},
			binary:   "new mac binary",
		},
		{
			name: "windows is a zip", goos: "windows", goarch: "amd64",
			archive: "sourceseedy-0.3.0_windows_amd64.zip", zipped: true,
			contents: map[string]string{"sourceseedy.exe": "new windows binary", "LICENSE": "MIT"},
			binary:   "new windows binary",
		},
		{
			name: "binary inside a directory", goos: "linux", goarch: "arm64",
			archive:  "sourceseedy-0.3.0_linux_arm64.tar.gz",
			contents: map[string]string{"sourceseedy-0.3.0/sourceseedy": "wrapped binary", "sourceseedy-0.3.0/LICENSE": "MIT"},
			binary:   "wrapped binary",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var archive []byte
			if tt.zipped {
				archive = zipFile(t, tt.contents)
			} else {
				archive = tarGz(t, tt.contents)
			}
			files := map[string][]byte{tt.archive: archive}
			// Other builds in the release, that shouldn't be touched
			files["sourceseedy-0.3.0_linux_riscv.tar.gz"] = []byte("nope")
			files["sourceseedy-0.3.0_SHA256SUMS"] = sums(files)
			rel := newRelease(t, files)
			dst := target(t)

			latest, upgraded, err := Upgrade(context.Background(), UpgradeOptions{
				Current: "v0.2.6", Target: dst, BaseURL: rel.url(), Client: rel.srv.Client(),
				GOOS: tt.goos, GOARCH: tt.goarch,
			})
			if err != nil {
				t.Fatal(err)
			}
			if latest != "v0.3.0" || !upgraded {
				t.Errorf("Upgrade = %q, %v, want v0.3.0, true", latest, upgraded)
			}
			if got := readFile(t, dst); got != tt.binary {
				t.Errorf("binary = %q, want %q", got, tt.binary)
			}
			if runtime.GOOS != "windows" {
				if fi, err := os.Stat(dst); err != nil || fi.Mode().Perm() != 0o755 {
					t.Errorf("mode = %v (err %v), want 0755", fi.Mode().Perm(), err)
				}
			}
			onlyTarget(t, dst)
		})
	}
}

func TestUpgradeAlreadyCurrent(t *testing.T) {
	for _, current := range []string{"v0.3.0", "v0.4.0"} {
		rel := newRelease(t, nil)
		dst := target(t)
		latest, upgraded, err := Upgrade(context.Background(), UpgradeOptions{
			Current: current, Target: dst, BaseURL: rel.url(), Client: rel.srv.Client(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if latest != "v0.3.0" || upgraded {
			t.Errorf("%s: Upgrade = %q, %v, want v0.3.0, false", current, latest, upgraded)
		}
		if len(rel.downloads) != 0 {
			t.Errorf("%s: downloaded %v", current, rel.downloads)
		}
		if got := readFile(t, dst); got != oldBinary {
			t.Errorf("%s: binary changed to %q", current, got)
		}
	}
}

func TestUpgradeDryRun(t *testing.T) {
	rel := newRelease(t, nil)
	dst := target(t)
	latest, upgraded, err := Upgrade(context.Background(), UpgradeOptions{
		Current: "v0.2.6", Target: dst, BaseURL: rel.url(), Client: rel.srv.Client(), DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if latest != "v0.3.0" || upgraded {
		t.Errorf("Upgrade = %q, %v, want v0.3.0, false", latest, upgraded)
	}
	if len(rel.downloads) != 0 {
		t.Errorf("downloaded %v", rel.downloads)
	}
}

func TestUpgradeDevelopmentBuild(t *testing.T) {
	for _, current := range []string{"dev", "v0.2.6-dirty", "v0.2.6-3-gabcdef"} {
		rel := newRelease(t, nil)
		dst := target(t)
		_, _, err := Upgrade(context.Background(), UpgradeOptions{
			Current: current, Target: dst, BaseURL: rel.url(), Client: rel.srv.Client(),
		})
		if err == nil || !strings.Contains(err.Error(), "development build") {
			t.Errorf("%s: error = %v", current, err)
		}
		if got := readFile(t, dst); got != oldBinary {
			t.Errorf("%s: binary changed to %q", current, got)
		}
	}
}

// Every way an upgrade can go wrong leaves the old binary as it was
func TestUpgradeFailures(t *testing.T) {
	const archiveName = "sourceseedy-0.3.0_linux_amd64.tar.gz"
	good := tarGz(t, map[string]string{"sourceseedy": "new binary"})
	for _, tt := range []struct {
		name    string
		files   func() map[string][]byte
		goarch  string
		wantErr string
	}{
		{
			name: "checksum doesn't match",
			files: func() map[string][]byte {
				return map[string][]byte{
					archiveName:                    good,
					"sourceseedy-0.3.0_SHA256SUMS": sums(map[string][]byte{archiveName: []byte("something else")}),
				}
			},
			wantErr: "doesn't match its checksum",
		},
		{
			name: "no line for this build",
			files: func() map[string][]byte {
				return map[string][]byte{
					archiveName:                    good,
					"sourceseedy-0.3.0_SHA256SUMS": sums(map[string][]byte{"sourceseedy-0.3.0_linux_arm64.tar.gz": good}),
				}
			},
			wantErr: "no checksum for " + archiveName,
		},
		{
			name:    "no checksums file",
			files:   func() map[string][]byte { return map[string][]byte{archiveName: good} },
			wantErr: "SHA256SUMS: 404",
		},
		{
			name: "no archive",
			files: func() map[string][]byte {
				return map[string][]byte{"sourceseedy-0.3.0_SHA256SUMS": sums(map[string][]byte{archiveName: good})}
			},
			wantErr: archiveName + ": 404",
		},
		{
			name: "binary isn't in the archive",
			files: func() map[string][]byte {
				bad := tarGz(t, map[string]string{"LICENSE": "MIT"})
				return map[string][]byte{archiveName: bad, "sourceseedy-0.3.0_SHA256SUMS": sums(map[string][]byte{archiveName: bad})}
			},
			wantErr: "sourceseedy isn't in the release archive",
		},
		{
			name: "not an archive",
			files: func() map[string][]byte {
				bad := []byte("not gzip")
				return map[string][]byte{archiveName: bad, "sourceseedy-0.3.0_SHA256SUMS": sums(map[string][]byte{archiveName: bad})}
			},
			wantErr: "reading " + archiveName,
		},
		{
			name:    "no build for this architecture",
			files:   func() map[string][]byte { return nil },
			goarch:  "riscv64",
			wantErr: "no release is built for linux/riscv64",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rel := newRelease(t, tt.files())
			dst := target(t)
			goarch := tt.goarch
			if goarch == "" {
				goarch = "amd64"
			}
			_, upgraded, err := Upgrade(context.Background(), UpgradeOptions{
				Current: "v0.2.6", Target: dst, BaseURL: rel.url(), Client: rel.srv.Client(),
				GOOS: "linux", GOARCH: goarch,
			})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want one containing %q", err, tt.wantErr)
			}
			if upgraded {
				t.Error("reported an upgrade")
			}
			if got := readFile(t, dst); got != oldBinary {
				t.Errorf("binary changed to %q", got)
			}
			onlyTarget(t, dst)
		})
	}
}

func TestUpgradeNoLatestRelease(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	dst := target(t)
	_, _, err := Upgrade(context.Background(), UpgradeOptions{
		Current: "v0.2.6", Target: dst, BaseURL: srv.URL + "/o/r/releases", Client: srv.Client(),
	})
	if err == nil {
		t.Fatal("expected an error")
	}
	if got := readFile(t, dst); got != oldBinary {
		t.Errorf("binary changed to %q", got)
	}
}

func TestUpgradeUnwritableDirectory(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("directory permissions don't stop this here")
	}
	archiveName := "sourceseedy-0.3.0_linux_amd64.tar.gz"
	archive := tarGz(t, map[string]string{"sourceseedy": "new binary"})
	rel := newRelease(t, map[string][]byte{
		archiveName:                    archive,
		"sourceseedy-0.3.0_SHA256SUMS": sums(map[string][]byte{archiveName: archive}),
	})
	dst := target(t)
	dir := filepath.Dir(dst)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	_, _, err := Upgrade(context.Background(), UpgradeOptions{
		Current: "v0.2.6", Target: dst, BaseURL: rel.url(), Client: rel.srv.Client(),
		GOOS: "linux", GOARCH: "amd64",
	})
	if err == nil || !strings.Contains(err.Error(), "can't write to "+dir) {
		t.Fatalf("error = %v, want one about not being able to write to %s", err, dir)
	}
	if got := readFile(t, dst); got != oldBinary {
		t.Errorf("binary changed to %q", got)
	}
}

func TestAssetNames(t *testing.T) {
	for _, tt := range []struct {
		tag, goos, goarch string
		archive           string
	}{
		{"v0.3.0", "linux", "amd64", "sourceseedy-0.3.0_linux_amd64.tar.gz"},
		{"v0.3.0", "linux", "arm64", "sourceseedy-0.3.0_linux_arm64.tar.gz"},
		{"v0.3.0", "darwin", "arm64", "sourceseedy-0.3.0_macOS_arm64.tar.gz"},
		{"v0.3.0", "darwin", "amd64", "sourceseedy-0.3.0_macOS_amd64.tar.gz"},
		{"v1.12.3", "windows", "amd64", "sourceseedy-1.12.3_windows_amd64.zip"},
	} {
		archive, sums, err := assetNames(tt.tag, tt.goos, tt.goarch)
		if err != nil {
			t.Fatal(err)
		}
		if archive != tt.archive {
			t.Errorf("%s %s/%s: archive = %q, want %q", tt.tag, tt.goos, tt.goarch, archive, tt.archive)
		}
		if want := "sourceseedy-" + strings.TrimPrefix(tt.tag, "v") + "_SHA256SUMS"; sums != want {
			t.Errorf("%s: sums = %q, want %q", tt.tag, sums, want)
		}
	}
	for _, tt := range [][2]string{{"freebsd", "amd64"}, {"linux", "386"}, {"linux", "riscv64"}} {
		if _, _, err := assetNames("v0.3.0", tt[0], tt[1]); err == nil {
			t.Errorf("%s/%s: expected an error", tt[0], tt[1])
		}
	}
}

func TestChecksumFor(t *testing.T) {
	text := "aaa  one.tar.gz\nbbb *two.tar.gz\r\nccc  three.tar.gz"
	for name, want := range map[string]string{"one.tar.gz": "aaa", "two.tar.gz": "bbb", "three.tar.gz": "ccc"} {
		got, err := checksumFor(text, name)
		if err != nil || got != want {
			t.Errorf("checksumFor(%q) = %q, %v, want %q", name, got, err, want)
		}
	}
	// A name that is only the end of another one doesn't count
	if _, err := checksumFor(text, "ne.tar.gz"); err == nil {
		t.Error("matched part of a name")
	}
}

func TestDownloadLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("x"), 100))
	}))
	defer srv.Close()
	var buf bytes.Buffer
	if err := download(context.Background(), srv.Client(), srv.URL+"/a.tar.gz", &buf, 100); err != nil {
		t.Errorf("exactly the limit: %v", err)
	}
	if err := download(context.Background(), srv.Client(), srv.URL+"/a.tar.gz", &buf, 99); err == nil {
		t.Error("over the limit: expected an error")
	}
}

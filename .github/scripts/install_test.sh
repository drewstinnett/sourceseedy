#!/usr/bin/env bash
# Tests install.sh against a fake releases page, served from a throwaway local
# web server. Needs python3 and curl. Run it from anywhere:
# .github/scripts/install_test.sh
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
installer=$(cd "$here/../.." && pwd)/install.sh
work=$(mktemp -d)
failed=0
server_pid=

# shellcheck disable=SC2317,SC2329 # run by the trap below. Older shellcheck calls this SC2317, newer SC2329
cleanup() {
  if [ -n "$server_pid" ]; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
  rm -rf "$work"
}
trap cleanup EXIT

case $(uname -s) in
  Darwin) os=macOS ;;
  Linux) os=linux ;;
  *) echo "these tests run on macOS or Linux"; exit 1 ;;
esac
case $(uname -m) in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) echo "unsupported architecture $(uname -m)"; exit 1 ;;
esac

sum() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | awk '{ print $1 }'; else shasum -a 256 "$1" | awk '{ print $1 }'; fi
}

# A release, like goreleaser makes them: sourceseedy at the root of a tar.gz
# for each build, and a checksums file. The binary is a script that says which
# release it is
make_release() {
  local tag=$1 dir=$work/site/releases/download/$1
  local version=${tag#v}
  mkdir -p "$dir" "$work/build"
  printf '#!/bin/sh\necho "stub %s"\n' "$tag" >"$work/build/sourceseedy"
  chmod 755 "$work/build/sourceseedy"
  : >"$dir/sourceseedy-${version}_SHA256SUMS"
  local b
  for b in linux_amd64 linux_arm64 macOS_amd64 macOS_arm64; do
    tar -czf "$dir/sourceseedy-${version}_$b.tar.gz" -C "$work/build" sourceseedy
    echo "$(sum "$dir/sourceseedy-${version}_$b.tar.gz")  sourceseedy-${version}_$b.tar.gz" >>"$dir/sourceseedy-${version}_SHA256SUMS"
  done
}

# Serves the fake site, and sends /releases/latest on to the newest tag, the way
# GitHub does
cat >"$work/server.py" <<'PY'
import http.server, os, socketserver, sys

root = sys.argv[1]

class Handler(http.server.SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=root, **kwargs)

    def redirect(self):
        self.send_response(302)
        self.send_header("Location", "/releases/tag/v9.9.9")
        self.end_headers()

    def do_HEAD(self):
        self.redirect() if self.path == "/releases/latest" else super().do_HEAD()

    def do_GET(self):
        self.redirect() if self.path == "/releases/latest" else super().do_GET()

    def log_message(self, *args):
        pass

with socketserver.TCPServer(("127.0.0.1", 0), Handler) as srv:
    print(srv.server_address[1], flush=True)
    srv.serve_forever()
PY

make_release v9.9.9
make_release v1.2.3
python3 "$work/server.py" "$work/site" >"$work/port" &
server_pid=$!
for _ in $(seq 100); do
  [ -s "$work/port" ] && break
  sleep 0.1
done
url=http://127.0.0.1:$(cat "$work/port")/releases

n=0
# run NAME EXPECT_EXIT [VAR=value...] runs the installer into a fresh directory,
# leaving its output in $work/out and the directory in $dest
run() {
  local name=$1 want=$2
  shift 2
  n=$((n + 1))
  dest=$work/dest$n
  local rc=0
  env SOURCESEEDY_RELEASE_URL="$url" SOURCESEEDY_INSTALL_DIR="$dest/bin" "$@" sh "$installer" >"$work/out" 2>&1 || rc=$?
  if [ "$rc" != "$want" ]; then
    echo "FAIL $name: exit $rc, want $want"
    sed 's/^/     /' "$work/out"
    failed=1
    return 1
  fi
}

ok() { echo "ok   $1"; }
bad() { echo "FAIL $1"; failed=1; }

# expect_out NAME TEXT checks the output of the last run has TEXT
expect_out() {
  if grep -qF -- "$2" "$work/out"; then ok "$1"; else bad "$1: output doesn't say '$2'"; sed 's/^/     /' "$work/out"; fi
}

if run "latest release" 0; then
  if [ "$("$dest/bin/sourceseedy")" = "stub v9.9.9" ]; then ok "installs the latest release"; else bad "installs the latest release: wrong binary"; fi
  expect_out "says where it went" "Installed sourceseedy v9.9.9 to $dest/bin/sourceseedy"
  # Nothing but the binary is left in the install directory
  if [ "$(ls -A "$dest/bin")" = "sourceseedy" ]; then ok "leaves nothing else behind"; else bad "leaves nothing else behind: $(ls -A "$dest/bin")"; fi
fi

if run "pinned version" 0 SOURCESEEDY_VERSION=v1.2.3; then
  if [ "$("$dest/bin/sourceseedy")" = "stub v1.2.3" ]; then ok "installs a pinned version"; else bad "installs a pinned version"; fi
fi

if run "pinned version without the v" 0 SOURCESEEDY_VERSION=1.2.3; then
  if [ "$("$dest/bin/sourceseedy")" = "stub v1.2.3" ]; then ok "takes a version without the v"; else bad "takes a version without the v"; fi
fi

if run "not on PATH" 0 PATH="/usr/bin:/bin"; then
  expect_out "warns when the directory isn't on PATH" "isn't on your PATH"
fi

if run "on PATH" 0 PATH="$work/dest$((n + 1))/bin:/usr/bin:/bin"; then
  if grep -q "isn't on your PATH" "$work/out"; then bad "no PATH warning when it is on PATH"; else ok "no PATH warning when it is on PATH"; fi
fi

# Installing again replaces what is there
if run "first install" 0 SOURCESEEDY_VERSION=v1.2.3; then
  first=$dest
  n=$((n - 1))
  if run "reinstall" 0 SOURCESEEDY_VERSION=v9.9.9 SOURCESEEDY_INSTALL_DIR="$first/bin"; then
    if [ "$("$first/bin/sourceseedy")" = "stub v9.9.9" ]; then ok "reinstall replaces the binary"; else bad "reinstall replaces the binary"; fi
  fi
fi

# A bad download installs nothing. The archive is changed after its checksum was
# made
make_release v0.0.1
bad_release=$work/site/releases/download/v0.0.1
printf 'tampered' >"$bad_release/sourceseedy-0.0.1_${os}_${arch}.tar.gz"
if run "checksum mismatch" 1 SOURCESEEDY_VERSION=v0.0.1; then
  expect_out "checksum mismatch is reported" "doesn't match its checksum"
  if [ -e "$dest/bin/sourceseedy" ]; then bad "checksum mismatch installs nothing"; else ok "checksum mismatch installs nothing"; fi
fi

# A checksum for a different file doesn't count
: >"$bad_release/sourceseedy-0.0.1_SHA256SUMS"
echo "$(sum "$work/build/sourceseedy")  x-sourceseedy-0.0.1_${os}_${arch}.tar.gz" >>"$bad_release/sourceseedy-0.0.1_SHA256SUMS"
if run "no checksum line" 1 SOURCESEEDY_VERSION=v0.0.1; then
  expect_out "a missing checksum is reported" "no checksum for sourceseedy-0.0.1_${os}_${arch}.tar.gz"
fi

if run "no such release" 1 SOURCESEEDY_VERSION=v8.8.8; then
  expect_out "a missing release is reported" "couldn't download"
  if [ -e "$dest/bin/sourceseedy" ]; then bad "a missing release installs nothing"; else ok "a missing release installs nothing"; fi
fi

# Pretend to be a machine that has no build, with a uname that lies
mkdir -p "$work/fake"
cat >"$work/fake/uname" <<'UNAME'
#!/bin/sh
case $1 in
  -s) echo "${FAKE_OS:-Linux}" ;;
  -m) echo "${FAKE_ARCH:-x86_64}" ;;
esac
UNAME
chmod 755 "$work/fake/uname"
if run "unsupported OS" 1 PATH="$work/fake:$PATH" FAKE_OS=FreeBSD; then
  expect_out "an unsupported OS is reported" "FreeBSD isn't supported"
fi
if run "unsupported architecture" 1 PATH="$work/fake:$PATH" FAKE_ARCH=riscv64; then
  expect_out "an unsupported architecture is reported" "riscv64 isn't supported"
fi
if run "arm linux name" 0 PATH="$work/fake:$PATH" FAKE_OS=Linux FAKE_ARCH=aarch64 SOURCESEEDY_VERSION=v9.9.9; then
  ok "aarch64 is taken as arm64"
fi

exit $failed

#!/bin/sh
# Installs sourceseedy from its latest GitHub release:
#
#   curl -fsSL https://raw.githubusercontent.com/drewstinnett/sourceseedy/main/install.sh | sh
#
# It works out the OS and architecture, downloads that build, checks it against
# the checksums published with the release, and puts sourceseedy in
# ~/.local/bin. Change what it does with environment variables, which go on the
# sh side of the pipe:
#
#   curl -fsSL .../install.sh | SOURCESEEDY_INSTALL_DIR=/usr/local/bin sh
#
#   SOURCESEEDY_VERSION      release to install, like v0.3.0. Default is the latest
#   SOURCESEEDY_INSTALL_DIR  where to put the binary. Default is ~/.local/bin
#
# SOURCESEEDY_RELEASE_URL is where releases are looked for, which is only
# changed to test this script.
#
# macOS and Linux only. For Windows, download the .zip from the releases page.
# Once installed, `sourceseedy upgrade` keeps it up to date.

# Everything is in a function that is only run on the last line, so a download
# that is cut short can't run half of it
main() {
  set -eu

  releases=${SOURCESEEDY_RELEASE_URL:-https://github.com/drewstinnett/sourceseedy/releases}
  install_dir=${SOURCESEEDY_INSTALL_DIR:-$HOME/.local/bin}

  case $(uname -s) in
    Darwin) os=macOS ;;
    Linux) os=linux ;;
    *) fail "$(uname -s) isn't supported by this script. Windows builds are on $releases" ;;
  esac
  case $(uname -m) in
    x86_64 | amd64) arch=amd64 ;;
    arm64 | aarch64) arch=arm64 ;;
    *) fail "$(uname -m) isn't supported. Builds are on $releases" ;;
  esac

  tag=${SOURCESEEDY_VERSION:-}
  if [ -z "$tag" ]; then
    say "Finding the latest release"
    tag=$(latest_tag "$releases")
  fi
  case $tag in
    v*) ;;
    *) tag=v$tag ;;
  esac
  version=${tag#v}

  archive=sourceseedy-${version}_${os}_${arch}.tar.gz
  sums=sourceseedy-${version}_SHA256SUMS

  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT INT TERM

  say "Downloading sourceseedy $tag for $os/$arch"
  download "$releases/download/$tag/$sums" "$tmp/$sums"
  download "$releases/download/$tag/$archive" "$tmp/$archive"

  # Each line is "<hash>  <file>". The name is matched whole, since a substring
  # could be a different build
  want=$(awk -v f="$archive" '$2 == f || $2 == "*" f { print $1 }' "$tmp/$sums")
  [ -n "$want" ] || fail "no checksum for $archive in $sums"
  got=$(sha256 "$tmp/$archive")
  [ "$got" = "$want" ] || fail "$archive doesn't match its checksum (got $got, want $want)"

  tar -xzf "$tmp/$archive" -C "$tmp" sourceseedy
  mkdir -p "$install_dir"
  # Copied next to where it is going, then moved, so a sourceseedy that is
  # running is never overwritten half way
  cp "$tmp/sourceseedy" "$install_dir/.sourceseedy.$$"
  chmod 755 "$install_dir/.sourceseedy.$$"
  mv -f "$install_dir/.sourceseedy.$$" "$install_dir/sourceseedy"

  say "Installed sourceseedy $tag to $install_dir/sourceseedy"
  case :$PATH: in
    *":$install_dir:"*) ;;
    *) say "$install_dir isn't on your PATH. Add it with: export PATH=\"$install_dir:\$PATH\"" ;;
  esac
  say "For the scd command, add this to your shell config: eval \"\$(sourceseedy init zsh)\""
  say "Upgrade later with: sourceseedy upgrade"
}

say() { printf '%s\n' "$*"; }

fail() {
  printf 'install.sh: %s\n' "$*" >&2
  exit 1
}

# download URL FILE
download() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL -o "$2" "$1" || fail "couldn't download $1"
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O "$2" "$1" || fail "couldn't download $1"
  else
    fail "curl or wget is needed"
  fi
}

# latest_tag RELEASES prints the tag of the latest release. GitHub sends
# <releases>/latest on to <releases>/tag/<tag>, so this reads that redirect,
# rather than using the API, which limits how often an address can use it
latest_tag() {
  if command -v curl >/dev/null 2>&1; then
    url=$(curl -fsSI -o /dev/null -w '%{redirect_url}' "$1/latest") || fail "couldn't look up the latest release"
  elif command -v wget >/dev/null 2>&1; then
    url=$(wget -q --max-redirect=0 -S -O /dev/null "$1/latest" 2>&1 | awk 'tolower($1) == "location:" { print $2 }' | tr -d '\r') || true
  else
    fail "curl or wget is needed"
  fi
  case $url in
    */tag/?*) printf '%s\n' "${url##*/}" ;;
    *) fail "couldn't work out the latest release from '$url'" ;;
  esac
}

# sha256 FILE prints the checksum of the file
sha256() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{ print $1 }'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{ print $1 }'
  else
    fail "sha256sum or shasum is needed to check the download"
  fi
}

main "$@"

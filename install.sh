#!/bin/sh
# Install tline from GitHub releases.
#
#   curl -fsSL https://raw.githubusercontent.com/jbadhree/timeline/main/install.sh | sh
#
# Settings, as environment variables:
#   TLINE_VERSION       release to install, such as v0.2.0 (default: the newest)
#   TLINE_INSTALL_DIR   where to put tline (default: ~/.local/bin)
#   GITHUB_TOKEN        optional; used to find the newest release, avoiding
#                       GitHub's limit on anonymous requests
#
# It downloads the archive for this OS and CPU, checks it against the
# release's SHA256SUMS, and installs it without needing sudo.

set -eu

REPO="jbadhree/timeline"

say() { printf '%s\n' "$*" >&2; }
fail() { say "tline install: $*"; exit 1; }

# fetch URL FILE downloads URL to FILE with curl or wget.
fetch() {
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$1" -o "$2"
	elif command -v wget >/dev/null 2>&1; then
		wget -q "$1" -O "$2"
	else
		fail "curl or wget is required"
	fi
}

# fetch_api URL FILE fetches a GitHub API URL, with GITHUB_TOKEN if set.
fetch_api() {
	if [ -z "${GITHUB_TOKEN:-}" ]; then
		fetch "$1" "$2"
	elif command -v curl >/dev/null 2>&1; then
		curl -fsSL -H "Authorization: Bearer $GITHUB_TOKEN" "$1" -o "$2"
	else
		wget -q --header="Authorization: Bearer $GITHUB_TOKEN" "$1" -O "$2"
	fi
}

case "$(uname -s)" in
Darwin) os=darwin ;;
Linux) os=linux ;;
*) fail "unsupported operating system $(uname -s); tline supports macOS and Linux" ;;
esac

case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) fail "unsupported CPU $(uname -m); tline supports amd64 and arm64" ;;
esac

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

version="${TLINE_VERSION:-}"
if [ -z "$version" ]; then
	# The releases API includes prereleases, which /releases/latest skips.
	fetch_api "https://api.github.com/repos/$REPO/releases?per_page=1" "$tmp/releases.json" ||
		fail "could not ask GitHub for the newest release"
	version="$(sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' "$tmp/releases.json" | head -n 1)"
	[ -n "$version" ] || fail "could not find a release of $REPO"
fi

name="tline_${version}_${os}_${arch}"
base="https://github.com/$REPO/releases/download/$version"
say "Downloading tline $version for $os/$arch"
fetch "$base/$name.tar.gz" "$tmp/$name.tar.gz" || fail "could not download $base/$name.tar.gz"
fetch "$base/SHA256SUMS" "$tmp/SHA256SUMS" || fail "could not download $base/SHA256SUMS"

expected="$(awk -v f="$name.tar.gz" '$2 == f { print $1 }' "$tmp/SHA256SUMS")"
if command -v sha256sum >/dev/null 2>&1; then
	actual="$(sha256sum "$tmp/$name.tar.gz" | awk '{ print $1 }')"
else
	actual="$(shasum -a 256 "$tmp/$name.tar.gz" | awk '{ print $1 }')"
fi
[ -n "$expected" ] || fail "SHA256SUMS has no entry for $name.tar.gz"
[ "$expected" = "$actual" ] || fail "checksum mismatch for $name.tar.gz; not installing"

tar -xzf "$tmp/$name.tar.gz" -C "$tmp"
dir="${TLINE_INSTALL_DIR:-$HOME/.local/bin}"
mkdir -p "$dir"
# Copy, then rename, so a running tline is never left half-written.
cp "$tmp/$name/tline" "$dir/.tline.new"
chmod 0755 "$dir/.tline.new"
mv -f "$dir/.tline.new" "$dir/tline"
if [ "$os" = darwin ]; then
	xattr -d com.apple.quarantine "$dir/tline" 2>/dev/null || true
fi

say "Installed $("$dir/tline" --version) to $dir/tline"
case ":$PATH:" in
*":$dir:"*) ;;
*) say "Add $dir to your PATH to run tline from anywhere, for example:
  echo 'export PATH=\"$dir:\$PATH\"' >> ~/.zshrc" ;;
esac

#!/bin/sh
# Install the latest nanaoki release binary.
#   curl -fsSL https://raw.githubusercontent.com/JustinyAhin/nanaoki/main/install.sh | sh
# Options (env): NANAOKI_VERSION=v0.1.0 to pin a version, NANAOKI_BIN_DIR=~/.local/bin to choose the folder.
set -eu

REPO="JustinyAhin/nanaoki"
BIN_DIR="${NANAOKI_BIN_DIR:-$HOME/.local/bin}"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in darwin|linux) ;; *) echo "nanaoki: unsupported OS $os (use the Windows zip from the releases page)" >&2; exit 1 ;; esac
arch=$(uname -m)
case "$arch" in x86_64|amd64) arch=amd64 ;; arm64|aarch64) arch=arm64 ;; *) echo "nanaoki: unsupported CPU $arch" >&2; exit 1 ;; esac

version="${NANAOKI_VERSION:-}"
if [ -z "$version" ]; then
  version=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)
fi
[ -n "$version" ] || { echo "nanaoki: could not find the latest release" >&2; exit 1; }

file="nanaoki_${version#v}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/$version"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading nanaoki $version ($os/$arch)..."
curl -fsSL -o "$tmp/$file" "$base/$file"
curl -fsSL -o "$tmp/checksums.txt" "$base/checksums.txt"

expected=$(grep " $file\$" "$tmp/checksums.txt" | awk '{print $1}')
if command -v shasum >/dev/null 2>&1; then actual=$(shasum -a 256 "$tmp/$file" | awk '{print $1}')
else actual=$(sha256sum "$tmp/$file" | awk '{print $1}'); fi
[ "$expected" = "$actual" ] || { echo "nanaoki: checksum mismatch for $file" >&2; exit 1; }

tar -xzf "$tmp/$file" -C "$tmp"
mkdir -p "$BIN_DIR"
install -m 0755 "$tmp/nanaoki" "$BIN_DIR/nanaoki"

echo "Installed $BIN_DIR/nanaoki"
case ":$PATH:" in *":$BIN_DIR:"*) ;; *) echo "Add $BIN_DIR to your PATH." ;; esac
echo "Next: nanaoki --split auto      (your numbers)"
echo "      nanaoki setup claude        (plugin for Claude Code)"
echo "      nanaoki setup codex         (hooks for Codex)"

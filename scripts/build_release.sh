#!/usr/bin/env bash
# Builds one program per system and checksums.txt: build_release.sh <tag> <folder>.
set -euo pipefail
cd "$(dirname "$0")/.."

tag="$1"
out="$2"
number="$(sed -n 's/^version = "\(.*\)"$/\1/p' version.toml)"
if [ "$tag" != "v$number" ]; then
  echo "tag $tag does not match version.toml $number" >&2
  exit 1
fi

mkdir -p "$out"
for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
  os="${target%/*}"
  arch="${target#*/}"
  name="version-gate-$os-$arch"
  if [ "$os" = windows ]; then
    name="$name.exe"
  fi
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
    go build -trimpath -ldflags "-s -w -X main.version=$number" -o "$out/$name" ./cmd/version-gate
done
(cd "$out" && shasum -a 256 version-gate-* > checksums.txt)

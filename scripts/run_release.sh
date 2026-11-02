#!/usr/bin/env bash
# Downloads one exact release, verifies its checksum and runs it: run_release.sh <tag> <base ref>.
set -euo pipefail

tag="$1"
base="$2"
url="https://github.com/bystrovmaxim/version-gate/releases/download/$tag"

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  MINGW* | MSYS* | CYGWIN*) os=windows ;;
  *) echo "unsupported system $(uname -s)" >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) echo "unsupported machine $(uname -m)" >&2; exit 1 ;;
esac
name="version-gate-$os-$arch"
if [ "$os" = windows ]; then
  name="$name.exe"
fi

folder="$(mktemp -d)"
trap 'rm -rf "$folder"' EXIT
curl --fail --silent --show-error --location --retry 3 -o "$folder/$name" "$url/$name"
curl --fail --silent --show-error --location --retry 3 -o "$folder/checksums.txt" "$url/checksums.txt"
(cd "$folder" && grep " $name\$" checksums.txt | shasum -a 256 -c -)
chmod +x "$folder/$name"
"$folder/$name" check --base "$base"

#!/usr/bin/env bash
# The repository's check run: formatting, vet, tests, and a build for every system.
set -euo pipefail
cd "$(dirname "$0")/.."

unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
  echo "gofmt needed:" >&2
  echo "$unformatted" >&2
  exit 1
fi

go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./...
go test ./...

out="$(mktemp -d)"
trap 'rm -rf "$out"' EXIT
for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
  CGO_ENABLED=0 GOOS="${target%/*}" GOARCH="${target#*/}" \
    go build -o "$out/version-gate-${target%/*}-${target#*/}" ./cmd/version-gate
done

echo "All checks passed."

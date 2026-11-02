#!/usr/bin/env bash
# Runs every case of docs/cases.md in a throwaway repository and prints what the tool answers.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
(cd "$root" && go build -o "$work/version-gate" ./cmd/version-gate)

g() { git -c user.name=demo -c user.email=demo@example.com -c commit.gpgsign=false "$@"; }

# start creates a repository with a package and two components, and a branch named pr.
start() {
  rm -rf "$work/repo"
  mkdir -p "$work/repo/p/a" "$work/repo/p/b"
  cd "$work/repo"
  g init -q -b main
  printf 'kind = "repository"\nversion = "1.0.0"\n' > version.toml
  printf 'kind = "package"\ndriver = "node"\n' > p/version.toml
  printf '{"version": "1.0.0"}\n' > p/package.json
  printf 'kind = "component"\nversion = "1.0.0"\n' > p/a/version.toml
  printf 'kind = "component"\nversion = "1.0.0"\n' > p/b/version.toml
  echo one > p/a/code.js
  echo two > p/b/code.js
  g add -A
  g commit -q -m main
  g switch -q -c pr
}

# raise sets the three versions a change inside component a needs.
raise() {
  printf 'kind = "repository"\nversion = "%s"\n' "$1" > version.toml
  printf '{"version": "%s"}\n' "$1" > p/package.json
  printf 'kind = "component"\nversion = "%s"\n' "$1" > p/a/version.toml
}

# verdict commits and prints the command's answer.
verdict() {
  echo "### $1"
  g add -A
  g commit -q --allow-empty -m pr
  local code=0
  "$work/version-gate" check --base main > "$work/out" 2>&1 || code=$?
  grep -v '^tracked:' "$work/out" || true
  echo "exit code: $code"
  echo
}

start; raise 1.0.1; echo x >> p/a/code.js
verdict "Every version grew"

start; echo x >> p/a/code.js
verdict "No version grew"

start; raise 1.0.1; printf 'kind = "component"\nversion = "1.0.0"\n' > p/a/version.toml; echo x >> p/a/code.js
verdict "One version did not grow"

start; raise 0.9.0; echo x >> p/a/code.js
verdict "A version went down"

start; printf 'kind = "repository"\nversion = "1.0.0"\n\n[changes]\nignore = ["**/*.js"]\n' > version.toml; echo x >> p/a/code.js
verdict "A new ignore pattern does not cover its own pull request"

start; raise 1.0.1; g mv p/a/code.js p/a/moved.js; g mv p/b/code.js p/a/code.js
verdict "A file moved between components"

start; g rm -q p/a/version.toml; echo x >> p/a/code.js
verdict "A level removed while its code changes"

start; g rm -q p/a/version.toml; printf 'kind = "repository"\nversion = "1.0.1"\n' > version.toml; printf '{"version": "1.0.1"}\n' > p/package.json
verdict "A level removed in a pull request of its own"

start; mkdir -p q; echo '{}' > q/package.json
verdict "A package file with no version.toml"

start; mkdir -p p/a/inner; printf 'kind = "component"\nversion = "1.0.0"\n' > p/a/inner/version.toml
verdict "A component inside a component"

start; printf 'kind = "component"\nversion = "1.0.1"\nchangelog = "CHANGELOG.md"\n' > p/a/version.toml; printf 'kind = "repository"\nversion = "1.0.1"\n' > version.toml; printf '{"version": "1.0.1"}\n' > p/package.json; echo x >> p/a/code.js
verdict "A version grew with no changelog entry"

start; raise one; echo x >> p/a/code.js
verdict "A number that is not valid"

# Changelog

## [0.1.0]

version-gate now checks that versions really grow when code changes, for a project in any language. Add a small `version.toml` beside what you want tracked, and run `version-gate check` on pull requests into `main`: if a file changed and a version it belongs to did not go up, the check fails and names the file to edit.

It works at three sizes: a repository, a package, and a component, which is any folder you treat as a solution of its own. Nine languages are read directly, and any other project can write its number in `version.toml`. A level can also name a changelog, and then raising its version needs an entry.

A pull request cannot talk its way out of the check: the rules are read from `main`, and deleting `version.toml`, hiding a change behind a new ignore pattern, or moving a file between components all still fail.

Two things are yours to set up: require branches to be up to date before merging, and fetch full history in CI. No ready-made program or GitHub Action is published yet, so for now it is built from source.

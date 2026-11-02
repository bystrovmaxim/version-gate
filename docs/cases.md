# Every case

Each case below is run by [`examples/cases.sh`](../examples/cases.sh) in a throwaway repository, and what you see is what it printed. The repository has a root, package `p`, and components `p/a` and `p/b`, all at `1.0.0`.

## Every version grew

A file changed inside component `p/a`, and the component, its package and the repository all grew.

```text
every required version grew
exit code: 0
```

## No version grew

The same change with nothing raised. Every level the file belongs to is named, with the file to edit.

```text
version-gate: repository . (version.toml): version 1.0.0 did not grow; main has 1.0.0, raise it above that
version-gate: package p (p/package.json): version 1.0.0 did not grow; main has 1.0.0, raise it above that
version-gate: component p/a (p/a/version.toml): version 1.0.0 did not grow; main has 1.0.0, raise it above that
exit code: 1
```

## One version did not grow

Only the level at fault is named.

```text
version-gate: component p/a (p/a/version.toml): version 1.0.0 did not grow; main has 1.0.0, raise it above that
exit code: 1
```

## A version went down

A version that changed but fell is not growth.

```text
version-gate: repository . (version.toml): version 0.9.0 did not grow; main has 1.0.0, raise it above that
version-gate: package p (p/package.json): version 0.9.0 did not grow; main has 1.0.0, raise it above that
version-gate: component p/a (p/a/version.toml): version 0.9.0 did not grow; main has 1.0.0, raise it above that
exit code: 1
```

## A new ignore pattern does not cover its own pull request

`changes.ignore` is read from `main`, so a pattern works only after it is merged.

```text
version-gate: repository . (version.toml): version 1.0.0 did not grow; main has 1.0.0, raise it above that
version-gate: package p (p/package.json): version 1.0.0 did not grow; main has 1.0.0, raise it above that
version-gate: component p/a (p/a/version.toml): version 1.0.0 did not grow; main has 1.0.0, raise it above that
exit code: 1
```

## A file moved between components

Renames are off: a move is a deletion in `p/a` and an addition in `p/b`, so both must grow.

```text
version-gate: component p/b (p/b/version.toml): version 1.0.0 did not grow; main has 1.0.0, raise it above that
exit code: 1
```

## A level removed while its code changes

Deleting `version.toml` cannot remove the duty to raise the version.

```text
version-gate: p/a/version.toml removes level p/a and also changes p/a/code.js; remove a level in a pull request of its own
exit code: 1
```

## A level removed in a pull request of its own

With no other file of the level touched, the level goes, and the levels above must grow.

```text
every required version grew
exit code: 0
```

## A package file with no version.toml

So a level is never forgotten. List a deliberate exception under `untracked`, in a pull request merged first.

```text
version-gate: q/package.json has no version.toml beside it; add one, or list q under untracked
exit code: 1
```

## A component inside a component

Components cannot nest.

```text
version-gate: component p/a/inner lies inside component p/a
exit code: 1
```

## A version grew with no changelog entry

The level names a `changelog`, so its file must change in this pull request.

```text
version-gate: component p/a (p/a/version.toml): version grew to 1.0.1, but p/a/CHANGELOG.md did not change; add an entry
exit code: 1
```

## A number that is not valid

A number must be valid under the level's scheme.

```text
version-gate: repository . (version.toml): "one" is not a semver number
version-gate: package p (p/package.json): "one" is not a semver number
version-gate: component p/a (p/a/version.toml): "one" is not a semver number
exit code: 1
```

## Adopting the tool

With no root `version.toml` on `main`, every level counts as new: the pull request passes, and its numbers are only validated. A new level added later works the same way.

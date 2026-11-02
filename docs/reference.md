# Reference

## `version.toml`

A level is tracked only if a `version.toml` lies in its folder. Levels are worked out from folder paths.

| Key | Where | Meaning |
|---|---|---|
| `kind` | every file | `repository`, `package` or `component`. `repository` is allowed only in the root |
| `version` | every file | The number, written directly. Required for a component |
| `driver` | package, repository | The language that holds the number. Exactly one of `driver` and `version` |
| `file`, `key` | with `driver` | Replace the driver's defaults |
| `changelog` | every file | Path to a changelog, relative to the file's folder |
| `require_bump` | root | `false` frees the repository version only. Default `true` |
| `untracked` | root | Folders that need no `version.toml`. A change there still counts for the levels above |
| `scheme` | root | The scheme of direct numbers with no driver above |
| `[changes] ignore` | root | Path patterns that do not count as changes. `**` crosses folders; `*` and `?` do not |
| `[setup]` | root | Reserved and ignored |

Any other key fails, so a misspelled option is not silently ignored. A component cannot lie inside a component. A package may lie inside a package; a file inside then requires every package above it to grow.

## Drivers

| Driver | File | Key | Scheme |
|---|---|---|---|
| `python` | `pyproject.toml` | `project.version` | PEP 440 |
| `rust` | `Cargo.toml` | `package.version` | semver |
| `node` | `package.json` | `version` | semver |
| `php` | `composer.json` | `version` | semver |
| `dart` | `pubspec.yaml` | `version` | semver |
| `maven` | `pom.xml` | `project/version` | Maven |
| `gradle` | `gradle.properties` | `version` | Maven |
| `dotnet` | `*.csproj` | `Project/PropertyGroup/Version` | NuGet |
| `ruby` | `lib/**/version.rb` | `VERSION` | RubyGems |

Go and Swift keep versions in git tags, so write `version` directly.

The tool never runs project code. A computed version therefore fails with a message: Python `dynamic`, Maven `${revision}`, .NET `$(Base)`, a Ruby expression. Write the number directly in `version.toml`; it is then the level's own number, and keeping it equal to what the build produces is your job. Rust `version.workspace = true` and a Maven parent's version are read.

A direct number uses the scheme of the nearest level above with a `driver`, otherwise the root `scheme`, otherwise semver.

## Which version must grow

| Where the file changed | What must grow |
|---|---|
| inside a component | component, every package above, repository |
| inside a package, outside any component | every package above, repository |
| outside all packages | repository |

Changed files are counted from the point where the branch left `main`, including deletions and moves. Versions are compared with the current tip of `main`. `changes.ignore`, `untracked` and `require_bump` are read from `main`.

## The changelog

When a version grew and its level names a `changelog`, the file must have changed in this pull request, and its top section must be the new version or a non-empty `[Unreleased]`, in [Keep a Changelog](https://keepachangelog.com) headings. A level with no `changelog` is not checked. The tool proves an entry exists, not that it is well written.

## What you must set up yourself

- **Up-to-date branches.** A green check goes stale when `main` moves. Require a branch to be up to date before merging, or use a merge queue. The tool cannot enforce this.
- **Full history in CI.** Use `fetch-depth: 0` in `actions/checkout`, and pass the base ref.
- **Only pull requests into `main`.** Run the check on those events; the tool does not look at the CI event.

## Known limits

- **Maven:** numbers are compared as a flat row of tokens, not as Maven's nested lists, so unusual forms such as `1-1` against `1.1` may order differently. Not compared against Maven itself.
- **Dart:** `1.2.4+5` to `1.2.4+6` is not growth, because semver ignores build metadata. Raise the main part.
- **.NET:** a version in `Directory.Build.props` is not found by default; name the file with `file`, or write the number directly.
- **Gradle:** `gradle.properties` is read line by line, with no continuation lines or escapes.

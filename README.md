# version-gate

**Fail CI when code changes without a version bump.**

`version-gate` checks a pull request against its base branch and tells you which versions need to increase. Use it for a single project or a monorepo with separately versioned packages and folders, in any language. It reads files and Git history without running your project code.

[Get started](#get-started) · [GitHub Actions](#github-actions) · [Monorepos](#monorepos-and-components) · [Reference](docs/reference.md) · [Example results](docs/cases.md)

## Get started

### 1. Install

You need Git and Go **1.27.3 or newer** to install from source:

```bash
go install github.com/bystrovmaxim/version-gate/cmd/version-gate@latest
```

Make sure Go's binary directory (`GOBIN`, or `$(go env GOPATH)/bin` by default) is on your `PATH`. Check that the command is available:

```bash
version-gate --help
```

### 2. Add `version.toml` to your repository root

For a Python project with a version in `pyproject.toml`:

```toml
kind = "repository"
driver = "python"
```

This reads `[project].version` from `pyproject.toml`. For other ecosystems, choose a [supported driver](#supported-projects).

Alternatively, keep the version directly in `version.toml`. This works for any language, including Go and Swift:

```toml
kind = "repository"
version = "1.0.0"
```

Choose **either `driver` or `version`**, not both.

### 3. Commit your changes and run the check

From your repository root, on a branch targeting `main`:

```bash
git fetch origin main
version-gate check --base origin/main
```

**The check reads committed files at `HEAD`.** Commit your configuration, code changes and version updates before running it. Your clone must have full Git history; for a shallow clone, run `git fetch --unshallow origin` first.

When you first add `version.toml`, the check validates the initial versions without requiring a bump. Once that setup is merged into `main`, subsequent changes require the affected versions to be higher than those on `main`.

For example, if `main` has version `1.0.0`, a code change with the same version fails:

```text
version-gate: repository . (version.toml): version 1.0.0 did not grow; main has 1.0.0, raise it above that
```

Update the version to, for example, `1.0.1`, commit it and rerun. A successful check ends with:

```text
every required version grew
```

Exit code `0` means success; `1` means the check failed; `2` means invalid command usage. The tool checks that the version increased; it does not choose a patch, minor or major release for you.

## GitHub Actions

Add `.github/workflows/version-gate.yml` to the project you want to check:

```yaml
name: Version gate

on:
  pull_request:
    branches: [main]

permissions:
  contents: read

jobs:
  version:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: bystrovmaxim/version-gate@v0.1.1
```

The action downloads that exact release, verifies its checksum and runs the check, so your CI needs no Go. In another CI, download the program for your system and `checksums.txt` from [Releases](https://github.com/bystrovmaxim/version-gate/releases).

Make the `version` job a required check in your branch protection or ruleset, and require branches to be up to date before merging. A result can become stale when another pull request updates the version on `main`.

## Supported projects

Set `driver` to read the version from an existing project file:

| Project | Driver | Default version source |
| --- | --- | --- |
| Python | `python` | `pyproject.toml` → `project.version` |
| Rust | `rust` | `Cargo.toml` → `package.version` |
| Node.js | `node` | `package.json` → `version` |
| PHP | `php` | `composer.json` → `version` |
| Dart | `dart` | `pubspec.yaml` → `version` |
| Maven | `maven` | `pom.xml` → `project/version` |
| Gradle | `gradle` | `gradle.properties` → `version` |
| .NET | `dotnet` | `*.csproj` → `Project/PropertyGroup/Version` |
| Ruby | `ruby` | `lib/**/version.rb` → `VERSION` |
| Any other project | Use `version` directly | `version.toml` → `version` |

Drivers use their ecosystem's version comparison rules. Direct versions use semver by default, or inherit the scheme of a parent driver. Computed versions that require executing code are unsupported; use a direct `version` instead. See the [reference](docs/reference.md) for source overrides, version schemes and driver limitations.

## Monorepos and components

Add a `version.toml` to each folder you want to version independently:

| Kind | Where it goes | What it tracks |
| --- | --- | --- |
| `repository` | Repository root | The whole repository |
| `package` | Package folder | A package and everything inside it |
| `component` | A folder inside the repository or a package | A separately versioned part that is not a package |

For example:

```text
version.toml                  # kind = "repository"
packages/
  api/
    version.toml              # kind = "package"
    package.json
    auth/
      version.toml            # kind = "component"
      handler.js
```

In `packages/api/version.toml`, read the package's version from `package.json`:

```toml
kind = "package"
driver = "node"
```

In `packages/api/auth/version.toml`, give the component its own version:

```toml
kind = "component"
version = "1.2.0"
```

Changing `handler.js` requires **all three versions** to increase: the component, its package and the repository. Unchanged sibling packages do not need a bump. Components cannot contain other components.

Once the tool is adopted, recognized package files must have a `version.toml` beside them. Use the root `untracked` option for intentional exceptions; changes in those folders still count toward their parent versions.

## Customize the checks

To exclude documentation changes, add this to the root `version.toml`:

```toml
[changes]
ignore = ["**/*.md", "docs/**"]
```

Other common options:

- **Require a changelog entry:** add `changelog = "CHANGELOG.md"` next to a level's `kind` and version source. When its version increases, the changelog must change and start with the new version or a non-empty `[Unreleased]` section.
- **Version packages without bumping the repository:** add `require_bump = false` at the top level of the root file.
- **Allow a folder without its own version file:** add `untracked = ["examples/demo"]` at the top level of the root file.

The check uses `changes.ignore`, `require_bump` and `untracked` from the **base branch**. A policy change takes effect after it is merged; it cannot exempt its own pull request.

## How the comparison works

- Changed paths come from the diff between the branch's common ancestor with the base and `HEAD`.
- Versions are compared against the **current tip of the base branch**, not the common ancestor.
- Deletions count as changes. Moving a file between components affects both components.
- Every enclosing tracked folder must increase its version, unless a configured exception applies.

See the [full reference](docs/reference.md) for configuration and limitations, or [example results](docs/cases.md) for successful checks and failure messages.

## Development

From a local checkout of this repository:

```bash
go build -o version-gate ./cmd/version-gate
go test ./...
```

Run `bash scripts/check.sh` for the full set of formatting, vet, static analysis, test and cross-platform build checks.

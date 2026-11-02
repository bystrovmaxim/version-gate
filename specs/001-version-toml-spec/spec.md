# Feature Specification: The `version.toml` format and the version-growth rules

**Feature Branch**: `feature/issue-1-version-gate`
**Created**: 2026-11-02
**Status**: Draft
**Input**: Phase 2 of issue #1 (part of bystrovmaxim/aoa#340): the format of `version.toml`, with passing and failing samples, including changes to the configuration itself.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Declare what is versioned (Priority: P1)

A project maintainer adds `version.toml` files, and every repository, package and component they want tracked is recognised from its folder alone.

**Why this priority**: Everything else reads this format, and bystrovmaxim/aoa#344 depends on it.

**Independent Test**: Validate each file in `samples/`; every valid one is accepted and every invalid one is refused with the reason listed in `samples/README.md`.

**Acceptance Scenarios**:

1. **Given** a component file with `kind` and `version`, **When** it is validated, **Then** it is accepted and the component takes its folder's name.
2. **Given** a component folder inside another component, **When** levels are worked out, **Then** the run fails and names both folders.
3. **Given** a `pyproject.toml` with no `version.toml` beside it and no `untracked` entry, **When** the tool runs, **Then** the run fails and names the folder.

### User Story 2 - Fail a pull request whose version did not grow (Priority: P1)

A contributor opens a pull request into `main`; if a level their files belong to did not raise its version, the check fails and says which one.

**Why this priority**: This is the purpose of the tool.

**Independent Test**: Apply each rule to a pair of `main` and pull-request states and compare the verdict with the expected one.

**Acceptance Scenarios**:

1. **Given** a changed file inside a component, **When** only the component's version grew, **Then** the check fails and names the package and the repository.
2. **Given** a version that changed but went down, **When** it is compared, **Then** the check fails.
3. **Given** two pull requests raising `1.2.0` to `1.2.1`, **When** the first is merged and the second is rechecked, **Then** the second fails, because versions are compared with the tip of `main`.
4. **Given** a pull request changing only ignored files, **When** it is checked, **Then** no version has to grow.

### User Story 3 - The rules cannot be bent by the pull request they check (Priority: P1)

A pull request cannot escape its duty by editing the configuration that checks it.

**Why this priority**: A check that its subject can switch off proves nothing.

**Independent Test**: The three adversarial cases of issue #1 each fail: deleting `version.toml` while changing code; hiding a change behind a new `changes.ignore`; moving a file between components with one version raised.

**Acceptance Scenarios**:

1. **Given** a level declared on `main`, **When** the pull request deletes its `version.toml` and changes files in that folder, **Then** the check fails.
2. **Given** a pull request adding an ignore pattern, **When** it changes a matching file, **Then** that file still counts.
3. **Given** a file moved from component A to component B, **When** only B grew, **Then** the check fails and names A.

### User Story 4 - A changelog entry goes with the version (Priority: P2)

A level that names a `changelog` cannot raise its version without an entry.

**Why this priority**: Optional per level, and it builds on the version rule.

**Independent Test**: Check changelog states against a grown version.

**Acceptance Scenarios**:

1. **Given** a grown version and an unchanged changelog, **When** checked, **Then** it fails, even with an old `[Unreleased]` section.
2. **Given** a changed changelog whose top section is the new version, or a non-empty `[Unreleased]`, **When** checked, **Then** it passes.

### Edge Cases

- Adoption: with no root file on `main`, the tool only validates files, so the first pull request passes.
- A new level has nothing to compare with, so any valid number passes.
- If a whole folder is gone, its level is gone, and the levels above must grow.
- A computed or inherited version is read from its parent file (Rust, Maven) or fails with a clear message.
- A missing referenced file or key fails with a clear message.
- A change of `driver`, `file` or `key`: the old number is read by `main`'s rules, the new one by the pull request's, and it must still grow.
- Semver build metadata alone is not growth; `1.0.0-rc.1` to `1.0.0` is.

## Requirements *(mandatory)*

### Functional Requirements

**Format**

- **FR-001**: A level is tracked only if a file named `version.toml` lies in its folder; `kind` MUST be `repository`, `package` or `component`.
- **FR-002**: `kind = "repository"` MUST appear only in the repository root, and a tracked repository's root MUST carry it.
- **FR-003**: A `component` MUST have `version`, and MAY have `changelog`. Its name is its folder's name.
- **FR-004**: A `package` or `repository` MUST take its number from exactly one source: `driver`, or a direct `version`. `file` and `key` MAY accompany `driver` only.
- **FR-005**: `driver` MUST be one of `python`, `rust`, `node`, `php`, `dart`, `maven`, `gradle`, `dotnet`, `ruby`.
- **FR-006**: `require_bump`, `untracked`, `scheme`, `[changes]` and `[setup]` MUST appear only in the root file. `require_bump` defaults to true, `changes.ignore` to an empty list. `[setup]` is reserved and ignored.
- **FR-007**: `scheme` MUST be one of `semver`, `pep440`, `maven`, `nuget`, `rubygems`.
- **FR-008**: Any other key MUST fail validation, so a misspelled option is not silently ignored.
- **FR-009**: `changelog` is a path relative to the file's folder.

**Levels**

- **FR-010**: Levels MUST be worked out from folder paths alone.
- **FR-011**: A component inside another component MUST fail.
- **FR-012**: A package MAY lie inside another package. A file inside it belongs to every package above it, and each of them must grow.
- **FR-013**: Every run MUST print what it tracks. A standard package file with no `version.toml` beside it MUST fail unless its folder is under `untracked`. `untracked` hides nothing: a change there still counts for the levels above.

**Growth**

- **FR-014**: A changed file requires growth of every level it belongs to: component, package and repository, as applicable.
- **FR-015**: Growth means strictly greater under the level's scheme. A direct number uses the scheme of the nearest level above with a `driver`, otherwise the root `scheme`, otherwise semver. An invalid number fails.
- **FR-016**: Only pull requests into `main` are checked. Changed files are counted from the fork point; versions are compared with the tip of `main`.
- **FR-017**: `require_bump = false` removes the requirement for the repository version only.
- **FR-018**: Every file counts unless it matches `changes.ignore`.
- **FR-019**: Files are compared with renames off, so deletions and moves count on both sides.

**Configuration changes**

- **FR-020**: Required levels are worked out from both `main` and the pull request; a level on `main` stays bound.
- **FR-021**: Deleting `version.toml` from a folder that keeps other files MUST fail if the pull request also changes those files.
- **FR-022**: `changes.ignore`, `untracked` and `require_bump` are read from `main`.
- **FR-023**: A level absent on `main` is new; any valid number passes. With no root file on `main`, the tool only validates.

**Changelog**

- **FR-024**: When a version grew and its level has `changelog`, the changelog MUST have changed in this pull request, and its top section MUST be the new version or a non-empty `[Unreleased]`, in Keep a Changelog headings.

**Failures**

- **FR-025**: Every failure MUST name the level, the file, and what is expected.

### Key Entities

- **Level**: a repository, package or component, with a folder, a kind and a version source.
- **Version source**: a driver with optional `file` and `key`, or a direct number.
- **Scheme**: the rule ordering numbers.
- **Sample**: a `version.toml` with an expected verdict, shared with `repo-graph-port`.

## Success Criteria *(mandatory)*

- **SC-001**: All samples give their expected verdict, in both readers of the format.
- **SC-002**: All three adversarial cases of issue #1 fail the check.
- **SC-003**: From a failure message alone, a contributor can name which version to raise, in 100% of the failure cases.
- **SC-004**: A single-package project adopts the tool with one file of at most 3 lines.

## Assumptions

- FR-002, FR-004, FR-006 and FR-008 tighten what issue #1 leaves implicit; unknown keys failing is a chosen default.
- Exactly how `changes.ignore` patterns match follows `**` glob conventions; details are for the plan.
- Requiring up-to-date branches or a merge queue on `main` is the project's setting, which the documentation must state.
- A direct number is the level's own: keeping it equal to the built number is the project's job.
- Out of scope: the two open questions of issue #1 (announcing releases, regular expressions).

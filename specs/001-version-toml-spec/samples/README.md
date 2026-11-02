# Shared samples of `version.toml`

Each file is one sample; its parent folder gives the verdict. `invalid/` samples fail for the reason below. Samples are validated as the kind of file they state, and those whose name starts with `nested-` as a file outside the repository root.

| Invalid sample | Reason | Requirement |
|---|---|---|
| `bad-kind.toml` | unknown `kind` | FR-001 |
| `missing-kind.toml` | no `kind` | FR-001 |
| `component-no-version.toml` | component without `version` | FR-003 |
| `component-with-driver.toml` | component with `driver` | FR-003 |
| `package-two-sources.toml` | both `driver` and `version` | FR-004 |
| `package-no-source.toml` | neither `driver` nor `version` | FR-004 |
| `file-without-driver.toml` | `file` without `driver` | FR-004 |
| `unknown-driver.toml` | unknown `driver` | FR-005 |
| `package-root-key.toml` | root-only key in a package | FR-006 |
| `bad-scheme.toml` | unknown `scheme` | FR-007 |
| `unknown-key.toml` | misspelled key | FR-008 |
| `nested-repository.toml` | repository outside the root | FR-002 |
| `bad-version.toml` | not a valid number under semver | FR-015 |

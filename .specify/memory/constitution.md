# version-gate Constitution

## Core Principles

### I. The maintainer decides (NON-NEGOTIABLE)

Commits, pushes, pull requests and merges MUST happen only on an explicit instruction from the
maintainer in the current conversation. A plan or an issue that lists a step is not an instruction.

A request to write an issue, or to add a requirement to one, IS the instruction to do it, in the
same turn. It never authorizes the branch that closes that issue.

This constitution is the maintainer's text: an agent edits it only when asked in the current turn.

### II. English in commits, in git, and in code (NON-NEGOTIABLE)

Commit messages, issues, pull requests, review comments, and every comment or example in the code
MUST be in English. History and code are read by people and agents who do not share the
maintainer's first language.

### III. Go code explains itself to its reader

Every package MUST have a package comment saying what it is for. Every exported identifier MUST
have a doc comment that starts with its name and states purpose, not mechanics. A longer
explanation belongs in the package comment. Code MUST pass `gofmt`, `go vet` and the linters the
repository configures.

### IV. One program, nothing else to install

`version-gate` MUST build into a single static program for Linux, macOS and Windows. It MUST need
only git at run time, and MUST work in any CI; the GitHub Action stays a thin wrapper with no logic
of its own. A published release is never replaced, and every release carries `checksums.txt`.

### V. The tool never runs project code

The tool MUST only read files and git history. It MUST NOT execute anything from the repository it
checks, so it stays fast, free of the project's environment, and safe on a pull request. It MUST
behave the same for a project in any language; language knowledge lives only in drivers.

### VI. The last three steps are the check run, the documentation and the changelog (NON-NEGOTIABLE)

Once the code is what it will be, three steps follow, in this order, each its own planned task:

1. **The check run, taken to zero.** Every remark is fixed; none is accepted as known.
2. **The documentation.** It shows every case the change introduces: each answer, each refusal,
   each failure, named as a case. Every example is executed while it is written, and what the
   reader sees is what it actually printed.
3. **The changelog, as one article.** Plain words about what the work created and the nuances a
   user must know: not a log of commits or files.

### VII. Verification is adversarial

A test proves something only if it can fail. For a new check, break the code the way the test must
catch, confirm that test fails and no other does, then restore. The samples of `version.toml` are
shared with `repo-graph-port` in AOA, so they are part of the contract and change only with the
specification.

## Working rules

- GitHub issues are the units of work: one branch per issue (`feature/issue-<n>-<slug>`) and one
  pull request per branch.
- One phase is one commit.
- Documentation ships with the change it describes.
- The tool's own versions obey the rules it enforces.

## Final step: the check run

Work is finished when the repository's own check run says so. The check run covers formatting,
`go vet`, linters, and the tests on all three systems. A failing check is either fixed in the
change or reported to the maintainer with the exact command and output. If the check run itself
cannot run, that is a defect repaired as part of the change.

## Governance

This constitution overrides every other practice in the repository. Only the maintainer amends it.
Versions follow semver: MAJOR removes or redefines a principle, MINOR adds one or materially expands
it, PATCH refines wording. Every plan and pull request is checked against it.

**Version**: 1.0.0 | **Ratified**: 2026-11-02 | **Last Amended**: 2026-11-02

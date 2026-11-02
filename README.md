# version-gate

A single program that fails CI when a version did not grow with the code, for a project in any language. It reads files and git history, and never runs your code.

A repository, a package and a **component** (a folder that is a solution of its own, too small to be a package) can each have a version. When a pull request changes a file, every version that file belongs to must grow.

## Use it

Add one file to the root of a single-package project:

```toml
kind = "repository"
driver = "python"
```

Then run this in CI on pull requests into `main`, with full history:

```bash
version-gate check --base origin/main
```

To version a folder of its own, add a `version.toml` there:

```toml
kind = "component"
version = "1.2.0"
```

No release is published yet, so build it from source with Go:

```bash
go install github.com/bystrovmaxim/version-gate/cmd/version-gate@latest
```

## Read more

- [Every case](docs/cases.md): each answer and refusal, as the tool printed it.
- [Reference](docs/reference.md): the file format, drivers, rules, setup and known limits.

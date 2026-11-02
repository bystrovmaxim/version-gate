// Command version-gate fails CI when a version did not grow with the code.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/bystrovmaxim/version-gate/internal/change"
	"github.com/bystrovmaxim/version-gate/internal/driver"
	"github.com/bystrovmaxim/version-gate/internal/gate"
	"github.com/bystrovmaxim/version-gate/internal/gitrepo"
	"github.com/bystrovmaxim/version-gate/internal/manifest"
)

// usage is the command's help text.
const usage = "usage: version-gate --version | check [--base ref]"

// version is the program's version, set at build time for a release.
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the command and returns its exit code.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "--version" {
		fmt.Fprintln(stdout, version)
		return 0
	}
	if len(args) > 0 && args[0] == "check" {
		return runCheck(args[1:], stdout, stderr)
	}
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintln(stdout, usage)
		return 0
	}
	fmt.Fprintln(stderr, usage)
	return 2
}

// runCheck prints what the current branch must raise against the base ref.
func runCheck(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	baseRef := flags.String("base", "origin/main", "the ref of main")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	repo := gitrepo.Repo{Dir: "."}
	failures, err := check(repo, *baseRef, "HEAD", stdout)
	if err != nil {
		fmt.Fprintln(stderr, "version-gate:", err)
		return 1
	}
	for _, failure := range failures {
		fmt.Fprintln(stderr, "version-gate:", failure)
	}
	if len(failures) > 0 {
		return 1
	}
	fmt.Fprintln(stdout, "every required version grew")
	return 0
}

// check reads both revisions, prints the tracked levels and judges what must grow.
func check(repo gitrepo.Repo, base, head string, stdout io.Writer) ([]string, error) {
	fork, err := repo.MergeBase(base, head)
	if err != nil {
		return nil, fmt.Errorf("cannot find where this branch left %s; fetch that ref with full history (fetch-depth: 0 in actions/checkout): %w", base, err)
	}
	changed, err := repo.Changed(fork, head)
	if err != nil {
		return nil, err
	}
	baseState, err := readState(repo, base)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", base, err)
	}
	headState, err := readState(repo, head)
	if err != nil {
		return nil, err
	}
	for _, level := range headState.Tree.Levels {
		fmt.Fprintf(stdout, "tracked: %s %s\n", level.Manifest.Kind, level.Dir)
	}
	requirements, err := change.Required(change.Input{
		Base: baseState.Tree, Head: headState.Tree, HeadFiles: headState.Source.Files, Changed: changed,
	})
	if err != nil {
		return nil, err
	}
	return gate.Check(requirements, changed, baseState, headState), nil
}

// readState works out the levels and files at rev.
func readState(repo gitrepo.Repo, rev string) (gate.State, error) {
	files, err := repo.Files(rev)
	if err != nil {
		return gate.State{}, err
	}
	read := func(path string) ([]byte, error) { return repo.Read(rev, path) }
	tree, err := manifest.Discover(files, read)
	if err != nil {
		return gate.State{}, err
	}
	return gate.State{Tree: tree, Source: driver.Source{Files: files, Read: read}}, nil
}

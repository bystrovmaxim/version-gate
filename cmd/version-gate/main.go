// Command version-gate fails CI when a version did not grow with the code.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/bystrovmaxim/version-gate/internal/change"
	"github.com/bystrovmaxim/version-gate/internal/gitrepo"
	"github.com/bystrovmaxim/version-gate/internal/manifest"
)

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
	fmt.Fprintln(stderr, "usage: version-gate --version | check [--base ref]")
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
	requirements, err := required(gitrepo.Repo{Dir: "."}, *baseRef, "HEAD", stdout)
	if err != nil {
		fmt.Fprintln(stderr, "version-gate:", err)
		return 1
	}
	for _, r := range requirements {
		note := ""
		if r.New {
			note = " (new)"
		}
		fmt.Fprintf(stdout, "must grow: %s %s%s\n", r.Kind, r.Dir, note)
	}
	return 0
}

// required reads both revisions, prints the tracked levels and works out what must grow.
func required(repo gitrepo.Repo, base, head string, stdout io.Writer) ([]change.Requirement, error) {
	fork, err := repo.MergeBase(base, head)
	if err != nil {
		return nil, err
	}
	changed, err := repo.Changed(fork, head)
	if err != nil {
		return nil, err
	}
	baseTree, _, err := readTree(repo, base)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", base, err)
	}
	headTree, headFiles, err := readTree(repo, head)
	if err != nil {
		return nil, err
	}
	for _, level := range headTree.Levels {
		fmt.Fprintf(stdout, "tracked: %s %s\n", level.Manifest.Kind, level.Dir)
	}
	return change.Required(change.Input{Base: baseTree, Head: headTree, HeadFiles: headFiles, Changed: changed})
}

// readTree works out the levels at rev.
func readTree(repo gitrepo.Repo, rev string) (*manifest.Tree, []string, error) {
	files, err := repo.Files(rev)
	if err != nil {
		return nil, nil, err
	}
	tree, err := manifest.Discover(files, func(path string) ([]byte, error) { return repo.Read(rev, path) })
	return tree, files, err
}

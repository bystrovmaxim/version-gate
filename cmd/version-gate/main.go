// Command version-gate fails CI when a version did not grow with the code.
package main

import (
	"fmt"
	"io"
	"os"
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
	fmt.Fprintln(stderr, "usage: version-gate --version")
	return 2
}

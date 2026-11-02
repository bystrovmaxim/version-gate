// Package gitrepo reads files and history from a git repository without changing it.
package gitrepo

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Repo is a git repository on disk.
type Repo struct {
	Dir string
}

// git runs one git command and returns its standard output.
func (r Repo) git(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// MergeBase returns the commit where head left base.
func (r Repo) MergeBase(base, head string) (string, error) {
	out, err := r.git("merge-base", base, head)
	return strings.TrimSpace(string(out)), err
}

// Files lists every tracked path at rev.
func (r Repo) Files(rev string) ([]string, error) {
	out, err := r.git("ls-tree", "-r", "-z", "--name-only", rev)
	return splitPaths(out), err
}

// Changed lists the paths that differ between two revisions, with renames off.
func (r Repo) Changed(from, to string) ([]string, error) {
	out, err := r.git("diff", "--name-only", "--no-renames", "-z", from, to)
	return splitPaths(out), err
}

// Read returns a file's content at rev.
func (r Repo) Read(rev, path string) ([]byte, error) {
	return r.git("show", rev+":"+path)
}

// splitPaths splits git's NUL-separated output.
func splitPaths(out []byte) []string {
	var paths []string
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}

// Package change works out which levels a pull request must raise.
package change

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/bystrovmaxim/version-gate/internal/manifest"
)

// Requirement is one level whose version must grow.
type Requirement struct {
	Dir  string
	Kind string
	New  bool
}

// Input is the state of a pull request: Base is the tip of main, Head the pull request.
type Input struct {
	Base      *manifest.Tree
	Head      *manifest.Tree
	HeadFiles []string
	Changed   []string
}

// Required returns the levels that must grow, or an error for a forbidden change.
// With no root file on main every level counts as new, so the numbers are only validated.
func Required(in Input) ([]Requirement, error) {
	root := in.Base.Root()
	if root == nil {
		result := make([]Requirement, 0, len(in.Head.Levels))
		for _, level := range in.Head.Levels {
			result = append(result, Requirement{Dir: level.Dir, Kind: level.Manifest.Kind, New: true})
		}
		return result, nil
	}
	if err := in.Head.CheckStandardFiles(in.HeadFiles, root.Untracked); err != nil {
		return nil, err
	}
	counted, err := countedFiles(in.Changed, root)
	if err != nil {
		return nil, err
	}
	if err := checkRemovals(in, counted); err != nil {
		return nil, err
	}

	required := map[string]Requirement{}
	for _, file := range counted {
		for _, level := range in.Head.Owners(file) {
			_, onBase := in.Base.Level(level.Dir)
			required[level.Dir] = Requirement{Dir: level.Dir, Kind: level.Manifest.Kind, New: !onBase}
		}
	}
	if root.RequireBump != nil && !*root.RequireBump {
		delete(required, ".")
	}

	result := make([]Requirement, 0, len(required))
	for _, r := range required {
		result = append(result, r)
	}
	slices.SortFunc(result, func(a, b Requirement) int { return strings.Compare(a.Dir, b.Dir) })
	return result, nil
}

// countedFiles drops the changed files matched by the root file's ignore patterns.
func countedFiles(changed []string, root *manifest.Manifest) ([]string, error) {
	var patterns []*regexp.Regexp
	if root.Changes != nil {
		for _, pattern := range root.Changes.Ignore {
			re, err := compileGlob(pattern)
			if err != nil {
				return nil, err
			}
			patterns = append(patterns, re)
		}
	}
	var counted []string
	for _, file := range changed {
		if !slices.ContainsFunc(patterns, func(re *regexp.Regexp) bool { return re.MatchString(file) }) {
			counted = append(counted, file)
		}
	}
	return counted, nil
}

// checkRemovals refuses removing a level in the pull request that also changes its files.
func checkRemovals(in Input, counted []string) error {
	for _, level := range in.Base.Levels {
		if _, kept := in.Head.Level(level.Dir); kept || !folderSurvives(level.Dir, in.HeadFiles) {
			continue
		}
		own := path.Join(level.Dir, manifest.FileName)
		for _, file := range counted {
			if file != own && strings.HasPrefix(file, level.Dir+"/") {
				return fmt.Errorf("%s removes level %s and also changes %s; remove a level in a pull request of its own", own, level.Dir, file)
			}
		}
	}
	return nil
}

// folderSurvives reports whether dir still holds a file.
func folderSurvives(dir string, files []string) bool {
	return dir == "." || slices.ContainsFunc(files, func(f string) bool { return strings.HasPrefix(f, dir+"/") })
}

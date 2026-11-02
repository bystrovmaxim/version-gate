// Package gate compares each required level's number with the one on main.
package gate

import (
	"fmt"
	"slices"

	"github.com/bystrovmaxim/version-gate/internal/change"
	"github.com/bystrovmaxim/version-gate/internal/driver"
	"github.com/bystrovmaxim/version-gate/internal/manifest"
	"github.com/bystrovmaxim/version-gate/internal/scheme"
)

// State is one revision: its levels and its files.
type State struct {
	Tree   *manifest.Tree
	Source driver.Source
}

// Check returns one message for every requirement that is not met.
func Check(requirements []change.Requirement, base, head State) []string {
	var failures []string
	for _, r := range requirements {
		if err := checkLevel(r, base, head); err != nil {
			failures = append(failures, fmt.Sprintf("%s %s: %v", r.Kind, r.Dir, err))
		}
	}
	return failures
}

// checkLevel judges one level.
func checkLevel(r change.Requirement, base, head State) error {
	level, _ := head.Tree.Level(r.Dir)
	number, err := driver.Version(level, head.Source)
	if err != nil {
		return err
	}
	name, err := schemeOf(level, head.Tree)
	if err != nil {
		return err
	}
	if err := scheme.Valid(name, number); err != nil {
		return err
	}
	if r.New {
		return nil
	}
	old, _ := base.Tree.Level(r.Dir)
	before, err := driver.Version(old, base.Source)
	if err != nil {
		return fmt.Errorf("on main: %w", err)
	}
	order, err := scheme.Compare(name, number, before)
	if err != nil {
		return fmt.Errorf("on main: %w", err)
	}
	if order <= 0 {
		return fmt.Errorf("version %s did not grow; main has %s, raise it above that", number, before)
	}
	return nil
}

// schemeOf finds a level's scheme: its driver, the nearest driver above, the root's scheme, or semver.
func schemeOf(level manifest.Level, tree *manifest.Tree) (string, error) {
	owners := tree.Owners(level.Dir + "/x")
	slices.Reverse(owners)
	for _, owner := range owners {
		if owner.Manifest.Driver != "" {
			return driver.Scheme(owner.Manifest.Driver)
		}
	}
	if root := tree.Root(); root != nil && root.Scheme != "" {
		return root.Scheme, nil
	}
	return "semver", nil
}

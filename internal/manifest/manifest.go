// Package manifest reads version.toml files and validates each one on its own.
package manifest

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
)

// FileName is the name of the file that declares a level.
const FileName = "version.toml"

// The kinds of level a file can declare.
const (
	KindRepository = "repository"
	KindPackage    = "package"
	KindComponent  = "component"
)

var (
	drivers = []string{"python", "rust", "node", "php", "dart", "maven", "gradle", "dotnet", "ruby"}
	schemes = []string{"semver", "pep440", "maven", "nuget", "rubygems"}
)

// Changes holds the root file's [changes] table.
type Changes struct {
	Ignore []string `toml:"ignore"`
}

// Manifest is the content of one version.toml file.
type Manifest struct {
	Kind        string         `toml:"kind"`
	Version     string         `toml:"version"`
	Driver      string         `toml:"driver"`
	File        string         `toml:"file"`
	Key         string         `toml:"key"`
	Changelog   string         `toml:"changelog"`
	RequireBump *bool          `toml:"require_bump"`
	Untracked   []string       `toml:"untracked"`
	Scheme      string         `toml:"scheme"`
	Changes     *Changes       `toml:"changes"`
	Setup       map[string]any `toml:"setup"`
}

// Parse reads and validates one file; isRoot says it lies in the repository root.
func Parse(data []byte, isRoot bool) (*Manifest, error) {
	var m Manifest
	meta, err := toml.Decode(string(data), &m)
	if err != nil {
		return nil, err
	}
	if unknown := meta.Undecoded(); len(unknown) > 0 && !inSetup(unknown) {
		return nil, fmt.Errorf("unknown key %q", unknown[0].String())
	}
	if err := m.validate(isRoot); err != nil {
		return nil, err
	}
	return &m, nil
}

// inSetup reports whether every undecoded key lies under the reserved [setup] table.
func inSetup(keys []toml.Key) bool {
	for _, key := range keys {
		if key[0] != "setup" {
			return false
		}
	}
	return true
}

// validate applies the rules a single file can be judged by.
func (m *Manifest) validate(isRoot bool) error {
	switch m.Kind {
	case KindRepository:
		if !isRoot {
			return errors.New(`kind "repository" is allowed only in the repository root`)
		}
	case KindPackage:
	case KindComponent:
		if m.Version == "" {
			return errors.New("a component needs version")
		}
		if m.Driver != "" {
			return errors.New("a component cannot have driver")
		}
	case "":
		return errors.New("kind is missing")
	default:
		return fmt.Errorf("unknown kind %q", m.Kind)
	}

	if m.Kind != KindRepository && m.hasRootKeys() {
		return errors.New("require_bump, untracked, scheme, changes and setup belong in the root file")
	}
	if (m.Driver == "") == (m.Version == "") {
		return errors.New("give exactly one of driver and version")
	}
	if m.Driver == "" && (m.File != "" || m.Key != "") {
		return errors.New("file and key need driver")
	}
	if m.Driver != "" && !slices.Contains(drivers, m.Driver) {
		return fmt.Errorf("unknown driver %q; use one of %s", m.Driver, strings.Join(drivers, ", "))
	}
	if m.Scheme != "" && !slices.Contains(schemes, m.Scheme) {
		return fmt.Errorf("unknown scheme %q; use one of %s", m.Scheme, strings.Join(schemes, ", "))
	}
	return nil
}

// hasRootKeys reports whether the file sets a key reserved for the root file.
func (m *Manifest) hasRootKeys() bool {
	return m.RequireBump != nil || m.Untracked != nil || m.Scheme != "" || m.Changes != nil || m.Setup != nil
}

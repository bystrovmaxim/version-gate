// Package driver reads a level's version from the file its language keeps it in.
package driver

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/bystrovmaxim/version-gate/internal/manifest"
)

// Reader returns a file's content by its path from the repository root.
type Reader func(path string) ([]byte, error)

// definition is what one driver knows about its language.
type definition struct {
	file   string
	key    string
	scheme string
	decode func([]byte) (map[string]any, error)
}

var definitions = map[string]definition{
	"python": {"pyproject.toml", "project.version", "pep440", decodeTOML},
	"rust":   {"Cargo.toml", "package.version", "semver", decodeTOML},
	"node":   {"package.json", "version", "semver", decodeJSON},
	"php":    {"composer.json", "version", "semver", decodeJSON},
}

// Version returns the level's number, from its driver or written directly.
func Version(level manifest.Level, read Reader) (string, error) {
	m := level.Manifest
	if m.Driver == "" {
		return m.Version, nil
	}
	def, ok := definitions[m.Driver]
	if !ok {
		return "", fmt.Errorf("driver %q is not supported yet", m.Driver)
	}
	file := path.Join(level.Dir, orDefault(m.File, def.file))
	key := orDefault(m.Key, def.key)
	values, err := load(file, def, read)
	if err != nil {
		return "", err
	}
	value, found := lookup(values, key)
	if !found {
		return "", missing(m, file, key, values)
	}
	if version, ok := value.(string); ok && version != "" {
		return version, nil
	}
	if m.Driver == "rust" && inheritsWorkspace(value) {
		return workspaceVersion(level.Dir, def, read)
	}
	return "", fmt.Errorf("%s: key %s is not a version string", file, key)
}

// Scheme returns the comparison scheme of a driver.
func Scheme(name string) (string, error) {
	if def, ok := definitions[name]; ok {
		return def.scheme, nil
	}
	return "", fmt.Errorf("driver %q is not supported yet", name)
}

// orDefault returns the first non-empty string.
func orDefault(override, fallback string) string {
	if override != "" {
		return override
	}
	return fallback
}

// load reads and decodes one file.
func load(file string, def definition, read Reader) (map[string]any, error) {
	data, err := read(file)
	if err != nil {
		return nil, fmt.Errorf("%s: cannot read: %w", file, err)
	}
	values, err := def.decode(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return values, nil
}

// lookup follows a dotted key through nested tables.
func lookup(values map[string]any, key string) (any, bool) {
	var current any = values
	for _, part := range strings.Split(key, ".") {
		table, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		if current, ok = table[part]; !ok {
			return nil, false
		}
	}
	return current, true
}

// missing explains an absent key, naming the case of a computed Python version.
func missing(m *manifest.Manifest, file, key string, values map[string]any) error {
	if m.Driver == "python" {
		if dynamic, ok := lookup(values, "project.dynamic"); ok {
			if list, ok := dynamic.([]any); ok && slices.Contains(list, any("version")) {
				return fmt.Errorf("%s: the version is computed (dynamic); write version directly in %s", file, manifest.FileName)
			}
		}
	}
	return fmt.Errorf("%s: key %s is missing", file, key)
}

// inheritsWorkspace reports whether a Cargo value is version.workspace = true.
func inheritsWorkspace(value any) bool {
	table, ok := value.(map[string]any)
	return ok && table["workspace"] == true
}

// workspaceVersion finds the version in the nearest Cargo.toml above dir that declares one.
func workspaceVersion(dir string, def definition, read Reader) (string, error) {
	for current := dir; current != "."; {
		current = path.Dir(current)
		values, err := load(path.Join(current, def.file), def, read)
		if err != nil {
			continue
		}
		if value, ok := lookup(values, "workspace.package.version"); ok {
			if version, ok := value.(string); ok {
				return version, nil
			}
		}
	}
	return "", errors.New(dir + ": no Cargo.toml above declares workspace.package.version")
}

// decodeTOML parses a TOML document.
func decodeTOML(data []byte) (map[string]any, error) {
	values := map[string]any{}
	_, err := toml.Decode(string(data), &values)
	return values, err
}

// decodeJSON parses a JSON object.
func decodeJSON(data []byte) (map[string]any, error) {
	values := map[string]any{}
	err := json.Unmarshal(data, &values)
	return values, err
}

// Package driver reads a level's version from the file its language keeps it in.
package driver

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"go.yaml.in/yaml/v3"

	"github.com/bystrovmaxim/version-gate/internal/manifest"
)

// Source gives drivers access to the repository at one revision.
type Source struct {
	Files []string
	Read  func(path string) ([]byte, error)
}

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
	"dart":   {"pubspec.yaml", "version", "semver", decodeYAML},
	"gradle": {"gradle.properties", "version", "maven", decodeProperties},
	"ruby":   {"lib/**/version.rb", "VERSION", "rubygems", decodeRuby},
	"maven":  {"pom.xml", "project/version", "maven", decodeXML},
	"dotnet": {"*.csproj", "Project/PropertyGroup/Version", "nuget", decodeXML},
}

// rubyConstant matches a string constant such as VERSION = "1.2.0".
var rubyConstant = regexp.MustCompile(`(?m)^\s*([A-Z][A-Za-z0-9_]*)\s*=\s*["']([^"']+)["']`)

// Version returns the level's number, from its driver or written directly.
func Version(level manifest.Level, source Source) (string, error) {
	m := level.Manifest
	if m.Driver == "" {
		return m.Version, nil
	}
	def, ok := definitions[m.Driver]
	if !ok {
		return "", fmt.Errorf("driver %q is not supported yet", m.Driver)
	}
	file, err := locate(path.Join(level.Dir, orDefault(m.File, def.file)), source.Files)
	if err != nil {
		return "", err
	}
	key := orDefault(m.Key, def.key)
	values, err := load(file, def, source)
	if err != nil {
		return "", err
	}
	value, found := lookup(values, key)
	if !found && m.Driver == "maven" && m.Key == "" {
		value, found = lookup(values, "project/parent/version")
	}
	if !found {
		return "", missing(m, file, key, values)
	}
	if version, ok := value.(string); ok && version != "" {
		if strings.Contains(version, "${") || strings.Contains(version, "$(") {
			return "", fmt.Errorf("%s: the version %s is computed; write version directly in %s", file, version, manifest.FileName)
		}
		return version, nil
	}
	if m.Driver == "rust" && inheritsWorkspace(value) {
		return workspaceVersion(level.Dir, def, source)
	}
	return "", fmt.Errorf("%s: key %s is not a version string", file, key)
}

// Location names the file a level's number is edited in.
func Location(level manifest.Level, source Source) string {
	def, ok := definitions[level.Manifest.Driver]
	if !ok {
		return path.Join(level.Dir, manifest.FileName)
	}
	pattern := path.Join(level.Dir, orDefault(level.Manifest.File, def.file))
	if file, err := locate(pattern, source.Files); err == nil {
		return file
	}
	return pattern
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

// locate resolves a path with * or ** to the one tracked file that matches it.
func locate(pattern string, files []string) (string, error) {
	if !strings.Contains(pattern, "*") {
		return pattern, nil
	}
	prefix, _, deep := strings.Cut(pattern, "**")
	var matches []string
	for _, file := range files {
		matched := strings.HasPrefix(file, prefix) && path.Base(file) == path.Base(pattern)
		if !deep {
			matched, _ = path.Match(pattern, file)
		}
		if matched {
			matches = append(matches, file)
		}
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("%s matches %d files, want 1; name the file with file", pattern, len(matches))
	}
	return matches[0], nil
}

// load reads and decodes one file.
func load(file string, def definition, source Source) (map[string]any, error) {
	data, err := source.Read(file)
	if err != nil {
		return nil, fmt.Errorf("%s: cannot read: %w", file, err)
	}
	values, err := def.decode(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return values, nil
}

// lookup follows a key with . or / through nested tables.
func lookup(values map[string]any, key string) (any, bool) {
	if value, ok := values[key]; ok {
		return value, true
	}
	separator := "."
	if strings.Contains(key, "/") {
		separator = "/"
	}
	return descend(values, strings.Split(key, separator))
}

// descend follows a key's parts; a repeated element is searched in order.
func descend(current any, parts []string) (any, bool) {
	if len(parts) == 0 {
		return current, true
	}
	switch node := current.(type) {
	case map[string]any:
		if next, ok := node[parts[0]]; ok {
			return descend(next, parts[1:])
		}
	case []any:
		for _, item := range node {
			if value, ok := descend(item, parts); ok {
				return value, true
			}
		}
	}
	return nil, false
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
func workspaceVersion(dir string, def definition, source Source) (string, error) {
	for current := dir; current != "."; {
		current = path.Dir(current)
		values, err := load(path.Join(current, def.file), def, source)
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

// decodeYAML parses a YAML mapping.
func decodeYAML(data []byte) (map[string]any, error) {
	values := map[string]any{}
	err := yaml.Unmarshal(data, &values)
	return values, err
}

// decodeProperties parses a Java properties file; a comment yields a key nobody asks for.
func decodeProperties(data []byte) (map[string]any, error) {
	values := map[string]any{}
	for _, line := range strings.Split(string(data), "\n") {
		if i := strings.IndexAny(line, "=:"); i > 0 {
			values[strings.TrimSpace(line[:i])] = strings.TrimSpace(line[i+1:])
		}
	}
	return values, nil
}

// decodeRuby collects the string constants of a Ruby file without running it.
func decodeRuby(data []byte) (map[string]any, error) {
	values := map[string]any{}
	for _, match := range rubyConstant.FindAllSubmatch(data, -1) {
		values[string(match[1])] = string(match[2])
	}
	return values, nil
}

// decodeXML parses an XML document into nested tables keyed by local element names.
func decodeXML(data []byte) (map[string]any, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	values := map[string]any{}
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return values, nil
		}
		if err != nil {
			return nil, err
		}
		if start, ok := token.(xml.StartElement); ok {
			value, err := decodeElement(decoder)
			if err != nil {
				return nil, err
			}
			addElement(values, start.Name.Local, value)
		}
	}
}

// decodeElement reads one element's content: a table if it has children, else its text.
func decodeElement(decoder *xml.Decoder) (any, error) {
	children := map[string]any{}
	var text strings.Builder
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		switch t := token.(type) {
		case xml.StartElement:
			value, err := decodeElement(decoder)
			if err != nil {
				return nil, err
			}
			addElement(children, t.Name.Local, value)
		case xml.CharData:
			text.Write(t)
		case xml.EndElement:
			if len(children) > 0 {
				return children, nil
			}
			return strings.TrimSpace(text.String()), nil
		}
	}
}

// addElement stores a child, keeping repeated names as a list in document order.
func addElement(table map[string]any, name string, value any) {
	switch existing := table[name].(type) {
	case nil:
		table[name] = value
	case []any:
		table[name] = append(existing, value)
	default:
		table[name] = []any{existing, value}
	}
}

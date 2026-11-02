package manifest

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

// Level is a tracked folder; Dir is slash-separated and "." for the root.
type Level struct {
	Dir      string
	Manifest *Manifest
}

// Name returns the level's name, the last element of its folder.
func (l Level) Name() string {
	return path.Base(l.Dir)
}

// Tree holds a repository's levels, sorted by folder.
type Tree struct {
	Levels []Level
}

// standardFiles are the package files that need a version.toml beside them.
var standardFiles = []string{
	"pyproject.toml", "Cargo.toml", "package.json", "composer.json",
	"pubspec.yaml", "pom.xml", "gradle.properties",
}

// Discover works out the levels from tracked slash-separated paths; read returns a file's content.
func Discover(files []string, read func(string) ([]byte, error)) (*Tree, error) {
	tree := &Tree{}
	for _, file := range files {
		if path.Base(file) != FileName {
			continue
		}
		data, err := read(file)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		dir := path.Dir(file)
		m, err := Parse(data, dir == ".")
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		tree.Levels = append(tree.Levels, Level{Dir: dir, Manifest: m})
	}
	slices.SortFunc(tree.Levels, func(a, b Level) int { return strings.Compare(a.Dir, b.Dir) })

	if err := tree.checkNesting(); err != nil {
		return nil, err
	}
	return tree, nil
}

// checkNesting refuses a component inside another component.
func (t *Tree) checkNesting() error {
	for _, level := range t.Levels {
		if level.Manifest.Kind != KindComponent {
			continue
		}
		for _, outer := range t.Owners(path.Dir(level.Dir) + "/x") {
			if outer.Dir != level.Dir && outer.Manifest.Kind == KindComponent {
				return fmt.Errorf("component %s lies inside component %s", level.Dir, outer.Dir)
			}
		}
	}
	return nil
}

// CheckStandardFiles refuses a package file with no version.toml beside it.
func (t *Tree) CheckStandardFiles(files, untracked []string) error {
	for _, file := range files {
		dir := path.Dir(file)
		if !isStandardFile(path.Base(file)) || t.has(dir) || slices.Contains(untracked, dir) {
			continue
		}
		return fmt.Errorf("%s has no %s beside it; add one, or list %s under untracked", file, FileName, dir)
	}
	return nil
}

// Root returns the repository's root file, or nil when it has none.
func (t *Tree) Root() *Manifest {
	if level, ok := t.Level("."); ok {
		return level.Manifest
	}
	return nil
}

// Level returns the level declared in dir.
func (t *Tree) Level(dir string) (Level, bool) {
	i := slices.IndexFunc(t.Levels, func(l Level) bool { return l.Dir == dir })
	if i < 0 {
		return Level{}, false
	}
	return t.Levels[i], true
}

// has reports whether dir is a level.
func (t *Tree) has(dir string) bool {
	_, ok := t.Level(dir)
	return ok
}

// Owners returns every level a file belongs to, outermost first.
func (t *Tree) Owners(file string) []Level {
	var owners []Level
	for _, level := range t.Levels {
		if level.Dir == "." || strings.HasPrefix(file, level.Dir+"/") {
			owners = append(owners, level)
		}
	}
	return owners
}

// isStandardFile reports whether name is a package file of a known language.
func isStandardFile(name string) bool {
	return slices.Contains(standardFiles, name) ||
		strings.HasSuffix(name, ".csproj") || strings.HasSuffix(name, ".gemspec")
}

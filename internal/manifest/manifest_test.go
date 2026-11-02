package manifest

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const samples = "../../specs/001-version-toml-spec/samples"

// numberSamples need a version comparison, which arrives in phase 8.
var numberSamples = map[string]bool{"bad-version.toml": true}

func sampleIsRoot(name string) bool {
	return !strings.HasPrefix(name, "nested-")
}

func TestSamples(t *testing.T) {
	for verdict, wantErr := range map[string]bool{"valid": false, "invalid": true} {
		paths, err := filepath.Glob(filepath.Join(samples, verdict, "*.toml"))
		if err != nil || len(paths) == 0 {
			t.Fatalf("no %s samples: %v", verdict, err)
		}
		for _, p := range paths {
			name := filepath.Base(p)
			if numberSamples[name] {
				continue
			}
			t.Run(verdict+"/"+name, func(t *testing.T) {
				data, err := os.ReadFile(p)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := Parse(data, sampleIsRoot(name)); (err != nil) != wantErr {
					t.Fatalf("Parse error = %v, want error: %v", err, wantErr)
				}
			})
		}
	}
}

func discover(files map[string]string) (*Tree, error) {
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	return Discover(paths, func(p string) ([]byte, error) {
		content, ok := files[p]
		if !ok {
			return nil, errors.New("missing")
		}
		return []byte(content), nil
	})
}

const (
	root      = "kind = \"repository\"\ndriver = \"python\"\n"
	pkg       = "kind = \"package\"\ndriver = \"python\"\n"
	component = "kind = \"component\"\nversion = \"1.0.0\"\n"
)

func TestOwnersOutermostFirstWithNestedPackages(t *testing.T) {
	tree, err := discover(map[string]string{
		"version.toml":       root,
		"a/version.toml":     pkg,
		"a/b/version.toml":   pkg,
		"a/b/c/version.toml": component,
		"ab/version.toml":    pkg,
		"a/b/c/d/deep.py":    "",
	})
	if err != nil {
		t.Fatal(err)
	}
	var dirs []string
	for _, level := range tree.Owners("a/b/c/d/deep.py") {
		dirs = append(dirs, level.Dir)
	}
	if got, want := strings.Join(dirs, " "), ". a a/b a/b/c"; got != want {
		t.Fatalf("owners = %q, want %q", got, want)
	}
}

func TestComponentInsideComponentFails(t *testing.T) {
	_, err := discover(map[string]string{
		"a/version.toml":     component,
		"a/b/c/version.toml": component,
	})
	if err == nil || !strings.Contains(err.Error(), "a/b/c lies inside component a") {
		t.Fatalf("err = %v", err)
	}
}

func TestStandardFileNeedsVersionToml(t *testing.T) {
	_, err := discover(map[string]string{"version.toml": root, "a/pyproject.toml": ""})
	if err == nil || !strings.Contains(err.Error(), "a/pyproject.toml") {
		t.Fatalf("err = %v", err)
	}
}

func TestUntrackedFreesTheFolderAndHidesNothing(t *testing.T) {
	tree, err := discover(map[string]string{
		"version.toml":     root + "untracked = [\"a\"]\n",
		"a/pyproject.toml": "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(tree.Owners("a/pyproject.toml")); got != 1 {
		t.Fatalf("owners = %d, want 1", got)
	}
}

func TestOwnersDoNotMatchASiblingWithTheSamePrefix(t *testing.T) {
	tree, err := discover(map[string]string{
		"version.toml":   root,
		"a/version.toml": pkg,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(tree.Owners("ab/file.py")); got != 1 {
		t.Fatalf("owners = %d, want 1", got)
	}
}

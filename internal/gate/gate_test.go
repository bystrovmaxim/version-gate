package gate

import (
	"os"
	"strings"
	"testing"

	"github.com/bystrovmaxim/version-gate/internal/change"
	"github.com/bystrovmaxim/version-gate/internal/driver"
	"github.com/bystrovmaxim/version-gate/internal/manifest"
)

func parse(t *testing.T, data []byte, isRoot bool) *manifest.Manifest {
	t.Helper()
	m, err := manifest.Parse(data, isRoot)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func tree(levels ...manifest.Level) *manifest.Tree {
	return &manifest.Tree{Levels: levels}
}

func TestSchemeOrder(t *testing.T) {
	root := manifest.Level{Dir: ".", Manifest: parse(t, []byte("kind = \"repository\"\nversion = \"1.0\"\nscheme = \"pep440\"\n"), true)}
	driven := manifest.Level{Dir: "p", Manifest: parse(t, []byte("kind = \"package\"\ndriver = \"rust\"\n"), false)}
	inside := manifest.Level{Dir: "p/c", Manifest: parse(t, []byte("kind = \"component\"\nversion = \"1.0.0\"\n"), false)}
	outside := manifest.Level{Dir: "q", Manifest: parse(t, []byte("kind = \"component\"\nversion = \"1.0\"\n"), false)}
	bare := manifest.Level{Dir: "c", Manifest: parse(t, []byte("kind = \"component\"\nversion = \"1.0.0\"\n"), false)}

	pythonRoot := manifest.Level{Dir: ".", Manifest: parse(t, []byte("kind = \"repository\"\ndriver = \"python\"\n"), true)}

	for _, c := range []struct {
		name  string
		level manifest.Level
		tree  *manifest.Tree
		want  string
	}{
		{"driver above wins over root", inside, tree(root, driven, inside), "semver"},
		{"nearest driver wins", inside, tree(pythonRoot, driven, inside), "semver"},
		{"root scheme", outside, tree(root, outside), "pep440"},
		{"default", bare, tree(bare), "semver"},
	} {
		got, err := schemeOf(c.level, c.tree)
		if err != nil || got != c.want {
			t.Errorf("%s: got %q, %v", c.name, got, err)
		}
	}
}

func TestBadVersionSampleFails(t *testing.T) {
	data, err := os.ReadFile("../../specs/001-version-toml-spec/samples/invalid/bad-version.toml")
	if err != nil {
		t.Fatal(err)
	}
	level := manifest.Level{Dir: "c", Manifest: parse(t, data, false)}
	failures := Check([]change.Requirement{{Dir: "c", Kind: "component", New: true}}, nil, State{Tree: tree()}, State{Tree: tree(level)})
	if len(failures) != 1 {
		t.Fatalf("failures = %v", failures)
	}
}

func changelogLevel() manifest.Level {
	return manifest.Level{Dir: "p", Manifest: &manifest.Manifest{Kind: "package", Version: "1.1.0", Changelog: "CHANGELOG.md"}}
}

func TestChangelog(t *testing.T) {
	const file = "p/CHANGELOG.md"
	cases := []struct {
		name    string
		text    string
		changed []string
		want    string
	}{
		{"new version on top", "# Changelog\n\n## [1.1.0] - 2026-01-01\n- x\n\n## [1.0.0]\n", []string{file}, ""},
		{"filled unreleased", "## [Unreleased]\n### Added\n- x\n\n## [1.0.0]\n", []string{file}, ""},
		{"no brackets", "## 1.1.0\n", []string{file}, ""},
		{"not changed", "## [Unreleased]\n- x\n", nil, "did not change"},
		{"empty unreleased", "## [Unreleased]\n\n### Added\n\n## [1.0.0]\n- old\n", []string{file}, "is empty"},
		{"old version on top", "## [1.0.0]\n- x\n", []string{file}, "top section is 1.0.0"},
		{"no sections", "# Changelog\n", []string{file}, "no section found"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			source := driver.Source{Read: func(string) ([]byte, error) { return []byte(c.text), nil }}
			err := checkChangelog(changelogLevel(), "1.1.0", c.changed, source)
			if c.want == "" && err != nil {
				t.Fatal(err)
			}
			if c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)) {
				t.Fatalf("err = %v, want %q", err, c.want)
			}
		})
	}
}

func TestLevelWithoutChangelogIsNotChecked(t *testing.T) {
	level := changelogLevel()
	level.Manifest.Changelog = ""
	if err := checkChangelog(level, "1.1.0", nil, driver.Source{}); err != nil {
		t.Fatal(err)
	}
}

package driver

import (
	"errors"
	"strings"
	"testing"

	"github.com/bystrovmaxim/version-gate/internal/manifest"
)

func level(dir, drv, file, key string) manifest.Level {
	return manifest.Level{Dir: dir, Manifest: &manifest.Manifest{Kind: "package", Driver: drv, File: file, Key: key}}
}

func reader(files map[string]string) Reader {
	return func(p string) ([]byte, error) {
		if content, ok := files[p]; ok {
			return []byte(content), nil
		}
		return nil, errors.New("no such file")
	}
}

func TestVersion(t *testing.T) {
	cases := []struct {
		name  string
		level manifest.Level
		files map[string]string
		want  string
	}{
		{"python", level("p", "python", "", ""), map[string]string{"p/pyproject.toml": "[project]\nversion = \"1.2.0\"\n"}, "1.2.0"},
		{"rust", level("p", "rust", "", ""), map[string]string{"p/Cargo.toml": "[package]\nversion = \"1.2.1\"\n"}, "1.2.1"},
		{"node", level("p", "node", "", ""), map[string]string{"p/package.json": `{"version": "1.2.2"}`}, "1.2.2"},
		{"php", level(".", "php", "", ""), map[string]string{"composer.json": `{"version": "1.2.3"}`}, "1.2.3"},
		{"file and key", level("p", "node", "app/meta.json", "build.version"), map[string]string{"p/app/meta.json": `{"build": {"version": "2.0.0"}}`}, "2.0.0"},
		{"rust workspace", level("crates/a", "rust", "", ""), map[string]string{
			"crates/a/Cargo.toml": "[package]\nversion.workspace = true\n",
			"Cargo.toml":          "[workspace.package]\nversion = \"3.1.0\"\n",
		}, "3.1.0"},
		{"direct", manifest.Level{Dir: "p", Manifest: &manifest.Manifest{Kind: "component", Version: "4.0.0"}}, nil, "4.0.0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Version(c.level, reader(c.files))
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Fatalf("version = %q, want %q", got, c.want)
			}
		})
	}
}

func TestVersionFailures(t *testing.T) {
	cases := []struct {
		name  string
		level manifest.Level
		files map[string]string
		want  string
	}{
		{"missing file", level("p", "node", "", ""), nil, "p/package.json: cannot read"},
		{"missing key", level("p", "node", "", ""), map[string]string{"p/package.json": `{}`}, "key version is missing"},
		{"not a string", level("p", "node", "", ""), map[string]string{"p/package.json": `{"version": 1}`}, "not a version string"},
		{"empty string", level("p", "node", "", ""), map[string]string{"p/package.json": `{"version": ""}`}, "not a version string"},
		{"broken file", level("p", "node", "", ""), map[string]string{"p/package.json": `{`}, "p/package.json"},
		{"python dynamic", level("p", "python", "", ""), map[string]string{"p/pyproject.toml": "[project]\ndynamic = [\"version\"]\n"}, "computed"},
		{"rust no workspace", level("a", "rust", "", ""), map[string]string{"a/Cargo.toml": "[package]\nversion.workspace = true\n"}, "workspace.package.version"},
		{"later driver", level("p", "maven", "", ""), nil, "not supported yet"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Version(c.level, reader(c.files))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want %q", err, c.want)
			}
		})
	}
}

func TestScheme(t *testing.T) {
	for name, want := range map[string]string{"python": "pep440", "rust": "semver", "node": "semver", "php": "semver"} {
		if got, err := Scheme(name); err != nil || got != want {
			t.Errorf("Scheme(%s) = %q, %v", name, got, err)
		}
	}
}

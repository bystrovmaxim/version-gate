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

func source(files map[string]string) Source {
	s := Source{Read: func(p string) ([]byte, error) {
		if content, ok := files[p]; ok {
			return []byte(content), nil
		}
		return nil, errors.New("no such file")
	}}
	for p := range files {
		s.Files = append(s.Files, p)
	}
	return s
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
		{"dart", level("p", "dart", "", ""), map[string]string{"p/pubspec.yaml": "name: app\nversion: 1.2.4+5\n"}, "1.2.4+5"},
		{"gradle", level("p", "gradle", "", ""), map[string]string{"p/gradle.properties": "# note\ngroup=x\nversion = 1.2.5\n"}, "1.2.5"},
		{"gradle dotted key", level("p", "gradle", "", "app.version"), map[string]string{"p/gradle.properties": "app.version=1.2.6\n"}, "1.2.6"},
		{"ruby", level("p", "ruby", "", ""), map[string]string{"p/lib/my/gem/version.rb": "module My\n  VERSION = \"1.2.7\".freeze\nend\n"}, "1.2.7"},
		{"ruby at the root", level(".", "ruby", "", ""), map[string]string{"lib/gem/version.rb": "VERSION = '1.2.8'\n"}, "1.2.8"},
		{"gradle comment", level("p", "gradle", "", ""), map[string]string{"p/gradle.properties": "#version=9.9.9\n! version=9.9.9\nversion=1.2.9\n"}, "1.2.9"},
		{"ruby in another folder", level("p", "ruby", "", ""), map[string]string{"p/lib/version.rb": "VERSION = '1.3.0'\n", "q/lib/version.rb": "VERSION = '9.9.9'\n"}, "1.3.0"},
		{"maven", level("p", "maven", "", ""), map[string]string{"p/pom.xml": `<project xmlns="http://maven.apache.org/POM/4.0.0"><parent><version>9.0</version></parent><version>1.4.0</version></project>`}, "1.4.0"},
		{"maven parent", level("p", "maven", "", ""), map[string]string{"p/pom.xml": `<project><parent><version>1.4.1</version></parent></project>`}, "1.4.1"},
		{"maven comment", level("p", "maven", "", ""), map[string]string{"p/pom.xml": `<project><!-- <version>9</version> --><version> 1.4.2 </version></project>`}, "1.4.2"},
		{"dotnet", level("p", "dotnet", "", ""), map[string]string{"p/App.csproj": "\ufeff<Project><PropertyGroup><Nullable>enable</Nullable></PropertyGroup><PropertyGroup><Version>1.5.0</Version></PropertyGroup></Project>", "q/Other.csproj": "", "p/sub/Deep.csproj": ""}, "1.5.0"},
		{"dotnet first group", level("p", "dotnet", "", ""), map[string]string{"p/A.csproj": `<Project><PropertyGroup><Version>1.5.2</Version></PropertyGroup><PropertyGroup><Nullable>enable</Nullable></PropertyGroup></Project>`}, "1.5.2"},
		{"dotnet at the root", level(".", "dotnet", "", ""), map[string]string{"App.csproj": `<Project><PropertyGroup><Version>1.5.1</Version></PropertyGroup></Project>`}, "1.5.1"},
		{"file and key", level("p", "node", "app/meta.json", "build.version"), map[string]string{"p/app/meta.json": `{"build": {"version": "2.0.0"}}`}, "2.0.0"},
		{"rust workspace", level("crates/a", "rust", "", ""), map[string]string{
			"crates/a/Cargo.toml": "[package]\nversion.workspace = true\n",
			"Cargo.toml":          "[workspace.package]\nversion = \"3.1.0\"\n",
		}, "3.1.0"},
		{"direct", manifest.Level{Dir: "p", Manifest: &manifest.Manifest{Kind: "component", Version: "4.0.0"}}, nil, "4.0.0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Version(c.level, source(c.files))
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
		{"yaml number", level("p", "dart", "", ""), map[string]string{"p/pubspec.yaml": "version: 1.2\n"}, "not a version string"},
		{"ruby no file", level("p", "ruby", "", ""), map[string]string{"p/lib/other.rb": ""}, "matches 0 files"},
		{"ruby two files", level("p", "ruby", "", ""), map[string]string{"p/lib/a/version.rb": "", "p/lib/b/version.rb": ""}, "matches 2 files"},
		{"ruby computed", level("p", "ruby", "", ""), map[string]string{"p/lib/version.rb": "VERSION = compute()\n"}, "key VERSION is missing"},
		{"ruby commented out", level("p", "ruby", "", ""), map[string]string{"p/lib/version.rb": "# VERSION = \"1.0\"\n"}, "key VERSION is missing"},
		{"maven property", level("p", "maven", "", ""), map[string]string{"p/pom.xml": `<project><version>${revision}</version></project>`}, "computed"},
		{"maven nothing", level("p", "maven", "", ""), map[string]string{"p/pom.xml": `<project></project>`}, "key project/version is missing"},
		{"maven broken", level("p", "maven", "", ""), map[string]string{"p/pom.xml": `<project>`}, "p/pom.xml"},
		{"dotnet property", level("p", "dotnet", "", ""), map[string]string{"p/A.csproj": `<Project><PropertyGroup><Version>$(Base)</Version></PropertyGroup></Project>`}, "computed"},
		{"dotnet two projects", level("p", "dotnet", "", ""), map[string]string{"p/A.csproj": "", "p/B.csproj": ""}, "matches 2 files"},
		{"dotnet no version", level("p", "dotnet", "", ""), map[string]string{"p/A.csproj": `<Project><PropertyGroup/></Project>`}, "is missing"},
		{"unknown driver", level("p", "cobol", "", ""), nil, "not supported yet"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Version(c.level, source(c.files))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want %q", err, c.want)
			}
		})
	}
}

func TestScheme(t *testing.T) {
	for name, want := range map[string]string{"python": "pep440", "rust": "semver", "node": "semver", "php": "semver", "dart": "semver", "gradle": "maven", "ruby": "rubygems", "maven": "maven", "dotnet": "nuget"} {
		if got, err := Scheme(name); err != nil || got != want {
			t.Errorf("Scheme(%s) = %q, %v", name, got, err)
		}
	}
}

func TestLocation(t *testing.T) {
	cases := []struct {
		level manifest.Level
		want  string
	}{
		{level("p", "node", "", ""), "p/package.json"},
		{level("p", "ruby", "", ""), "p/lib/x/version.rb"},
		{manifest.Level{Dir: "c", Manifest: &manifest.Manifest{Version: "1.0.0"}}, "c/version.toml"},
	}
	for _, c := range cases {
		s := source(map[string]string{"p/lib/x/version.rb": ""})
		if got := Location(c.level, s); got != c.want {
			t.Errorf("Location = %q, want %q", got, c.want)
		}
	}
}

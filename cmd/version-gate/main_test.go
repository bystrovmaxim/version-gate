package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPrintsVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got := strings.TrimSpace(stdout.String()); got != version {
		t.Fatalf("stdout = %q, want %q", got, version)
	}
}

func TestRunRejectsUnknownArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--nope"}, &stdout, &stderr); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("stderr = %q, want usage", stderr.String())
	}
}

// repository builds a real repository with main and a branch named pr, and enters it.
func repository(t *testing.T, files map[string]string) func(args ...string) {
	dir := t.TempDir()
	t.Chdir(dir)
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main")
	write(t, files)
	git("add", "-A")
	git("commit", "-q", "-m", "main")
	git("switch", "-q", "-c", "pr")
	return git
}

func write(t *testing.T, files map[string]string) {
	t.Helper()
	for name, content := range files {
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// verdict commits the working tree and runs the command against main.
func verdict(t *testing.T, git func(...string)) (int, string) {
	t.Helper()
	git("add", "-A")
	git("commit", "-q", "--allow-empty", "-m", "pr")
	var stdout, stderr bytes.Buffer
	code := run([]string{"check", "--base", "main"}, &stdout, &stderr)
	return code, stderr.String()
}

const (
	rootFile = "kind = \"repository\"\nversion = \"1.0.0\"\n"
	pkgFile  = "kind = \"package\"\ndriver = \"node\"\n"
)

func tomlVersion(number string) string {
	return "kind = \"component\"\nversion = \"" + number + "\"\n"
}

func project() map[string]string {
	return map[string]string{
		"version.toml":     rootFile,
		"p/version.toml":   pkgFile,
		"p/package.json":   `{"version": "1.0.0"}`,
		"p/a/version.toml": tomlVersion("1.0.0"),
		"p/a/code.js":      "",
		"p/b/version.toml": tomlVersion("1.0.0"),
		"p/b/other.js":     "",
	}
}

func raised() map[string]string {
	return map[string]string{
		"version.toml":     "kind = \"repository\"\nversion = \"1.0.1\"\n",
		"p/package.json":   `{"version": "1.0.1"}`,
		"p/a/version.toml": tomlVersion("1.0.1"),
	}
}

func TestEveryVersionRaisedPasses(t *testing.T) {
	git := repository(t, project())
	write(t, raised())
	write(t, map[string]string{"p/a/code.js": "x"})
	if code, stderr := verdict(t, git); code != 0 {
		t.Fatalf("code = %d: %s", code, stderr)
	}
}

func TestVersionNotRaisedFailsAndNamesIt(t *testing.T) {
	git := repository(t, project())
	write(t, raised())
	write(t, map[string]string{"p/a/code.js": "x", "p/a/version.toml": tomlVersion("1.0.0")})
	code, stderr := verdict(t, git)
	if code != 1 || !strings.Contains(stderr, "component p/a") || !strings.Contains(stderr, "did not grow") {
		t.Fatalf("code = %d: %s", code, stderr)
	}
	if strings.Contains(stderr, "package p") {
		t.Fatalf("named a level that grew: %s", stderr)
	}
}

func TestVersionGoingDownFails(t *testing.T) {
	git := repository(t, project())
	write(t, raised())
	write(t, map[string]string{"p/a/code.js": "x", "p/a/version.toml": tomlVersion("0.9.0")})
	if code, _ := verdict(t, git); code != 1 {
		t.Fatalf("code = %d", code)
	}
}

func TestSecondPullRequestMustGoAboveTheFirst(t *testing.T) {
	git := repository(t, project())
	write(t, raised())
	write(t, map[string]string{"p/a/code.js": "x"})
	git("add", "-A")
	git("commit", "-q", "-m", "pr")
	git("switch", "-q", "main")
	git("switch", "-q", "-c", "first")
	write(t, raised())
	write(t, map[string]string{"p/b/other.js": "y"})
	git("add", "-A")
	git("commit", "-q", "-m", "first")
	git("switch", "-q", "main")
	git("merge", "-q", "first")
	git("switch", "-q", "pr")
	code, stderr := verdict(t, git)
	if code != 1 || !strings.Contains(stderr, "main has 1.0.1") {
		t.Fatalf("code = %d: %s", code, stderr)
	}
}

func TestHidingAChangeBehindANewIgnoreFails(t *testing.T) {
	git := repository(t, project())
	write(t, map[string]string{"version.toml": rootFile + "[changes]\nignore = [\"**/*.js\"]\n", "p/a/code.js": "x"})
	code, stderr := verdict(t, git)
	if code != 1 || !strings.Contains(stderr, "component p/a") {
		t.Fatalf("code = %d: %s", code, stderr)
	}
}

func TestMoveWithOneVersionRaisedFails(t *testing.T) {
	git := repository(t, project())
	write(t, raised())
	git("mv", "p/a/code.js", "p/b/code.js")
	code, stderr := verdict(t, git)
	if code != 1 || !strings.Contains(stderr, "component p/b") {
		t.Fatalf("code = %d: %s", code, stderr)
	}
}

func TestDeletingVersionTomlWhileChangingCodeFails(t *testing.T) {
	git := repository(t, project())
	git("rm", "-q", "p/a/version.toml")
	write(t, map[string]string{"p/a/code.js": "x"})
	code, stderr := verdict(t, git)
	if code != 1 || !strings.Contains(stderr, "removes level p/a") {
		t.Fatalf("code = %d: %s", code, stderr)
	}
}

func TestAdoptionPassesAndStillValidatesNumbers(t *testing.T) {
	git := repository(t, map[string]string{"code.js": ""})
	write(t, map[string]string{"version.toml": rootFile})
	if code, stderr := verdict(t, git); code != 0 {
		t.Fatalf("code = %d: %s", code, stderr)
	}
	write(t, map[string]string{"version.toml": "kind = \"repository\"\nversion = \"one\"\n"})
	if code, _ := verdict(t, git); code != 1 {
		t.Fatalf("code = %d", code)
	}
}

func TestHelpGoesToStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--help"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "usage:") {
		t.Fatalf("code = %d, stdout = %q", code, stdout.String())
	}
}

func TestMissingBaseExplainsTheFix(t *testing.T) {
	repository(t, project())
	var stdout, stderr bytes.Buffer
	code := run([]string{"check", "--base", "origin/main"}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "fetch-depth: 0") {
		t.Fatalf("code = %d: %s", code, stderr.String())
	}
}

func TestFailureNamesTheFileToEdit(t *testing.T) {
	git := repository(t, project())
	write(t, map[string]string{"p/a/code.js": "x"})
	_, stderr := verdict(t, git)
	for _, want := range []string{"(p/package.json)", "(p/a/version.toml)", "(version.toml)"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr lacks %s: %s", want, stderr)
		}
	}
}

func TestChangelogFailureReachesTheCommand(t *testing.T) {
	files := project()
	files["p/a/version.toml"] += "changelog = \"CHANGELOG.md\"\n"
	git := repository(t, files)
	write(t, raised())
	write(t, map[string]string{"p/a/version.toml": tomlVersion("1.0.1") + "changelog = \"CHANGELOG.md\"\n", "p/a/code.js": "x"})
	code, stderr := verdict(t, git)
	if code != 1 || !strings.Contains(stderr, "p/a/CHANGELOG.md did not change") {
		t.Fatalf("code = %d: %s", code, stderr)
	}
}

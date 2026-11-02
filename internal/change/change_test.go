package change

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/bystrovmaxim/version-gate/internal/gitrepo"
	"github.com/bystrovmaxim/version-gate/internal/manifest"
)

const (
	root      = "kind = \"repository\"\ndriver = \"python\"\n"
	pkg       = "kind = \"package\"\ndriver = \"python\"\n"
	component = "kind = \"component\"\nversion = \"1.0.0\"\n"
)

// sandbox is a real git repository with main and a branch named pr.
type sandbox struct {
	t   *testing.T
	dir string
}

func (s sandbox) git(args ...string) {
	s.t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = s.dir
	if out, err := cmd.CombinedOutput(); err != nil {
		s.t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

func (s sandbox) write(files map[string]string) {
	s.t.Helper()
	for name, content := range files {
		p := filepath.Join(s.dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			s.t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			s.t.Fatal(err)
		}
	}
}

func (s sandbox) commit() {
	s.git("add", "-A")
	s.git("commit", "-q", "--allow-empty", "-m", "c")
}

func newSandbox(t *testing.T, files map[string]string) sandbox {
	s := sandbox{t, t.TempDir()}
	s.git("init", "-q", "-b", "main")
	s.write(files)
	s.commit()
	s.git("switch", "-q", "-c", "pr")
	return s
}

// required commits the working tree and runs the whole path from git to the requirements.
func (s sandbox) required() (string, error) {
	s.t.Helper()
	s.commit()
	repo := gitrepo.Repo{Dir: s.dir}
	tree := func(rev string) (*manifest.Tree, []string) {
		files, err := repo.Files(rev)
		if err != nil {
			s.t.Fatal(err)
		}
		tree, err := manifest.Discover(files, func(p string) ([]byte, error) { return repo.Read(rev, p) })
		if err != nil {
			s.t.Fatal(err)
		}
		return tree, files
	}
	fork, err := repo.MergeBase("main", "pr")
	if err != nil {
		s.t.Fatal(err)
	}
	changed, err := repo.Changed(fork, "pr")
	if err != nil {
		s.t.Fatal(err)
	}
	base, _ := tree("main")
	head, headFiles := tree("pr")
	reqs, err := Required(Input{Base: base, Head: head, HeadFiles: headFiles, Changed: changed})
	var dirs []string
	for _, r := range reqs {
		dirs = append(dirs, r.Dir)
	}
	return strings.Join(dirs, " "), err
}

func want(t *testing.T, s sandbox, expected string) {
	t.Helper()
	got, err := s.required()
	if err != nil {
		t.Fatal(err)
	}
	if got != expected {
		t.Fatalf("required = %q, want %q", got, expected)
	}
}

var base = map[string]string{
	"version.toml":     root,
	"top.txt":          "",
	"p/version.toml":   pkg,
	"p/code.py":        "",
	"p/a/version.toml": component,
	"p/a/one.py":       "",
	"p/a/two.py":       "",
	"p/b/version.toml": component,
	"p/b/three.py":     "",
}

func TestFileInsideAComponentRequiresEveryLevel(t *testing.T) {
	s := newSandbox(t, base)
	s.write(map[string]string{"p/a/one.py": "x"})
	want(t, s, ". p p/a")
}

func TestFileInsideAPackageRequiresPackageAndRepository(t *testing.T) {
	s := newSandbox(t, base)
	s.write(map[string]string{"p/code.py": "x"})
	want(t, s, ". p")
}

func TestFileOutsidePackagesRequiresRepository(t *testing.T) {
	s := newSandbox(t, base)
	s.write(map[string]string{"top.txt": "x"})
	want(t, s, ".")
}

func TestRequireBumpFalseFreesOnlyTheRepository(t *testing.T) {
	files := map[string]string{"version.toml": root + "require_bump = false\n", "p/version.toml": pkg, "p/code.py": ""}
	s := newSandbox(t, files)
	s.write(map[string]string{"p/code.py": "x"})
	want(t, s, "p")
}

func TestPullRequestCannotSwitchOffRequireBumpForItself(t *testing.T) {
	s := newSandbox(t, base)
	s.write(map[string]string{"version.toml": root + "require_bump = false\n", "top.txt": "x"})
	want(t, s, ".")
}

func TestOnlyIgnoredFilesRequireNothing(t *testing.T) {
	files := map[string]string{"version.toml": root + "[changes]\nignore = [\"**/*.md\"]\n", "p/version.toml": pkg, "p/doc.md": ""}
	s := newSandbox(t, files)
	s.write(map[string]string{"p/doc.md": "x"})
	want(t, s, "")
}

func TestNewIgnorePatternDoesNotCoverItsOwnPullRequest(t *testing.T) {
	s := newSandbox(t, base)
	s.write(map[string]string{"version.toml": root + "[changes]\nignore = [\"**/*.py\"]\n", "p/a/one.py": "x"})
	want(t, s, ". p p/a")
}

func TestMoveBetweenComponentsRequiresBoth(t *testing.T) {
	s := newSandbox(t, base)
	s.git("mv", "p/a/one.py", "p/b/one.py")
	want(t, s, ". p p/a p/b")
}

func TestDeletingVersionTomlWhileChangingCodeFails(t *testing.T) {
	s := newSandbox(t, base)
	s.git("rm", "-q", "p/a/version.toml")
	s.write(map[string]string{"p/a/one.py": "x"})
	if _, err := s.required(); err == nil || !strings.Contains(err.Error(), "removes level p/a") {
		t.Fatalf("err = %v", err)
	}
}

func TestRemovingALevelAloneIsAllowed(t *testing.T) {
	s := newSandbox(t, base)
	s.git("rm", "-q", "p/a/version.toml")
	want(t, s, ". p")
}

func TestRemovingAWholeFolderRequiresTheLevelsAbove(t *testing.T) {
	s := newSandbox(t, base)
	s.git("rm", "-rq", "p/a")
	want(t, s, ". p")
}

func TestChangingKindKeepsTheLevelBound(t *testing.T) {
	s := newSandbox(t, base)
	s.write(map[string]string{"p/a/version.toml": pkg, "p/a/one.py": "x"})
	want(t, s, ". p p/a")
}

func TestNewLevelIsMarkedNew(t *testing.T) {
	repoFiles := []string{"version.toml", "p/version.toml", "p/c/version.toml"}
	read := func(p string) ([]byte, error) {
		return []byte(map[string]string{"version.toml": root, "p/version.toml": pkg, "p/c/version.toml": component}[p]), nil
	}
	head, err := manifest.Discover(repoFiles, read)
	if err != nil {
		t.Fatal(err)
	}
	baseTree, err := manifest.Discover(repoFiles[:2], read)
	if err != nil {
		t.Fatal(err)
	}
	reqs, err := Required(Input{Base: baseTree, Head: head, HeadFiles: repoFiles, Changed: []string{"p/c/new.py"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 3 || !reqs[2].New || reqs[1].New {
		t.Fatalf("requirements = %+v", reqs)
	}
}

func TestAdoptionOnlyValidates(t *testing.T) {
	s := newSandbox(t, map[string]string{"code.py": ""})
	s.write(map[string]string{"version.toml": root, "code.py": "x"})
	want(t, s, ".")
}

func TestUntrackedIsReadFromMain(t *testing.T) {
	s := newSandbox(t, base)
	s.write(map[string]string{"version.toml": root + "untracked = [\"q\"]\n", "q/package.json": ""})
	_, err := s.required()
	if err == nil || !strings.Contains(err.Error(), "q/package.json") {
		t.Fatalf("err = %v", err)
	}
}

func TestPathWithSpaceAndNonASCIILetters(t *testing.T) {
	s := newSandbox(t, base)
	s.write(map[string]string{"p/a/файл с пробелом.py": "x"})
	want(t, s, ". p p/a")
}

func TestGlob(t *testing.T) {
	cases := []struct {
		pattern, path string
		match         bool
	}{
		{"**/*.md", "README.md", true},
		{"**/*.md", "a/b/c.md", true},
		{"**/*.md", "a/b.py", false},
		{"**/tests/**", "a/tests/b/c.py", true},
		{"**/tests/**", "tests/c.py", true},
		{"**/tests/**", "latests/c.py", false},
		{"*.md", "a/b.md", false},
		{"a?c.txt", "abc.txt", true},
		{"a.c", "abc", false},
	}
	for _, c := range cases {
		re := mustGlob(t, c.pattern)
		if got := re.MatchString(c.path); got != c.match {
			t.Errorf("%q on %q = %v", c.pattern, c.path, got)
		}
	}
}

func mustGlob(t *testing.T, pattern string) *regexp.Regexp {
	t.Helper()
	re, err := compileGlob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	return re
}

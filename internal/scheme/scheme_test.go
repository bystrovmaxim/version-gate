package scheme

import "testing"

// ascending checks that each number is strictly below the next, and the order is total.
func ascending(t *testing.T, name string, numbers []string) {
	t.Helper()
	for i, a := range numbers {
		for j, b := range numbers {
			got, err := Compare(name, a, b)
			if err != nil {
				t.Fatal(err)
			}
			if sign(got) != sign(i-j) {
				t.Errorf("%s: compare(%s, %s) = %d", name, a, b, got)
			}
		}
	}
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

func TestSemverOrder(t *testing.T) {
	ascending(t, "semver", []string{
		"0.9.9", "1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-alpha.beta", "1.0.0-beta",
		"1.0.0-beta.2", "1.0.0-beta.11", "1.0.0-rc.1", "1.0.0", "1.0.1", "1.2.0", "1.10.0",
		"2.0.0", "99999999999999999999999.0.0",
	})
}

func TestSemverBuildMetadataIsNotGrowth(t *testing.T) {
	if got, err := Compare("semver", "1.0.0+a", "1.0.0+b"); err != nil || got != 0 {
		t.Fatalf("compare = %d, %v", got, err)
	}
}

func TestPEP440Order(t *testing.T) {
	ascending(t, "pep440", []string{
		"1.dev0", "1.0.dev456", "1.0a1", "1.0a2.dev456", "1.0a12.dev456", "1.0a12",
		"1.0b1.dev456", "1.0b2", "1.0b2.post345.dev456", "1.0b2.post345", "1.0rc1.dev456",
		"1.0rc1", "1.0", "1.0+abc", "1.0+abc.5", "1.0+5", "1.0+5.1", "1.0.post456.dev34",
		"1.0.post456", "1.0.15", "1.1.dev1", "1.1", "2!0.1",
	})
}

func TestPEP440Spellings(t *testing.T) {
	for _, pair := range [][2]string{
		{"1.0", "1"}, {"1.0.0", "1"}, {"v1.0", "1.0"}, {"1.0alpha1", "1.0a1"}, {"1.0-A.1", "1.0a1"},
		{"1.0c1", "1.0rc1"}, {"1.0pre1", "1.0rc1"}, {"1.0preview1", "1.0rc1"}, {"1.0-1", "1.0.post1"},
		{"1.0rev2", "1.0.post2"}, {"1.0r2", "1.0.post2"}, {"1.0.post", "1.0.post0"}, {"1.0a", "1.0a0"},
		{"1.0.dev", "1.0.dev0"}, {"01.02", "1.2"}, {"0.0", "0"},
		{"1.0+A-1", "1.0+a.1"},
	} {
		if got, err := Compare("pep440", pair[0], pair[1]); err != nil || got != 0 {
			t.Errorf("compare(%s, %s) = %d, %v", pair[0], pair[1], got, err)
		}
	}
}

func TestInvalidNumbers(t *testing.T) {
	cases := map[string][]string{
		"semver": {"", "one", "1", "1.2", "v1.2.3", "01.2.3", "1.2.3-", "1.2.3-01", "1.2.3+", "1.2.3.4"},
		"pep440": {"", "one", "1.0-", "1..0", "1.0+", "1.0gamma1"},
	}
	for name, numbers := range cases {
		for _, number := range numbers {
			if Valid(name, number) == nil {
				t.Errorf("%s accepted %q", name, number)
			}
		}
	}
}

func TestUnknownScheme(t *testing.T) {
	if Valid("calver", "1") == nil {
		t.Fatal("unknown scheme accepted")
	}
}

package scheme

import (
	"fmt"
	"regexp"
	"strings"
)

// semverPattern is the grammar of semver.org, without build metadata being compared.
var semverPattern = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*)?$`)

// compareSemver orders two numbers by semver precedence.
func compareSemver(a, b string) (int, error) {
	x := semverPattern.FindStringSubmatch(a)
	if x == nil {
		return 0, fmt.Errorf("%q is not a semver number", a)
	}
	y := semverPattern.FindStringSubmatch(b)
	if y == nil {
		return 0, fmt.Errorf("%q is not a semver number", b)
	}
	for i := 1; i <= 3; i++ {
		if c := compareNumbers(x[i], y[i]); c != 0 {
			return c, nil
		}
	}
	return comparePrerelease(x[4], y[4]), nil
}

// comparePrerelease orders pre-release tags; no tag is above any tag.
func comparePrerelease(a, b string) int {
	switch {
	case a == b:
		return 0
	case a == "":
		return 1
	case b == "":
		return -1
	}
	x, y := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(x) && i < len(y); i++ {
		if c := compareIdentifier(x[i], y[i]); c != 0 {
			return c
		}
	}
	return len(x) - len(y)
}

// compareIdentifier orders one tag part: numbers by value and below words.
func compareIdentifier(a, b string) int {
	aNumber, bNumber := isDigits(a), isDigits(b)
	switch {
	case aNumber && bNumber:
		return compareNumbers(a, b)
	case aNumber:
		return -1
	case bNumber:
		return 1
	}
	return strings.Compare(a, b)
}

// isDigits reports whether s holds digits only.
func isDigits(s string) bool {
	return s != "" && strings.Trim(s, "0123456789") == ""
}

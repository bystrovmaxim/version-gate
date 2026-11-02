package scheme

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	mavenPattern = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z._+-]*$`)
	mavenToken   = regexp.MustCompile(`\d+|[A-Za-z]+`)
)

// mavenRanks orders the known qualifiers; rank 5 is the plain release.
var mavenRanks = map[string]int{
	"alpha": 0, "a": 0, "beta": 1, "b": 1, "milestone": 2, "m": 2, "rc": 3, "cr": 3,
	"snapshot": 4, "": 5, "final": 5, "ga": 5, "release": 5, "sp": 6,
}

// parseMaven splits a number into tokens.
func parseMaven(s string) ([]string, error) {
	if !mavenPattern.MatchString(s) {
		return nil, fmt.Errorf("%q is not a Maven version", s)
	}
	return mavenToken.FindAllString(strings.ToLower(s), -1), nil
}

// compareMaven orders two numbers as Maven's ComparableVersion does on flat versions.
func compareMaven(a, b string) (int, error) {
	x, err := parseMaven(a)
	if err != nil {
		return 0, err
	}
	y, err := parseMaven(b)
	if err != nil {
		return 0, err
	}
	for i := 0; i < len(x) || i < len(y); i++ {
		var left, right string
		if i < len(x) {
			left = x[i]
		}
		if i < len(y) {
			right = y[i]
		}
		if c := compareMavenToken(left, right); c != 0 {
			return c, nil
		}
	}
	return 0, nil
}

// compareMavenToken orders two tokens; an empty one stands for a padded release.
func compareMavenToken(a, b string) int {
	aNumber, bNumber := isDigits(a), isDigits(b)
	switch {
	case aNumber && bNumber:
		return compareNumbers(a, b)
	case aNumber && b != "":
		return 1
	case bNumber && a != "":
		return -1
	case aNumber:
		return compareNumbers(a, "0")
	case bNumber:
		return compareNumbers("0", b)
	}
	return compareQualifiers(a, b)
}

// compareQualifiers orders two words; unknown words sort after the known, alphabetically.
func compareQualifiers(a, b string) int {
	x, xKnown := mavenRanks[a]
	y, yKnown := mavenRanks[b]
	switch {
	case xKnown && yKnown:
		return x - y
	case xKnown:
		return -1
	case yKnown:
		return 1
	}
	return strings.Compare(a, b)
}

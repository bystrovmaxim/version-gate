package scheme

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

var (
	gemPattern = regexp.MustCompile(`^\s*[0-9]+(?:\.[0-9a-zA-Z]+)*(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?\s*$`)
	gemSegment = regexp.MustCompile(`[0-9]+|[a-zA-Z]+`)
)

// parseGem splits a number into canonical segments, as Gem::Version does.
func parseGem(s string) ([]string, error) {
	if !gemPattern.MatchString(s) {
		return nil, fmt.Errorf("%q is not a RubyGems version", s)
	}
	segments := gemSegment.FindAllString(strings.ReplaceAll(s, "-", ".pre."), -1)
	word := slices.IndexFunc(segments, func(segment string) bool { return !isDigits(segment) })
	if word < 0 {
		return trimGemZeros(segments), nil
	}
	return append(trimGemZeros(segments[:word:word]), trimGemZeros(segments[word:])...), nil
}

// trimGemZeros drops the trailing zero segments.
func trimGemZeros(segments []string) []string {
	for len(segments) > 0 && isDigits(segments[len(segments)-1]) && strings.Trim(segments[len(segments)-1], "0") == "" {
		segments = segments[:len(segments)-1]
	}
	return segments
}

// compareGems orders two numbers: words sort below numbers, so 1.0.a is below 1.0.
func compareGems(a, b string) (int, error) {
	x, err := parseGem(a)
	if err != nil {
		return 0, err
	}
	y, err := parseGem(b)
	if err != nil {
		return 0, err
	}
	for i := 0; i < len(x) || i < len(y); i++ {
		left, right := "0", "0"
		if i < len(x) {
			left = x[i]
		}
		if i < len(y) {
			right = y[i]
		}
		if c := compareGemSegment(left, right); c != 0 {
			return c, nil
		}
	}
	return 0, nil
}

// compareGemSegment orders one pair of segments.
func compareGemSegment(a, b string) int {
	aNumber, bNumber := isDigits(a), isDigits(b)
	switch {
	case aNumber && bNumber:
		return compareNumbers(a, b)
	case aNumber:
		return 1
	case bNumber:
		return -1
	}
	return strings.Compare(a, b)
}

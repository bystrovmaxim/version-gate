package scheme

import (
	"fmt"
	"regexp"
	"strings"
)

// nugetPattern allows one to four release parts, a pre-release tag and metadata.
var nugetPattern = regexp.MustCompile(`^(\d+(?:\.\d+){0,3})(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)

// compareNuGet orders two numbers: four release parts, then a case-insensitive tag.
func compareNuGet(a, b string) (int, error) {
	x := nugetPattern.FindStringSubmatch(a)
	if x == nil {
		return 0, fmt.Errorf("%q is not a NuGet version", a)
	}
	y := nugetPattern.FindStringSubmatch(b)
	if y == nil {
		return 0, fmt.Errorf("%q is not a NuGet version", b)
	}
	if c := compareRelease(strings.Split(x[1], "."), strings.Split(y[1], ".")); c != 0 {
		return c, nil
	}
	return comparePrerelease(strings.ToLower(x[2]), strings.ToLower(y[2])), nil
}

package scheme

import (
	"fmt"
	"regexp"
	"strings"
)

// pep440Pattern is the version grammar of PEP 440, matched case-insensitively.
var pep440Pattern = regexp.MustCompile(`(?i)^\s*v?(?:(\d+)!)?(\d+(?:\.\d+)*)(?:[-_.]?(a|b|c|rc|alpha|beta|pre|preview)[-_.]?(\d+)?)?(?:-(\d+)|[-_.]?(post|rev|r)[-_.]?(\d+)?)?(?:[-_.]?(dev)[-_.]?(\d+)?)?(?:\+([a-z0-9]+(?:[-_.][a-z0-9]+)*))?\s*$`)

// pepVersion is a parsed number; absent parts are marked by their flags.
type pepVersion struct {
	epoch            string
	release          []string
	preLabel, preNum string
	hasPre           bool
	postNum          string
	hasPost          bool
	devNum           string
	hasDev           bool
	local            []string
}

// orZero returns s, or "0" when it is empty.
func orZero(s string) string {
	if s == "" {
		return "0"
	}
	return s
}

// parsePEP440 reads a number into its comparable parts.
func parsePEP440(s string) (pepVersion, error) {
	m := pep440Pattern.FindStringSubmatch(s)
	if m == nil {
		return pepVersion{}, fmt.Errorf("%q is not a PEP 440 number", s)
	}
	v := pepVersion{epoch: orZero(m[1]), release: strings.Split(m[2], ".")}
	for len(v.release) > 0 && strings.Trim(v.release[len(v.release)-1], "0") == "" {
		v.release = v.release[:len(v.release)-1]
	}
	if m[3] != "" {
		v.hasPre, v.preNum = true, orZero(m[4])
		switch strings.ToLower(m[3]) {
		case "a", "alpha":
			v.preLabel = "a"
		case "b", "beta":
			v.preLabel = "b"
		default:
			v.preLabel = "rc"
		}
	}
	if m[5] != "" || m[6] != "" {
		v.hasPost, v.postNum = true, orZero(m[5]+m[7])
	}
	if m[8] != "" {
		v.hasDev, v.devNum = true, orZero(m[9])
	}
	if m[10] != "" {
		v.local = regexp.MustCompile(`[-_.]`).Split(strings.ToLower(m[10]), -1)
	}
	return v, nil
}

// comparePEP440 orders two numbers by PEP 440.
func comparePEP440(a, b string) (int, error) {
	x, err := parsePEP440(a)
	if err != nil {
		return 0, err
	}
	y, err := parsePEP440(b)
	if err != nil {
		return 0, err
	}
	if c := compareNumbers(x.epoch, y.epoch); c != 0 {
		return c, nil
	}
	if c := compareRelease(x.release, y.release); c != 0 {
		return c, nil
	}
	if c := comparePre(x, y); c != 0 {
		return c, nil
	}
	if c := compareOptional(x.hasPost, y.hasPost, x.postNum, y.postNum, -1); c != 0 {
		return c, nil
	}
	if c := compareOptional(x.hasDev, y.hasDev, x.devNum, y.devNum, 1); c != 0 {
		return c, nil
	}
	return compareLocal(x.local, y.local), nil
}

// compareRelease orders release segments; a missing segment counts as zero.
func compareRelease(a, b []string) int {
	for i := 0; i < len(a) || i < len(b); i++ {
		x, y := "0", "0"
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if c := compareNumbers(x, y); c != 0 {
			return c
		}
	}
	return 0
}

// preRank places a number's pre-release: a bare .dev sorts below every pre-release.
func preRank(v pepVersion) int {
	switch {
	case v.hasPre:
		return 0
	case !v.hasPost && v.hasDev:
		return -1
	}
	return 1
}

// comparePre orders the pre-release parts of two numbers.
func comparePre(x, y pepVersion) int {
	if c := preRank(x) - preRank(y); c != 0 || !x.hasPre {
		return c
	}
	if c := strings.Compare(x.preLabel, y.preLabel); c != 0 {
		return c
	}
	return compareNumbers(x.preNum, y.preNum)
}

// compareOptional orders a part that may be absent; absent is the sign of missing.
func compareOptional(hasX, hasY bool, x, y string, missing int) int {
	switch {
	case hasX && hasY:
		return compareNumbers(x, y)
	case hasX:
		return -missing
	case hasY:
		return missing
	}
	return 0
}

// compareLocal orders local labels: none is lowest, numbers rank above words.
func compareLocal(a, b []string) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if c := compareLocalPart(a[i], b[i]); c != 0 {
			return c
		}
	}
	return len(a) - len(b)
}

// compareLocalPart orders one local segment.
func compareLocalPart(a, b string) int {
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

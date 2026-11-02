// Package scheme orders version numbers under the rules of each ecosystem.
package scheme

import (
	"fmt"
	"math/big"
)

// compareFuncs maps a scheme's name to its comparison.
var compareFuncs = map[string]func(a, b string) (int, error){
	"semver": compareSemver,
	"pep440": comparePEP440,
}

// Compare returns a negative number, zero or a positive number as a is below, equal to or above b.
func Compare(name, a, b string) (int, error) {
	compare, ok := compareFuncs[name]
	if !ok {
		return 0, fmt.Errorf("scheme %q is not supported yet", name)
	}
	return compare(a, b)
}

// Valid reports why a number is not valid under a scheme, or nil.
func Valid(name, number string) error {
	_, err := Compare(name, number, number)
	return err
}

// compareNumbers orders two strings of digits of any length.
func compareNumbers(a, b string) int {
	x, _ := new(big.Int).SetString(a, 10)
	y, _ := new(big.Int).SetString(b, 10)
	return x.Cmp(y)
}

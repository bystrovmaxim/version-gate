package change

import (
	"fmt"
	"regexp"
	"strings"
)

// compileGlob turns a path pattern into a regexp; ** crosses folders, * and ? do not.
func compileGlob(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); {
		rest := pattern[i:]
		switch {
		case strings.HasPrefix(rest, "**/"):
			b.WriteString("(?:.*/)?")
			i += 3
		case strings.HasPrefix(rest, "**"):
			b.WriteString(".*")
			i += 2
		case rest[0] == '*':
			b.WriteString("[^/]*")
			i++
		case rest[0] == '?':
			b.WriteString("[^/]")
			i++
		default:
			b.WriteString(regexp.QuoteMeta(rest[:1]))
			i++
		}
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	if err != nil {
		return nil, fmt.Errorf("pattern %q: %w", pattern, err)
	}
	return re, nil
}

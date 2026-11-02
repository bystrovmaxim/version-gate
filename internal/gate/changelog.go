package gate

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/bystrovmaxim/version-gate/internal/driver"
	"github.com/bystrovmaxim/version-gate/internal/manifest"
)

// sectionHeading matches a Keep a Changelog heading such as ## [1.2.0] - 2026-01-01.
var sectionHeading = regexp.MustCompile(`^##\s+\[?([^\]\s]+)\]?`)

// checkChangelog judges the changelog of a level whose version grew.
func checkChangelog(level manifest.Level, number string, changed []string, source driver.Source) error {
	if level.Manifest.Changelog == "" {
		return nil
	}
	file := path.Join(level.Dir, level.Manifest.Changelog)
	if !slices.Contains(changed, file) {
		return fmt.Errorf("version grew to %s, but %s did not change; add an entry", number, file)
	}
	data, err := source.Read(file)
	if err != nil {
		return fmt.Errorf("%s: cannot read: %w", file, err)
	}
	title, body := topSection(string(data))
	switch {
	case title == number:
		return nil
	case strings.EqualFold(title, "unreleased") && !blank(body):
		return nil
	case strings.EqualFold(title, "unreleased"):
		return fmt.Errorf("%s: the [Unreleased] section is empty; add an entry", file)
	case title == "":
		return fmt.Errorf("%s: no section found; add ## [%s] at the top", file, number)
	}
	return fmt.Errorf("%s: the top section is %s; want %s or [Unreleased]", file, title, number)
}

// topSection returns the first section's title and its lines.
func topSection(text string) (string, []string) {
	var title string
	var body []string
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		match := sectionHeading.FindStringSubmatch(line)
		if match != nil && title != "" {
			break
		}
		if match != nil {
			title = match[1]
		} else if title != "" {
			body = append(body, line)
		}
	}
	return title, body
}

// blank reports whether a section has no text besides sub-headings.
func blank(lines []string) bool {
	return !slices.ContainsFunc(lines, func(line string) bool {
		line = strings.TrimSpace(line)
		return line != "" && !strings.HasPrefix(line, "#")
	})
}

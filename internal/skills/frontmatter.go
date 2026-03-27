package skills

import (
	"fmt"
	"strings"
)

type Frontmatter struct {
	Name        string
	Description string
}

func ParseFrontmatter(body []byte) (Frontmatter, error) {
	text := string(body)
	lines := strings.Split(text, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return Frontmatter{}, fmt.Errorf("missing frontmatter fence")
	}

	end := -1
	for idx := 1; idx < len(lines); idx++ {
		if strings.TrimSpace(lines[idx]) == "---" {
			end = idx
			break
		}
	}
	if end == -1 {
		return Frontmatter{}, fmt.Errorf("missing closing frontmatter fence")
	}

	var fm Frontmatter
	for _, rawLine := range lines[1:end] {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return Frontmatter{}, fmt.Errorf("invalid frontmatter line: %s", line)
		}

		value = strings.TrimSpace(value)
		switch strings.TrimSpace(key) {
		case "name":
			fm.Name = value
		case "description":
			fm.Description = value
		}
	}

	return fm, nil
}

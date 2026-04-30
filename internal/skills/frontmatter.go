package skills

import (
	"fmt"
	"sort"
	"strings"
)

type Metadata map[string][]string

type Frontmatter struct {
	Name          string
	Description   string
	Compatibility string
	Metadata      Metadata
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

	fm := Frontmatter{Metadata: Metadata{}}
	inMetadata := false
	for _, rawLine := range lines[1:end] {
		line := strings.TrimRight(rawLine, "\r")
		if line == "" {
			continue
		}

		if inMetadata {
			if !isIndentedMetadataLine(line) {
				inMetadata = false
			} else {
				key, values, err := parseMetadataLine(strings.TrimSpace(line))
				if err != nil {
					return Frontmatter{}, err
				}
				fm.Metadata[key] = values
				continue
			}
		}

		trimmed := strings.TrimSpace(line)
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			return Frontmatter{}, fmt.Errorf("invalid frontmatter line: %s", trimmed)
		}

		value = strings.TrimSpace(value)
		switch strings.TrimSpace(key) {
		case "name":
			fm.Name = trimQuotes(value)
		case "description":
			fm.Description = trimQuotes(value)
		case "compatibility":
			fm.Compatibility = trimQuotes(value)
		case "metadata":
			if value != "" {
				return Frontmatter{}, fmt.Errorf("metadata must be declared as a block")
			}
			inMetadata = true
		}
	}

	return fm, nil
}

func (m Metadata) Keys() []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func isIndentedMetadataLine(line string) bool {
	return strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t")
}

func parseMetadataLine(line string) (string, []string, error) {
	key, value, ok := strings.Cut(line, ":")
	if !ok {
		return "", nil, fmt.Errorf("invalid metadata line: %s", line)
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return "", nil, fmt.Errorf("metadata key required")
	}
	if !isSupportedMetadataKey(key) {
		return "", nil, fmt.Errorf("invalid metadata key: %s", key)
	}

	values, err := parseMetadataValue(strings.TrimSpace(value))
	if err != nil {
		return "", nil, err
	}
	return key, values, nil
}

func parseMetadataValue(value string) ([]string, error) {
	if value == "" {
		return nil, fmt.Errorf("metadata value required")
	}

	if strings.HasPrefix(value, "[") {
		if !strings.HasSuffix(value, "]") {
			return nil, fmt.Errorf("invalid metadata list: %s", value)
		}
		inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
		if inner == "" {
			return []string{}, nil
		}

		parts := strings.Split(inner, ",")
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			item := trimQuotes(strings.TrimSpace(part))
			if item == "" {
				return nil, fmt.Errorf("metadata list contains empty value")
			}
			values = append(values, item)
		}
		return values, nil
	}

	return []string{trimQuotes(value)}, nil
}

func trimQuotes(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func isSupportedMetadataKey(key string) bool {
	if key == "" {
		return false
	}

	segments := strings.FieldsFunc(key, func(r rune) bool {
		return r == '_' || r == '-'
	})
	if len(segments) == 0 {
		return false
	}
	for _, segment := range segments {
		if segment == "" {
			return false
		}
		for _, r := range segment {
			if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
				return false
			}
		}
	}
	return true
}

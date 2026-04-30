package skills

import (
	"fmt"
	"regexp"
)

var metadataKeyPattern = regexp.MustCompile(`^[a-z0-9]+(?:[-_][a-z0-9]+)*$`)

func Validate(path string, body []byte) (Frontmatter, error) {
	fm, err := ParseFrontmatter(body)
	if err != nil {
		return Frontmatter{}, fmt.Errorf("%s: %w", path, err)
	}

	if fm.Name == "" || fm.Description == "" {
		return Frontmatter{}, fmt.Errorf("%s: name/description required", path)
	}

	if !metadataKeyPattern.MatchString(fm.Name) {
		return Frontmatter{}, fmt.Errorf("%s: skill name must be kebab-case", path)
	}

	if len(fm.Name) > 64 {
		return Frontmatter{}, fmt.Errorf("%s: skill name too long", path)
	}

	if len(fm.Description) > 140 {
		return Frontmatter{}, fmt.Errorf("%s: skill description too long", path)
	}

	if fm.Compatibility == "" && len(fm.Metadata) == 0 {
		return fm, nil
	}

	if fm.Compatibility != "" && len(fm.Compatibility) > 140 {
		return Frontmatter{}, fmt.Errorf("%s: compatibility too long", path)
	}

	for _, key := range fm.Metadata.Keys() {
		if !metadataKeyPattern.MatchString(key) {
			return Frontmatter{}, fmt.Errorf("%s: metadata key must be kebab-case or snake_case", path)
		}
		values := fm.Metadata[key]
		if len(values) == 0 {
			return Frontmatter{}, fmt.Errorf("%s: metadata values required for %s", path, key)
		}
		for _, value := range values {
			if value == "" {
				return Frontmatter{}, fmt.Errorf("%s: metadata values must be non-empty for %s", path, key)
			}
		}
	}

	return fm, nil
}

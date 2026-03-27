package skills

import (
	"fmt"
	"regexp"
)

var kebabCasePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func Validate(path string, body []byte) (Frontmatter, error) {
	fm, err := ParseFrontmatter(body)
	if err != nil {
		return Frontmatter{}, fmt.Errorf("%s: %w", path, err)
	}

	if fm.Name == "" || fm.Description == "" {
		return Frontmatter{}, fmt.Errorf("%s: name/description required", path)
	}

	if !kebabCasePattern.MatchString(fm.Name) {
		return Frontmatter{}, fmt.Errorf("%s: skill name must be kebab-case", path)
	}

	if len(fm.Name) > 64 {
		return Frontmatter{}, fmt.Errorf("%s: skill name too long", path)
	}

	if len(fm.Description) > 140 {
		return Frontmatter{}, fmt.Errorf("%s: skill description too long", path)
	}

	return fm, nil
}

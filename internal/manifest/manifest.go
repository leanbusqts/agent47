package manifest

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

type Manifest struct {
	RuleTemplates         []string
	ManagedTargets        []string
	PreservedTargets      []string
	RequiredTemplateFiles []string
	RequiredTemplateDirs  []string
}

func Parse(data []byte) (Manifest, error) {
	return parse(data, true)
}

func ParsePartial(data []byte) (Manifest, error) {
	return parse(data, false)
}

func parse(data []byte, validate bool) (Manifest, error) {
	var result Manifest
	sections := map[string]*[]string{
		"rule_templates":          &result.RuleTemplates,
		"managed_targets":         &result.ManagedTargets,
		"preserved_targets":       &result.PreservedTargets,
		"required_template_files": &result.RequiredTemplateFiles,
		"required_template_dirs":  &result.RequiredTemplateDirs,
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	currentSection := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			if _, ok := sections[currentSection]; !ok {
				return Manifest{}, fmt.Errorf("unknown manifest section: %s", currentSection)
			}
			continue
		}

		if currentSection == "" {
			return Manifest{}, fmt.Errorf("manifest entry outside section: %s", line)
		}

		target := sections[currentSection]
		*target = append(*target, line)
	}

	if err := scanner.Err(); err != nil {
		return Manifest{}, err
	}

	if !validate {
		return result, nil
	}

	return result, result.Validate()
}

func (m Manifest) Validate() error {
	required := map[string][]string{
		"rule_templates":          m.RuleTemplates,
		"managed_targets":         m.ManagedTargets,
		"preserved_targets":       m.PreservedTargets,
		"required_template_files": m.RequiredTemplateFiles,
		"required_template_dirs":  m.RequiredTemplateDirs,
	}

	for section, entries := range required {
		if len(entries) == 0 {
			return fmt.Errorf("Manifest section has no entries: %s", section)
		}
		for _, entry := range entries {
			if strings.TrimSpace(entry) == "" {
				return fmt.Errorf("Manifest section has empty entry: %s", section)
			}
		}
	}

	return nil
}

func (m Manifest) ContainsRuleTemplate(name string) bool {
	return contains(m.RuleTemplates, name)
}

func contains(items []string, expected string) bool {
	for _, item := range items {
		if item == expected {
			return true
		}
	}
	return false
}

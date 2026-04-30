package skills

import (
	"bytes"
	"fmt"
)

func (Service) GenerateAvailableSkillsSummaryMarkdown(skills []Skill) ([]byte, error) {
	ordered := orderedSkills(skills)
	var buf bytes.Buffer

	buf.WriteString("# Available Skills\n\n")
	buf.WriteString("Generated from local `SKILL.md` files.\n\n")

	for _, skill := range ordered {
		buf.WriteString("## ")
		buf.WriteString(skill.Name)
		buf.WriteString("\n\n")
		fmt.Fprintf(&buf, "- Description: %s\n", skill.Description)
		if skill.Compatibility != "" {
			fmt.Fprintf(&buf, "- Compatibility: %s\n", skill.Compatibility)
		}
		if len(skill.Metadata) > 0 {
			fmt.Fprintf(&buf, "- Metadata: %s\n", formatMetadataSummary(skill.Metadata))
		}
		fmt.Fprintf(&buf, "- Location: %s\n\n", skill.Location)
	}

	return buf.Bytes(), nil
}

func formatMetadataSummary(metadata Metadata) string {
	lines := formatMetadata(metadata)
	if len(lines) == 0 {
		return ""
	}
	return joinStrings(lines, "; ")
}

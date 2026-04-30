package skills

import (
	"sort"
	"strings"
)

func orderedSkills(skills []Skill) []Skill {
	ordered := append([]Skill(nil), skills...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Location == ordered[j].Location {
			return ordered[i].Name < ordered[j].Name
		}
		return ordered[i].Location < ordered[j].Location
	})
	return ordered
}

func formatMetadata(metadata Metadata) []string {
	keys := metadata.Keys()
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+"="+joinStrings(metadata[key], ", "))
	}
	return lines
}

func joinStrings(values []string, sep string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.Join(values, sep)
}

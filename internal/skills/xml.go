package skills

import (
	"bytes"
	"encoding/xml"
)

type availableSkills struct {
	XMLName xml.Name   `xml:"available_skills"`
	Skills  []xmlSkill `xml:"skill"`
}

type xmlSkill struct {
	Name          string             `xml:"name"`
	Description   string             `xml:"description"`
	Compatibility string             `xml:"compatibility,omitempty"`
	Metadata      []xmlMetadataEntry `xml:"metadata>entry,omitempty"`
	Location      string             `xml:"location"`
}

type xmlMetadataEntry struct {
	Key    string   `xml:"key,attr"`
	Values []string `xml:"value"`
}

func (Service) GenerateAvailableSkillsXML(skills []Skill) ([]byte, error) {
	doc := availableSkills{
		Skills: make([]xmlSkill, 0, len(skills)),
	}

	for _, skill := range orderedSkills(skills) {
		entry := xmlSkill{
			Name:          skill.Name,
			Description:   skill.Description,
			Compatibility: skill.Compatibility,
			Location:      skill.Location,
		}
		for _, key := range skill.Metadata.Keys() {
			entry.Metadata = append(entry.Metadata, xmlMetadataEntry{
				Key:    key,
				Values: append([]string(nil), skill.Metadata[key]...),
			})
		}
		doc.Skills = append(doc.Skills, entry)
	}

	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "  ")
	if err := encoder.Encode(doc); err != nil {
		return nil, err
	}
	if err := encoder.Flush(); err != nil {
		return nil, err
	}

	return append(buf.Bytes(), '\n'), nil
}

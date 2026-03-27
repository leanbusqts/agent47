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
	Name        string `xml:"name"`
	Description string `xml:"description"`
	Location    string `xml:"location"`
}

func (Service) GenerateAvailableSkillsXML(skills []Skill) ([]byte, error) {
	doc := availableSkills{
		Skills: make([]xmlSkill, 0, len(skills)),
	}

	for _, skill := range skills {
		doc.Skills = append(doc.Skills, xmlSkill{
			Name:        skill.Name,
			Description: skill.Description,
			Location:    skill.Location,
		})
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

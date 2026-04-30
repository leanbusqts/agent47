package skills

import (
	"bytes"
	"encoding/json"
)

type availableSkillsJSON struct {
	Skills []Skill `json:"skills"`
}

func (Service) GenerateAvailableSkillsJSON(skills []Skill) ([]byte, error) {
	doc := availableSkillsJSON{
		Skills: orderedSkills(skills),
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(doc); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

package skills

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"

	"github.com/leanbusqts/agent47/internal/templates"
)

type Skill struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Compatibility string   `json:"compatibility,omitempty"`
	Metadata      Metadata `json:"metadata,omitempty"`
	Location      string   `json:"location"`
}

type Service struct{}

func (Service) Discover(src templates.Source, root string) ([]Skill, error) {
	dirEntries, err := src.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var discovered []Skill
	for _, entry := range dirEntries {
		if !entry.IsDir() {
			continue
		}

		skillPath := path.Join(root, entry.Name(), "SKILL.md")
		body, err := src.ReadFile(skillPath)
		if err != nil {
			if isNotExist(err) {
				continue
			}
			return nil, err
		}

		fm, err := Validate(skillPath, body)
		if err != nil {
			return nil, err
		}

		discovered = append(discovered, Skill{
			Name:          fm.Name,
			Description:   fm.Description,
			Compatibility: fm.Compatibility,
			Metadata:      fm.Metadata,
			Location:      skillPath,
		})
	}

	sort.Slice(discovered, func(i, j int) bool {
		return discovered[i].Location < discovered[j].Location
	})

	if len(discovered) == 0 {
		return nil, fmt.Errorf("no valid skill templates found in %s", root)
	}

	return discovered, nil
}

func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

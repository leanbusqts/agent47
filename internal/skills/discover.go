package skills

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/leanbusqts/agent47/internal/templates"
)

type Skill struct {
	Name        string
	Description string
	Location    string
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
			Name:        fm.Name,
			Description: fm.Description,
			Location:    skillPath,
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
	return err != nil && (strings.Contains(err.Error(), "no such file") || strings.Contains(err.Error(), "file does not exist") || err == fs.ErrNotExist)
}

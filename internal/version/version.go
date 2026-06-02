package version

import (
	"os"
	"path/filepath"
	"strings"
)

var override string

func Current(repoRoot, agent47Home string) string {
	if override != "" {
		return override
	}

	var candidates []string
	if repoRoot != "" {
		candidates = append(candidates, filepath.Join(repoRoot, "VERSION"))
	}
	if agent47Home != "" {
		candidates = append(candidates, filepath.Join(agent47Home, "VERSION"))
	}

	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}

		value := strings.TrimSpace(string(data))
		if value != "" {
			return value
		}
	}

	return "unknown"
}

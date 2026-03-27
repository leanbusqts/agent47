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

	candidates := []string{
		filepath.Join(repoRoot, "VERSION"),
		filepath.Join(agent47Home, "VERSION"),
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}

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

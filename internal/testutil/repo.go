package testutil

import (
	"fmt"
	"os"
	"path/filepath"
)

func DetectRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return DetectRepoRootFrom(wd)
}

func DetectRepoRootFrom(start string) (string, error) {
	dir := start
	for {
		if FileExists(filepath.Join(dir, "go.mod")) && FileExists(filepath.Join(dir, "AGENTS.md")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not locate repository root from %s", start)
		}
		dir = parent
	}
}

func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

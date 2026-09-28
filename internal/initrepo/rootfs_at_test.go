//go:build darwin || linux

package initrepo

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestConfinedRootKeepsWritesInOpenedDirectoryAfterNameSwap(t *testing.T) {
	repositoryPath := t.TempDir()
	outside := t.TempDir()
	if err := os.Mkdir(filepath.Join(repositoryPath, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	repository, err := openConfinedRoot(repositoryPath)
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	rules, err := repository.OpenRoot("rules")
	if err != nil {
		t.Fatal(err)
	}
	defer rules.Close()

	if err := os.Rename(filepath.Join(repositoryPath, "rules"), filepath.Join(repositoryPath, "rules-original")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repositoryPath, "rules")); err != nil {
		t.Fatal(err)
	}
	if err := writeExclusive(rules, "stage", []byte("confined\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := rules.Link("stage", "linked"); err != nil {
		t.Fatal(err)
	}
	if err := rules.Rename("stage", "managed"); err != nil {
		t.Fatal(err)
	}

	assertInitFile(t, filepath.Join(repositoryPath, "rules-original", "managed"), "confined\n")
	assertInitFile(t, filepath.Join(repositoryPath, "rules-original", "linked"), "confined\n")
	for _, name := range []string{"managed", "linked"} {
		if _, err := os.Lstat(filepath.Join(outside, name)); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("descriptor-relative operation escaped to %s: %v", outside, err)
		}
	}
}

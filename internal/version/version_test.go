package version

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCurrentPrefersRepoVersionThenHomeVersion(t *testing.T) {
	repoRoot := t.TempDir()
	agent47Home := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "VERSION"), []byte("repo-version\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agent47Home, "VERSION"), []byte("home-version\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := Current(repoRoot, agent47Home); got != "repo-version" {
		t.Fatalf("expected repo version, got %s", got)
	}
}

func TestCurrentFallsBackToHomeVersionOrUnknown(t *testing.T) {
	agent47Home := t.TempDir()
	if err := os.WriteFile(filepath.Join(agent47Home, "VERSION"), []byte("home-version\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := Current("", agent47Home); got != "home-version" {
		t.Fatalf("expected home version, got %s", got)
	}
	if got := Current("", filepath.Join(t.TempDir(), "missing")); got != "unknown" {
		t.Fatalf("expected unknown, got %s", got)
	}
}

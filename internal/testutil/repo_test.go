package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectRepoRootFromFindsAncestor(t *testing.T) {
	repoRoot := t.TempDir()
	mustWriteTestutilFile(t, filepath.Join(repoRoot, "go.mod"), "module test\n")
	mustWriteTestutilFile(t, filepath.Join(repoRoot, "AGENTS.md"), "agents\n")

	start := filepath.Join(repoRoot, "internal", "pkg")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := DetectRepoRootFrom(start)
	if err != nil {
		t.Fatal(err)
	}
	if got != repoRoot {
		t.Fatalf("expected repo root %s, got %s", repoRoot, got)
	}
}

func TestDetectRepoRootUsesCurrentWorkingDirectory(t *testing.T) {
	repoRoot := t.TempDir()
	mustWriteTestutilFile(t, filepath.Join(repoRoot, "go.mod"), "module test\n")
	mustWriteTestutilFile(t, filepath.Join(repoRoot, "AGENTS.md"), "agents\n")
	nested := filepath.Join(repoRoot, "nested", "dir")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(wd) }()

	got, err := DetectRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		want = repoRoot
	}
	if got != want {
		t.Fatalf("expected repo root %s, got %s", repoRoot, got)
	}
}

func TestDetectRepoRootReturnsErrorOutsideRepo(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Chdir(outside); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(wd) }()

	if _, err := DetectRepoRoot(); err == nil {
		t.Fatal("expected repo root detection error")
	}
}

func TestDetectRepoRootFromReturnsErrorWhenMissing(t *testing.T) {
	_, err := DetectRepoRootFrom(t.TempDir())
	if err == nil {
		t.Fatal("expected repo root detection error")
	}
}

func TestFileExistsDistinguishesFilesAndDirectories(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "go.mod")
	mustWriteTestutilFile(t, file, "module test\n")
	if !FileExists(file) {
		t.Fatalf("expected file to exist: %s", file)
	}
	if FileExists(root) {
		t.Fatalf("did not expect directory to count as file: %s", root)
	}
}

func mustWriteTestutilFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

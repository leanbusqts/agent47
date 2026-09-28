package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func TestRunMapCreatesKeepsUpdatesAndProtectsContext(t *testing.T) {
	workDir := t.TempDir()
	mustWriteInitAppFile(t, filepath.Join(workDir, "go.mod"), "module example.com/project\n")
	mustWriteInitAppFile(t, filepath.Join(workDir, "cmd", "tool", "main.go"), "package main\nfunc main() {}\n")

	status, stdout, stderr := runMapTest(t, workDir, runtime.Config{})
	if status != 0 || !strings.Contains(stdout, "Created .agent47/context.md") || stderr != "" {
		t.Fatalf("unexpected create result: status=%d stdout=%q stderr=%q", status, stdout, stderr)
	}
	path := filepath.Join(workDir, ".agent47", "context.md")
	created, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	status, stdout, stderr = runMapTest(t, workDir, runtime.Config{})
	if status != 0 || !strings.Contains(stdout, "is current") || stderr != "" {
		t.Fatalf("unexpected current result: status=%d stdout=%q stderr=%q", status, stdout, stderr)
	}

	mustWriteInitAppFile(t, filepath.Join(workDir, "internal", "service.go"), "package service\n")
	status, stdout, stderr = runMapTest(t, workDir, runtime.Config{})
	if status != 0 || !strings.Contains(stdout, "Updated .agent47/context.md") || stderr != "" {
		t.Fatalf("unexpected update result: status=%d stdout=%q stderr=%q", status, stdout, stderr)
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(created, updated) {
		t.Fatal("expected structural change to update context")
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("manual\n"); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	status, _, stderr = runMapTest(t, workDir, runtime.Config{})
	if status != 1 || !strings.Contains(stderr, "rerun with --force") {
		t.Fatalf("expected manual edit failure, status=%d stderr=%q", status, stderr)
	}
	status, stdout, stderr = runMapTest(t, workDir, runtime.Config{}, "--force")
	if status != 0 || !strings.Contains(stdout, "Updated .agent47/context.md") || stderr != "" {
		t.Fatalf("unexpected forced update: status=%d stdout=%q stderr=%q", status, stdout, stderr)
	}
}

func TestRunMapRejectsUnsupportedOptions(t *testing.T) {
	status, stdout, stderr := runMapTest(t, t.TempDir(), runtime.Config{}, "--preview")
	if status != 2 || stdout != "" || !strings.Contains(stderr, mapUsage) {
		t.Fatalf("expected usage error, status=%d stdout=%q stderr=%q", status, stdout, stderr)
	}
}

func TestRunMapRejectsRuntimeHomeCollision(t *testing.T) {
	workDir := t.TempDir()
	cfg := runtime.Config{Agent47Home: filepath.Join(workDir, ".agent47")}
	status, _, stderr := runMapTest(t, workDir, cfg)
	if status != 1 || !strings.Contains(stderr, "conflicts with the Agent47 runtime home") {
		t.Fatalf("expected runtime collision, status=%d stderr=%q", status, stderr)
	}
	assertInitAppNotExists(t, filepath.Join(workDir, ".agent47"))
}

func runMapTest(t *testing.T, workDir string, cfg runtime.Config, args ...string) (int, string, string) {
	t.Helper()
	oldWorkDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWorkDir) }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))
	status := root.runMap(context.Background(), cfg, args)
	return status, stdout.String(), stderr.String()
}

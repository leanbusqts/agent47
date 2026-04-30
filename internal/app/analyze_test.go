package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func TestRunAnalyzeMatchesConciseGoldenForEmptyRepo(t *testing.T) {
	workDir := t.TempDir()
	stdout, stderr, status := runAnalyzeInDir(t, workDir, "analyze")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}

	assertGoldenOutput(t, "analyze_empty.golden", stdout)
}

func TestRunAnalyzeJSON(t *testing.T) {
	workDir := t.TempDir()
	stdout, stderr, status := runAnalyzeInDir(t, workDir, "analyze", "--json")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}
	if !bytes.Contains([]byte(stdout), []byte(`"install_plan"`)) {
		t.Fatalf("expected install plan output, got %s", stdout)
	}
}

func TestRunAnalyzeVerboseShowsConflictSection(t *testing.T) {
	workDir := t.TempDir()
	mustWriteFile(t, filepath.Join(workDir, "package.json"), `{"dependencies":{"react":"1.0.0","express":"1.0.0"}}`)
	if err := os.MkdirAll(filepath.Join(workDir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(workDir, "api"), 0o755); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, status := runAnalyzeInDir(t, workDir, "analyze", "--verbose")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}
	if !bytes.Contains([]byte(stdout), []byte("Conflict")) {
		t.Fatalf("expected conflict section, got %s", stdout)
	}
}

func TestRunAnalyzeMatchesVerboseGoldenForDominantInfraRepo(t *testing.T) {
	workDir := t.TempDir()
	mustWriteFile(t, filepath.Join(workDir, "main.tf"), "terraform {}\n")

	stdout, stderr, status := runAnalyzeInDir(t, workDir, "analyze", "--verbose")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}

	assertGoldenOutput(t, "analyze_infra_verbose.golden", stdout)
}

func TestRunAnalyzeVerboseShowsTestingStacksAndMappedSkills(t *testing.T) {
	workDir := t.TempDir()
	mustWriteFile(t, filepath.Join(workDir, "package.json"), `{"devDependencies":{"vitest":"1.0.0","playwright":"1.0.0"}}`)
	mustWriteFile(t, filepath.Join(workDir, "go.mod"), "module example.com/test\n")
	mustWriteFile(t, filepath.Join(workDir, "service_test.go"), "package main\n")
	mustWriteFile(t, filepath.Join(workDir, "vitest.config.ts"), "export default {}\n")
	mustWriteFile(t, filepath.Join(workDir, "playwright.config.ts"), "export default {}\n")

	stdout, stderr, status := runAnalyzeInDir(t, workDir, "analyze", "--verbose")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}
	if !bytes.Contains([]byte(stdout), []byte("Testing stacks")) {
		t.Fatalf("expected testing stacks section, got %s", stdout)
	}
	if !bytes.Contains([]byte(stdout), []byte("refactor")) {
		t.Fatalf("expected refactor skill in output, got %s", stdout)
	}
	if !bytes.Contains([]byte(stdout), []byte("optimize")) {
		t.Fatalf("expected optimize skill in output, got %s", stdout)
	}
}

func TestRunAddAgentPreviewDoesNotWriteFiles(t *testing.T) {
	env := newAddAgentEnv(t)
	if err := os.MkdirAll(filepath.Join(env.workDir, "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(env.workDir, "go.mod"), "module example.com/test\n")

	status, stdout, stderr := env.run(t, "add-agent", "--preview")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}
	if !bytes.Contains([]byte(stdout), []byte("Preview")) {
		t.Fatalf("expected preview output, got %s", stdout)
	}
	if !bytes.Contains([]byte(stdout), []byte("skills/AVAILABLE_SKILLS.json")) {
		t.Fatalf("expected preview to include JSON skills index, got %s", stdout)
	}
	if !bytes.Contains([]byte(stdout), []byte("skills/SUMMARY.md")) {
		t.Fatalf("expected preview to include summary skills index, got %s", stdout)
	}
	assertNotExists(t, filepath.Join(env.workDir, "AGENTS.md"))
}

func runAnalyzeInDir(t *testing.T, workDir string, args ...string) (string, string, int) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))
	status := root.Run(context.Background(), runtime.Config{Version: "vtest"}, args)
	return stdout.String(), stderr.String(), status
}

func assertGoldenOutput(t *testing.T, name string, got string) {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	if got != string(data) {
		t.Fatalf("golden mismatch for %s\nwant:\n%s\ngot:\n%s", name, string(data), got)
	}
}

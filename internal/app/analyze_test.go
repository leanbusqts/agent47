package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leanbusqts/agent47/internal/analyze"
	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/resolve"
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

func TestAnalyzeAndResolveUsesStandardAnalysisMode(t *testing.T) {
	result, set, err := analyzeAndResolve(t.TempDir(), resolve.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if result.AgentReadiness != nil || result.DeepAudit != nil {
		t.Fatalf("standard analysis unexpectedly enabled deep mode: %+v", result)
	}
	if len(set.Bundles) == 0 || set.Bundles[0] != "base" {
		t.Fatalf("expected base install set for an empty repository, got %+v", set)
	}
}

func TestRunAnalyzeRejectsUnexpectedFlagOnStderr(t *testing.T) {
	stdout, stderr, status := runAnalyzeInDir(t, t.TempDir(), "analyze", "--bogus")
	if status != 2 {
		t.Fatalf("expected usage status 2, got %d", status)
	}
	if stdout != "" || !bytes.Contains([]byte(stderr), []byte("Usage: afs analyze")) {
		t.Fatalf("expected diagnostic-only usage output, stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestRunAnalyzeVerboseShowsConflictSection(t *testing.T) {
	workDir := t.TempDir()
	mustWriteAppAnalyzeFile(t, filepath.Join(workDir, "package.json"), `{"dependencies":{"react":"1.0.0","express":"1.0.0"}}`)
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

func TestPrintAgentPolicyTextShowsWarningsWithoutEvidence(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))
	printAgentPolicyText(root, analyze.AgentPolicy{
		Warnings: []string{"repository entry limit reached"},
		Tools: analyze.AgentPolicyTools{
			Codex: analyze.ToolPolicy{State: analyze.PolicyInvalid, Warnings: []string{"malformed config"}},
		},
	}, false)
	if !bytes.Contains(stdout.Bytes(), []byte("policy warning: repository entry limit reached")) || !bytes.Contains(stdout.Bytes(), []byte("codex warning: malformed config")) {
		t.Fatalf("expected policy warnings without evidence, got %s", stdout.String())
	}
}

func TestRunAnalyzeMatchesVerboseGoldenForDominantInfraRepo(t *testing.T) {
	workDir := t.TempDir()
	mustWriteAppAnalyzeFile(t, filepath.Join(workDir, "main.tf"), "terraform {}\n")

	stdout, stderr, status := runAnalyzeInDir(t, workDir, "analyze", "--verbose")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}

	assertGoldenOutput(t, "analyze_infra_verbose.golden", stdout)
}

func TestRunAnalyzeVerboseShowsTestingStacksWithoutSkillMapping(t *testing.T) {
	workDir := t.TempDir()
	mustWriteAppAnalyzeFile(t, filepath.Join(workDir, "package.json"), `{"devDependencies":{"vitest":"1.0.0","playwright":"1.0.0"}}`)
	mustWriteAppAnalyzeFile(t, filepath.Join(workDir, "go.mod"), "module example.com/test\n")
	mustWriteAppAnalyzeFile(t, filepath.Join(workDir, "service_test.go"), "package main\n")
	mustWriteAppAnalyzeFile(t, filepath.Join(workDir, "vitest.config.ts"), "export default {}\n")
	mustWriteAppAnalyzeFile(t, filepath.Join(workDir, "playwright.config.ts"), "export default {}\n")

	stdout, stderr, status := runAnalyzeInDir(t, workDir, "analyze", "--verbose")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}
	if !bytes.Contains([]byte(stdout), []byte("Testing stacks")) {
		t.Fatalf("expected testing stacks section, got %s", stdout)
	}
	if bytes.Contains([]byte(stdout), []byte("Skills")) {
		t.Fatalf("did not expect removed skill mapping in output, got %s", stdout)
	}
}

func TestRunAnalyzeEvidenceShowsClassificationEvidence(t *testing.T) {
	workDir := t.TempDir()
	mustWriteAppAnalyzeFile(t, filepath.Join(workDir, "go.mod"), "module example.com/test\n")
	mustWriteAppAnalyzeFile(t, filepath.Join(workDir, "install.sh"), "#!/usr/bin/env bash\n")
	if err := os.MkdirAll(filepath.Join(workDir, "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(workDir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, status := runAnalyzeInDir(t, workDir, "analyze", "--evidence")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}
	if !bytes.Contains([]byte(stdout), []byte("project-type: Resolved project type cli")) {
		t.Fatalf("expected project-type evidence in output, got %s", stdout)
	}
	if !bytes.Contains([]byte(stdout), []byte("technology: Resolved technology shell")) {
		t.Fatalf("expected technology evidence in output, got %s", stdout)
	}
}

func TestRunAnalyzeDeepRendersReadinessWithoutFailingOnFindings(t *testing.T) {
	workDir := t.TempDir()
	stdout, stderr, status := runAnalyzeInDir(t, workDir, "analyze", "--deep")
	if status != 0 {
		t.Fatalf("expected findings to keep status 0, got %d: %s", status, stderr)
	}
	for _, marker := range []string{"Agent readiness", "status: not_ready", "Deep audit"} {
		if !bytes.Contains([]byte(stdout), []byte(marker)) {
			t.Fatalf("expected %q in deep output, got %s", marker, stdout)
		}
	}
}

func TestRunAnalyzeDeepVerboseEvidenceIncludesActionsAndEvidence(t *testing.T) {
	workDir := t.TempDir()
	mustWriteAppAnalyzeFile(t, filepath.Join(workDir, "README.md"), "[missing](docs/missing.md)\n")
	stdout, stderr, status := runAnalyzeInDir(t, workDir, "analyze", "--deep", "--verbose", "--evidence")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}
	for _, marker := range []string{"PA-008-", "action:", "location: README.md:1", "evidence: referenced path is absent"} {
		if !bytes.Contains([]byte(stdout), []byte(marker)) {
			t.Fatalf("expected %q in detailed deep output, got %s", marker, stdout)
		}
	}
}

func TestRunAnalyzeDeepJSONIncludesVersionedSchema(t *testing.T) {
	stdout, stderr, status := runAnalyzeInDir(t, t.TempDir(), "analyze", "--deep", "--json")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}
	for _, marker := range []string{`"analysis_version": 1`, `"agent_readiness"`, `"deep_audit"`} {
		if !bytes.Contains([]byte(stdout), []byte(marker)) {
			t.Fatalf("expected %s in JSON output, got %s", marker, stdout)
		}
	}
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

func mustWriteAppAnalyzeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertGoldenOutput(t *testing.T, name string, got string) {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	want := strings.ReplaceAll(string(data), "\r\n", "\n")
	got = strings.ReplaceAll(got, "\r\n", "\n")
	if got != want {
		t.Fatalf("golden mismatch for %s\nwant:\n%s\ngot:\n%s", name, string(data), got)
	}
}

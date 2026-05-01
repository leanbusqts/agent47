package analyze

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeEmptyRepoIsLowSignal(t *testing.T) {
	result, err := (Service{}).Analyze(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !result.LowSignal {
		t.Fatal("expected low-signal result")
	}
	if result.RepoShape != "empty" {
		t.Fatalf("expected empty repo shape, got %s", result.RepoShape)
	}
	if result.UnresolvedConflict {
		t.Fatal("did not expect unresolved conflict")
	}
}

func TestAnalyzeDetectsUnresolvedConflict(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "package.json"), `{"dependencies":{"react":"1.0.0","express":"1.0.0"}}`)
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "api"), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := (Service{}).Analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if !result.UnresolvedConflict {
		t.Fatal("expected unresolved conflict")
	}
	if len(result.ConflictProjectTypes) != 2 || result.ConflictProjectTypes[0] != "backend" || result.ConflictProjectTypes[1] != "frontend" {
		t.Fatalf("unexpected conflict project types: %v", result.ConflictProjectTypes)
	}
}

func TestAnalyzeAllowsSupportedCLIScriptsComposition(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "go.mod"), "module example.com/test\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "install.sh"), "#!/usr/bin/env bash\n")
	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := (Service{}).Analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.UnresolvedConflict {
		t.Fatal("did not expect unresolved conflict")
	}
}

func TestAnalyzeAllowsSupportedCLIMonorepoComposition(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "package.json"), `{"devDependencies":{"turbo":"1.0.0"}}`)
	mustWriteAnalyzeFile(t, filepath.Join(root, "pnpm-workspace.yaml"), "packages:\n  - apps/*\n")
	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "packages"), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := (Service{}).Analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.UnresolvedConflict {
		t.Fatal("did not expect unresolved conflict")
	}
	if !hasProjectType(result.ProjectTypes, "cli") || !hasProjectType(result.ProjectTypes, "monorepo-tooling") {
		t.Fatalf("expected cli and monorepo-tooling project types, got %v", result.ProjectTypes)
	}
}

func TestAnalyzeAllowsSupportedPluginDesktopComposition(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "package.json"), `{"dependencies":{"electron":"1.0.0"}}`)
	mustWriteAnalyzeFile(t, filepath.Join(root, "plugin.json"), `{"name":"sample-plugin"}`)
	if err := os.MkdirAll(filepath.Join(root, "plugins"), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := (Service{}).Analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.UnresolvedConflict {
		t.Fatal("did not expect unresolved conflict")
	}
	if !hasProjectType(result.ProjectTypes, "desktop") || !hasProjectType(result.ProjectTypes, "plugin") {
		t.Fatalf("expected desktop and plugin project types, got %v", result.ProjectTypes)
	}
}

func TestAnalyzeDetectsInfraSignals(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "main.tf"), "terraform {}\n")
	if err := os.MkdirAll(filepath.Join(root, "charts"), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := (Service{}).Analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasProjectType(result.ProjectTypes, "infra") {
		t.Fatalf("expected infra project type, got %v", result.ProjectTypes)
	}
}

func TestAnalyzeDetectsDedicatedTestingStacks(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "package.json"), `{
  "devDependencies": {
    "vitest": "^1.0.0",
    "jest": "^29.0.0",
    "@playwright/test": "^1.0.0",
    "cypress": "^13.0.0"
  }
}`)
	mustWriteAnalyzeFile(t, filepath.Join(root, "go.mod"), "module example.com/test\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "internal", "service_test.go"), "package internal\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "tests", "smoke.bats"), "#!/usr/bin/env bats\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "vitest.config.ts"), "export default {}\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "playwright.config.ts"), "export default {}\n")

	result, err := (Service{}).Analyze(root)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{
		"vitest":     false,
		"jest":       false,
		"playwright": false,
		"cypress":    false,
		"go-test":    false,
		"bats":       false,
	}
	for _, tech := range result.Technologies {
		if _, ok := want[tech.ID]; ok {
			want[tech.ID] = true
		}
	}
	for id, seen := range want {
		if !seen {
			t.Fatalf("expected testing technology %s to be detected: %v", id, result.Technologies)
		}
	}
}

func TestAnalyzeIgnoresAuxiliaryDirectoriesForPrimaryStackDetection(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "templates", "tool.py"), "print('template helper')\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "tests", "fixture.py"), "print('test helper')\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "vendor", "dep.py"), "print('vendored helper')\n")

	result, err := (Service{}).Analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, tech := range result.Technologies {
		if tech.ID == "python" {
			t.Fatalf("did not expect python detection from auxiliary directories: %v", result.Technologies)
		}
		if tech.ID == "shell" {
			t.Fatalf("did not expect shell detection from auxiliary directories: %v", result.Technologies)
		}
	}
}

func TestAnalyzeStillDetectsTestingStacksFromTestsDirectory(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "tests", "smoke.bats"), "#!/usr/bin/env bats\n")

	result, err := (Service{}).Analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasTechnology(result.Technologies, "bats") {
		t.Fatalf("expected bats detection from tests directory, got %v", result.Technologies)
	}
}

func TestAnalyzeIncludesClassificationEvidence(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "go.mod"), "module example.com/test\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "install.sh"), "#!/usr/bin/env bash\n")
	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := (Service{}).Analyze(root)
	if err != nil {
		t.Fatal(err)
	}

	projectEvidence, ok := findEvidence(result.Evidence, "project-type", "cli")
	if !ok {
		t.Fatalf("expected cli project-type evidence, got %v", result.Evidence)
	}
	if len(projectEvidence.SourcePaths) == 0 {
		t.Fatalf("expected cli project-type evidence sources, got %v", projectEvidence)
	}

	technologyEvidence, ok := findEvidence(result.Evidence, "technology", "shell")
	if !ok {
		t.Fatalf("expected shell technology evidence, got %v", result.Evidence)
	}
	if len(technologyEvidence.SourcePaths) == 0 {
		t.Fatalf("expected shell technology evidence sources, got %v", technologyEvidence)
	}
}

func mustWriteAnalyzeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasProjectType(projectTypes []DetectedProjectType, want string) bool {
	for _, projectType := range projectTypes {
		if projectType.ID == want {
			return true
		}
	}
	return false
}

func hasTechnology(technologies []DetectedTechnology, want string) bool {
	for _, technology := range technologies {
		if technology.ID == want {
			return true
		}
	}
	return false
}

func findEvidence(items []EvidenceItem, kind string, fragment string) (EvidenceItem, bool) {
	for _, item := range items {
		if item.Kind == kind && strings.Contains(item.Detail, fragment) {
			return item, true
		}
	}
	return EvidenceItem{}, false
}

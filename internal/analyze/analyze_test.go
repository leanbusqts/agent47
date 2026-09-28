package analyze

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestAnalyzeHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (Service{}).AnalyzeContext(ctx, t.TempDir(), AnalyzeOptions{Deep: true})
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestAnalyzeBoundsManifestReads(t *testing.T) {
	root := t.TempDir()
	body := strings.Repeat(" ", int(maxInspectedFileBytes)+1) + `{"dependencies":{"react":"1.0.0"}}`
	mustWriteAnalyzeFile(t, filepath.Join(root, "package.json"), body)

	result, err := (Service{}).Analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if hasProjectType(result.ProjectTypes, "frontend") {
		t.Fatalf("content beyond the manifest read limit must not influence classification: %+v", result.ProjectTypes)
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

func TestAnalyzeDetectsSwiftPMMacOSRepoAsDesktopNotMobile(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "Package.swift"), `// swift-tools-version: 5.10
import PackageDescription

let package = Package(
    name: "sauron",
    platforms: [
        .macOS(.v14)
    ],
    products: [
        .executable(name: "sauron", targets: ["sauron"])
    ]
)
`)
	mustWriteAnalyzeFile(t, filepath.Join(root, "sources", "sauron", "main.swift"), "import AppKit\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "scripts", "install.sh"), "#!/usr/bin/env bash\n")

	result, err := (Service{}).Analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasProjectType(result.ProjectTypes, "desktop") {
		t.Fatalf("expected desktop project type, got %v", result.ProjectTypes)
	}
	if hasProjectType(result.ProjectTypes, "mobile") {
		t.Fatalf("did not expect mobile project type, got %v", result.ProjectTypes)
	}
	if !hasProjectType(result.ProjectTypes, "scripts") {
		t.Fatalf("expected scripts project type, got %v", result.ProjectTypes)
	}
}

func TestAnalyzeDetectsSwiftPMIOSRepoAsMobile(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "Package.swift"), `// swift-tools-version: 5.10
import PackageDescription

let package = Package(
    name: "ios-app",
    platforms: [
        .iOS(.v17)
    ]
)
`)
	mustWriteAnalyzeFile(t, filepath.Join(root, "Sources", "App", "main.swift"), "import SwiftUI\n")

	result, err := (Service{}).Analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasProjectType(result.ProjectTypes, "mobile") {
		t.Fatalf("expected mobile project type, got %v", result.ProjectTypes)
	}
	if hasProjectType(result.ProjectTypes, "desktop") {
		t.Fatalf("did not expect desktop project type, got %v", result.ProjectTypes)
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

func TestAnalyzeDetectsProjectLocalAgentPolicies(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "AGENTS.md"), "# Root policy\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "services", "api", "AGENTS.md"), "# API policy\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "CLAUDE.md"), "# Claude instructions\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, ".codex", "config.toml"), "approval_policy = \"on-request\"\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, ".opencode", "opencode.jsonc"), "{\n  // project-local only\n  \"schema\": 1,\n}\n")

	result, err := (Service{}).Analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.AgentPolicy.AgentsFiles) != 2 {
		t.Fatalf("expected root and nested AGENTS.md files, got %+v", result.AgentPolicy.AgentsFiles)
	}
	if result.AgentPolicy.AgentsFiles[0].Path != "AGENTS.md" || result.AgentPolicy.AgentsFiles[1].Path != "services/api/AGENTS.md" {
		t.Fatalf("expected stable repository-relative policy paths, got %+v", result.AgentPolicy.AgentsFiles)
	}
	if result.AgentPolicy.Tools.Claude.State != PolicyValid || result.AgentPolicy.Tools.Codex.State != PolicyValid || result.AgentPolicy.Tools.OpenCode.State != PolicyValid {
		t.Fatalf("expected all tool evidence to be valid, got %+v", result.AgentPolicy.Tools)
	}
}

func TestAnalyzeReportsMalformedVendorConfigWithoutExposingValues(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, ".codex", "config.toml"), "api_key = \"secret-value\"\nbroken = [\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, ".opencode", "opencode.json"), "{\"token\":\"secret-value\"")

	result, err := (Service{}).Analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.AgentPolicy.Tools.Codex.State != PolicyInvalid || result.AgentPolicy.Tools.OpenCode.State != PolicyInvalid {
		t.Fatalf("expected malformed configs to be invalid, got %+v", result.AgentPolicy.Tools)
	}
	if strings.Contains(fmt.Sprint(result.AgentPolicy), "secret-value") {
		t.Fatalf("policy result exposed a configuration value: %+v", result.AgentPolicy)
	}
}

func TestAnalyzeDeepReadyRepository(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "AGENTS.md"), "# Policy\nChanges require approval. Never perform destructive work without rollback.\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "README.md"), "# Service\nBuild with `make build` and verify with `make test`.\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "Makefile"), "build:\n\tgo build ./...\n\ntest:\n\tgo test ./...\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "go.mod"), "module example.com/ready\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "service_test.go"), "package ready\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "rules", "security-global.yaml"), "rules:\n  - security boundary\n")

	result, err := (Service{}).AnalyzeWithOptions(root, AnalyzeOptions{Deep: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.AnalysisVersion != 1 || result.AgentReadiness == nil || result.DeepAudit == nil {
		t.Fatalf("expected versioned deep result, got %+v", result)
	}
	if result.AgentReadiness.Status != ReadinessReady {
		t.Fatalf("expected ready repository, got %+v", result.AgentReadiness)
	}
	if len(result.DeepAudit.DriftFindings) != 0 || len(result.DeepAudit.PromptFindings) != 0 {
		t.Fatalf("expected clean deterministic audit, got %+v", result.DeepAudit)
	}
}

func TestAnalyzeDeepDoesNotTreatPromptDirectoryAsPromptFile(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "prompts"), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := (Service{}).AnalyzeWithOptions(root, AnalyzeOptions{Deep: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.DeepAudit.PromptSurfaces != 0 || len(result.DeepAudit.Skipped) != 0 {
		t.Fatalf("empty prompt directory must not be inspected as a file: %+v", result.DeepAudit)
	}
}

func TestCurrentLiteContractIsNotClassifiedAsLegacyScaffold(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "AGENTS.md"), "# Policy\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "rules", "security-global.yaml"), "rules: []\n")

	result, err := (Service{}).Analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.ManagedState.LegacyScaffold {
		t.Fatalf("current Lite contract must not be classified as legacy: %+v", result.ManagedState)
	}
}

func TestRemovedPayloadsAreClassifiedAsLegacyScaffold(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, ".agents", "specs", "spec.yml"), "version: 1\n")

	result, err := (Service{}).Analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if !result.ManagedState.LegacyScaffold {
		t.Fatalf("removed managed payload must be classified as legacy: %+v", result.ManagedState)
	}
}

func TestAnalyzeDeepReportsDeterministicPromptAndDriftFindings(t *testing.T) {
	root := t.TempDir()
	policy := "# Vendor-neutral contract\nYou must use Codex.\nAll changes must include deterministic verification.\n"
	mustWriteAnalyzeFile(t, filepath.Join(root, "AGENTS.md"), policy)
	mustWriteAnalyzeFile(t, filepath.Join(root, "services", "api", "AGENTS.md"), "All changes must include deterministic verification.\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "README.md"), "[Missing contract](docs/missing.md)\nRun `make ghost`.\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "rules", "a.yaml"), "- id: RULE-1\n  rule: Use alpha.\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "rules", "b.yaml"), "- id: RULE-1\n  rule: Use beta.\n")

	result, err := (Service{}).AnalyzeWithOptions(root, AnalyzeOptions{Deep: true})
	if err != nil {
		t.Fatal(err)
	}
	wantRules := map[string]bool{"PA-001": false, "PA-002": false, "PA-007": false, "PA-008": false}
	for _, finding := range append(append([]Finding{}, result.DeepAudit.PromptFindings...), result.DeepAudit.DriftFindings...) {
		if _, ok := wantRules[finding.RuleID]; ok {
			wantRules[finding.RuleID] = true
		}
		if finding.FindingID == "" || finding.Title == "" || finding.Evidence == "" || finding.Confidence == "" || finding.SuggestedAction == "" {
			t.Fatalf("finding does not satisfy the shared schema: %+v", finding)
		}
	}
	for rule, found := range wantRules {
		if !found {
			t.Fatalf("expected %s finding, got prompt=%+v drift=%+v", rule, result.DeepAudit.PromptFindings, result.DeepAudit.DriftFindings)
		}
	}
}

func TestAnalyzeDeepTruncationPreventsReady(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "AGENTS.md"), "# Policy\nChanges require approval and rollback.\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "README.md"), "# Service\nUse `make build` and `make test`.\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "Makefile"), "build:\n\ttrue\ntest:\n\ttrue\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "go.mod"), "module example.com/truncated\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "service_test.go"), "package truncated\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "rules", "security-global.yaml"), "rules:\n  - security\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "prompts", "large.txt"), strings.Repeat("A", int(maxInspectedFileBytes)+1))

	result, err := (Service{}).AnalyzeWithOptions(root, AnalyzeOptions{Deep: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.DeepAudit.Truncations) == 0 {
		t.Fatal("expected explicit truncation")
	}
	if result.AgentReadiness.Status == ReadinessReady {
		t.Fatalf("truncated audit must not report ready: %+v", result.AgentReadiness)
	}
}

func TestAnalyzeDeepSkipsExternalSymlinkAndBinaryPrompt(t *testing.T) {
	root := t.TempDir()
	external := filepath.Join(t.TempDir(), "outside.md")
	mustWriteAnalyzeFile(t, external, "never read me\n")
	if err := os.MkdirAll(filepath.Join(root, "prompts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(root, "prompts", "outside.md")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "prompts", "binary.txt"), []byte{0xff, 0xfe, 0x00}, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := (Service{}).AnalyzeWithOptions(root, AnalyzeOptions{Deep: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.DeepAudit.Skipped) < 2 {
		t.Fatalf("expected symlink and binary skip records, got %+v", result.DeepAudit.Skipped)
	}
}

func TestAnalyzeDeepRejectsUnreadableRootContractBytes(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte{0xff, 0xfe, 0x00}, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := (Service{}).AnalyzeWithOptions(root, AnalyzeOptions{Deep: true})
	if err == nil || !strings.Contains(err.Error(), "required root contract") {
		t.Fatalf("expected required root contract error, got %v", err)
	}
}

func TestPolicyDetectionIsConservativeAndDoesNotMisclassifyOwnedPaths(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "CLAUDE.md"), "# valid\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, ".claude", "rules", "oversized.md"), strings.Repeat("x", int(maxInspectedFileBytes)+1))
	mustWriteAnalyzeFile(t, filepath.Join(root, ".codex", "config.toml"), "approval_policy = \"unterminated\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, ".opencode", "opencode.jsonc"), "{} /* unterminated")
	mustWriteAnalyzeFile(t, filepath.Join(root, "rules", "security-global.yaml"), "rules: []\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, ".agents", "notes.md"), "owned\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "skills", "legacy", "SKILL.md"), "legacy\n")

	result, err := (Service{}).Analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.AgentPolicy.Tools.Claude.State != PolicyInvalid || result.AgentPolicy.Tools.Codex.State != PolicyInvalid || result.AgentPolicy.Tools.OpenCode.State != PolicyInvalid {
		t.Fatalf("expected conservative invalid states, got %+v", result.AgentPolicy.Tools)
	}
	if fmt.Sprint(result.AgentPolicy.Legacy) != "[skills/]" {
		t.Fatalf("expected only removed payloads to be legacy, got %v", result.AgentPolicy.Legacy)
	}
}

func TestVendorConfigParsersRequireStructuredRootValues(t *testing.T) {
	for _, body := range []string{"null", "[]", `"string"`} {
		if validJSONConfig([]byte(body), false) {
			t.Fatalf("expected non-object JSON config to be rejected: %s", body)
		}
	}
	if !validJSONConfig([]byte(`{"schema":1}`), false) {
		t.Fatal("expected object JSON config to be accepted")
	}
	for _, body := range []string{"[]", "[[]]", "[[ ]]"} {
		if validMinimalTOML([]byte(body)) {
			t.Fatalf("expected empty TOML header to be rejected: %s", body)
		}
	}
	if !validMinimalTOML([]byte("[tool]\nenabled = true\n")) {
		t.Fatal("expected populated TOML config to be accepted")
	}
}

func TestAnalyzeDeepConflictMakesRepositoryNotReadyAndProducesClaims(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "AGENTS.md"), "# Policy\nChanges require approval. Never perform destructive work without rollback.\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "README.md"), "# Service\nUse `make build` and `make test`.\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "Makefile"), "build:\n\tgo build ./...\ntest:\n\tgo test ./...\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "go.mod"), "module example.com/conflict\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "service_test.go"), "package conflict\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "rules", "security-global.yaml"), "rules: []\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, ".codex", "rules", "a.yaml"), "- id: RULE-1\n  rule: Use alpha.\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, ".codex", "rules", "b.yaml"), "- id: RULE-1\n  rule: Use beta.\n")

	result, err := (Service{}).AnalyzeWithOptions(root, AnalyzeOptions{Deep: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.AgentReadiness.Status != ReadinessNotReady || result.AgentReadiness.Dimensions.Context != DimensionConflicting {
		t.Fatalf("expected policy conflict to block readiness, got %+v", result.AgentReadiness)
	}
	claim, ok := findClaim(result.AgentReadiness.Claims, "policy.consistency")
	if !ok || claim.Status != ClaimContradicted {
		t.Fatalf("expected contradicted consistency claim, got %+v", result.AgentReadiness.Claims)
	}
	if len(result.AgentReadiness.Claims) == 0 {
		t.Fatal("expected structured readiness claims")
	}
}

func TestAnalyzeDeepNegatedCommandsDoNotSatisfyOperability(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "AGENTS.md"), "# Policy\nApproval is required. Never perform destructive work without rollback.\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "README.md"), "# Service\nDo not run make test or make build.\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "Makefile"), "build:\n\ttrue\ntest:\n\ttrue\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "go.mod"), "module example.com/negative\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "service_test.go"), "package negative\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "rules", "security-global.yaml"), "rules: []\n")

	result, err := (Service{}).AnalyzeWithOptions(root, AnalyzeOptions{Deep: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.AgentReadiness.Dimensions.Operability != DimensionMissing {
		t.Fatalf("negated prose must not satisfy operability: %+v", result.AgentReadiness)
	}
}

func TestAnalyzeDeepReportsPackageTaskAndManagedPathDrift(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "README.md"), "Run `npm run ghost`, `task absent`, `sample ghost`, and inspect `rules/missing.yaml`.\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "package.json"), `{"name":"sample","scripts":{"test":"echo ok"}}`)
	mustWriteAnalyzeFile(t, filepath.Join(root, "Taskfile.yml"), "version: '3'\ntasks:\n  build:\n    cmds: [echo ok]\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "cmd", "sample", "main.go"), "package main\nfunc main() {}\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "internal", "app", "root.go"), "package app\nfunc run(command string) { switch command { case \"help\": } }\n")

	result, err := (Service{}).AnalyzeWithOptions(root, AnalyzeOptions{Deep: true})
	if err != nil {
		t.Fatal(err)
	}
	categories := map[string]bool{}
	for _, finding := range result.DeepAudit.DriftFindings {
		categories[finding.Category] = true
	}
	if !categories["stale_command"] || !categories["stale_managed_reference"] {
		t.Fatalf("expected package/task/path drift, got %+v", result.DeepAudit.DriftFindings)
	}
	if !hasFindingEvidence(result.DeepAudit.DriftFindings, "documented CLI command is absent: sample ghost") {
		t.Fatalf("expected deterministic CLI drift, got %+v", result.DeepAudit.DriftFindings)
	}
}

func TestAnalyzeDeepIgnoresHistoricalCommandsAndNonLiteralPaths(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "README.md"), "# Current contract\nInspect `rules/*.yaml` and `rules/security-<lang>.yaml`. Force cleanup removes legacy `.agents/specs/spec.yml`.\n\n## Breaking changes\nRemoval of `sample old-command`.\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "cmd", "sample", "main.go"), "package main\nfunc main() {}\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "internal", "app", "root.go"), "package app\nfunc run(command string) { switch command { case \"help\": } }\n")

	result, err := (Service{}).AnalyzeWithOptions(root, AnalyzeOptions{Deep: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.DeepAudit.DriftFindings) != 0 {
		t.Fatalf("historical commands and path patterns are not live stale references: %+v", result.DeepAudit.DriftFindings)
	}
}

func TestAnalyzeDeepDoesNotParseCodeSyntaxAsMarkdownLink(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "rules", "security-swift.yaml"), "rules:\n  - id: TEST-1\n    examples:\n      do: |\n        var bytes = [UInt8](repeating: 0, count: 32)\n")

	result, err := (Service{}).AnalyzeWithOptions(root, AnalyzeOptions{Deep: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.DeepAudit.DriftFindings) != 0 {
		t.Fatalf("non-Markdown code syntax must not be parsed as a link: %+v", result.DeepAudit.DriftFindings)
	}
}

func TestOrientationTruncationDoesNotInvalidatePromptConsistencyClaim(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "AGENTS.md"), "# Policy\nApproval is required. Never perform destructive work without rollback.\n")
	for index := 0; index <= maxDocumentsPerDir; index++ {
		mustWriteAnalyzeFile(t, filepath.Join(root, "docs", fmt.Sprintf("doc-%02d.md", index)), "# Reference\n")
	}

	result, err := (Service{}).AnalyzeWithOptions(root, AnalyzeOptions{Deep: true})
	if err != nil {
		t.Fatal(err)
	}
	claim, ok := findClaim(result.AgentReadiness.Claims, "policy.consistency")
	if !ok || claim.Status != ClaimConfirmed {
		t.Fatalf("orientation-only truncation must not invalidate the prompt audit: %+v", result.AgentReadiness.Claims)
	}
}

func TestOrientationSelectionPrioritizesReadinessRelevantPaths(t *testing.T) {
	paths := make([]string, 0, maxDocumentsPerDir+1)
	for index := 0; index < maxDocumentsPerDir; index++ {
		paths = append(paths, fmt.Sprintf("docs/general-%02d.md", index))
	}
	paths = append(paths, "docs/zz-testing.md")

	selected, notes := selectOrientationPaths(paths)
	if !containsString(selected, "docs/zz-testing.md") {
		t.Fatalf("expected readiness-relevant document to survive directory cap: %v", selected)
	}
	if len(notes) != 1 || notes[0].Reason != "per-directory orientation limit reached" {
		t.Fatalf("expected one explicit directory-cap note, got %+v", notes)
	}
}

func TestEvidenceRedactionCoversCommonCredentialsAndLocalPaths(t *testing.T) {
	input := "Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.payload.signature ghp_abcdefghijklmnopqrstuvwxyz /Users/alice/private alice@example.com https://user:pass@example.com"
	got := excerpt(input)
	for _, sensitive := range []string{"eyJ", "ghp_", "/Users/alice", "alice@example.com", "user:pass@"} {
		if strings.Contains(got, sensitive) {
			t.Fatalf("evidence leaked %q: %s", sensitive, got)
		}
	}
}

func TestAnalyzeDeepBoundsTraversalFindingsAndIgnoresTemplateMirrors(t *testing.T) {
	root := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(root, "AGENTS.md"), "Agents must preserve deterministic verification.\n")
	mustWriteAnalyzeFile(t, filepath.Join(root, "templates", "base", "AGENTS.md"), "Agents must preserve deterministic verification.\n")
	deep := root
	for index := 0; index < maxRepositoryDepth+1; index++ {
		deep = filepath.Join(deep, fmt.Sprintf("d%02d", index))
	}
	mustWriteAnalyzeFile(t, filepath.Join(deep, "ignored.md"), "deep\n")
	for index := 0; index < maxPromptSurfaces; index++ {
		body := "Agents must preserve deterministic verification.\nAgents must preserve deterministic verification.\n"
		mustWriteAnalyzeFile(t, filepath.Join(root, "prompts", fmt.Sprintf("p%03d.md", index)), body)
	}

	result, err := (Service{}).AnalyzeWithOptions(root, AnalyzeOptions{Deep: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.DeepAudit.PromptFindings)+len(result.DeepAudit.DriftFindings) > maxAuditFindings {
		t.Fatalf("finding limit exceeded: %+v", result.DeepAudit)
	}
	if result.DeepAudit.PromptSurfaces != maxPromptSurfaces {
		t.Fatalf("expected template mirror to be excluded before prompt cap, got %d", result.DeepAudit.PromptSurfaces)
	}
	if !hasAuditReason(result.DeepAudit.Truncations, "repository traversal depth limit") || !hasAuditReason(result.DeepAudit.Truncations, "finding limit reached") {
		t.Fatalf("expected explicit traversal and finding truncations, got %+v", result.DeepAudit.Truncations)
	}
	if !containsWarning(result.Warnings, "repository traversal depth limit") {
		t.Fatalf("expected traversal warning in standard result, got %+v", result.Warnings)
	}
	testsClaim, ok := findClaim(result.AgentReadiness.Claims, "verification.tests")
	if !ok || testsClaim.Status != ClaimNotChecked {
		t.Fatalf("expected traversal truncation to produce a not_checked claim, got %+v", result.AgentReadiness.Claims)
	}
}

func containsWarning(warnings []string, want string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, want) {
			return true
		}
	}
	return false
}

func TestFindingCapRetainsHighSeverityPromptConflicts(t *testing.T) {
	audit := DeepAudit{DriftFindings: []Finding{}, PromptFindings: []Finding{}}
	for index := 0; index < maxAuditFindings; index++ {
		audit.DriftFindings = append(audit.DriftFindings, newFinding("PA-008", "stale_reference", "medium", ConfidenceHigh, "README.md", index+1, fmt.Sprintf("missing-%d", index), "stale", "fix"))
	}
	audit.PromptFindings = append(audit.PromptFindings, newFinding("PA-002", "prompt_conflict", "high", ConfidenceHigh, "rules/a.yaml", 1, "conflict", "conflict", "fix"))

	capAuditFindings(&audit)
	if len(audit.DriftFindings)+len(audit.PromptFindings) != maxAuditFindings {
		t.Fatalf("unexpected capped count: drift=%d prompt=%d", len(audit.DriftFindings), len(audit.PromptFindings))
	}
	if len(audit.PromptFindings) != 1 || audit.PromptFindings[0].RuleID != "PA-002" {
		t.Fatalf("high-severity prompt conflict was dropped: %+v", audit.PromptFindings)
	}
	if !hasAuditReason(audit.Truncations, "finding limit reached") {
		t.Fatalf("expected cap truncation note, got %+v", audit.Truncations)
	}
}

func TestAnalyzeJSONUsesEmptyArraysInsteadOfNull(t *testing.T) {
	result, err := (Service{}).AnalyzeWithOptions(t.TempDir(), AnalyzeOptions{Deep: true})
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), ":null") {
		t.Fatalf("public JSON arrays must not serialize as null: %s", body)
	}
}

func TestSafeReadRepositoryFileRejectsSymlinkedParent(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	mustWriteAnalyzeFile(t, filepath.Join(external, "secret.md"), "outside\n")
	if err := os.Symlink(external, filepath.Join(root, "linked")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := safeReadRepositoryFile(root, "linked/secret.md", maxInspectedFileBytes); err == nil {
		t.Fatal("expected symlinked parent to be rejected")
	}
}

func findClaim(claims []Claim, id string) (Claim, bool) {
	for _, claim := range claims {
		if claim.ClaimID == id {
			return claim, true
		}
	}
	return Claim{}, false
}

func hasAuditReason(notes []AuditNote, fragment string) bool {
	for _, note := range notes {
		if strings.Contains(note.Reason, fragment) {
			return true
		}
	}
	return false
}

func hasFindingEvidence(findings []Finding, fragment string) bool {
	for _, finding := range findings {
		if strings.Contains(finding.Evidence, fragment) {
			return true
		}
	}
	return false
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
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

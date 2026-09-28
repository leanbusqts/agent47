package contextmap

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/leanbusqts/agent47/internal/analyze"
)

func TestBuildProducesDeterministicBoundedGraph(t *testing.T) {
	root := t.TempDir()
	mustWriteContextFile(t, root, "go.mod", "module example.com/project\n\ngo 1.22\n")
	mustWriteContextFile(t, root, "Makefile", "test:\n\tgo test ./...\n\nrelease:\n\techo release\n")
	mustWriteContextFile(t, root, "cmd/tool/main.go", "package main\n\nimport \"example.com/project/internal/service\"\n\nfunc main() {}\n")
	mustWriteContextFile(t, root, "internal/service/service.go", "package service\n\nimport \"fmt\"\n")
	mustWriteContextFile(t, root, "internal/service/service_test.go", "package service\n")
	mustWriteContextFile(t, root, "AGENTS.md", "policy body is not copied\n")

	analysis := analyze.AnalysisResult{
		RepoShape:    "single-project",
		ProjectTypes: []analyze.DetectedProjectType{{ID: "cli"}},
		Technologies: []analyze.DetectedTechnology{{ID: "go"}},
	}
	first, err := Build(context.Background(), root, analysis, BuildOptions{PlannedPolicies: []string{"rules/rules-cli.yaml"}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(context.Background(), root, analysis, BuildOptions{PlannedPolicies: []string{"rules/rules-cli.yaml"}})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Content, second.Content) {
		t.Fatal("expected deterministic context output")
	}
	if len(first.Content) > MaxDocumentBytes {
		t.Fatalf("context exceeds bound: %d", len(first.Content))
	}
	for _, fragment := range []string{
		"# Repository Context",
		"`.` — Go module; evidence: `go.mod`",
		"`cmd/tool` -> `internal/service`",
		"`cmd/tool/main.go` — executable; confidence: high",
		"`internal/service` — 1 test files",
		"`make test` — evidence: `Makefile`",
		"`rules/rules-cli.yaml`",
	} {
		if !bytes.Contains(first.Content, []byte(fragment)) {
			t.Fatalf("expected %q in context:\n%s", fragment, first.Content)
		}
	}
	if bytes.Contains(first.Content, []byte("policy body is not copied")) || bytes.Contains(first.Content, []byte("go test ./...")) {
		t.Fatal("context copied repository-controlled bodies")
	}
	metadata, body, err := Parse(first.Content)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.SchemaVersion != SchemaVersion || len(body) == 0 {
		t.Fatalf("unexpected metadata/body: %#v %q", metadata, body)
	}
	action, err := Decide(first.Content, second, false)
	if err != nil || action != ActionCurrent {
		t.Fatalf("expected current decision, action=%q err=%v", action, err)
	}
}

func TestBuildExcludesGeneratedContextAndHostilePaths(t *testing.T) {
	root := t.TempDir()
	mustWriteContextFile(t, root, "go.mod", "module example.com/project\n")
	mustWriteContextFile(t, root, "main.go", "package main\nfunc main() {}\n")
	mustWriteContextFile(t, root, ".agent47/context.md", "self-referential marker\n")
	mustWriteContextFile(t, root, "bad`INJECT.md", "# injected\n")

	document, err := Build(context.Background(), root, analyze.AnalysisResult{RepoShape: "single-project"}, BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(document.Content, []byte("self-referential marker")) || bytes.Contains(document.Content, []byte("bad`INJECT")) {
		t.Fatalf("unsafe or generated content leaked into map:\n%s", document.Content)
	}
	if !bytes.Contains(document.Content, []byte("unsafe or Markdown-active paths omitted")) {
		t.Fatalf("expected bounded omission note:\n%s", document.Content)
	}
}

func TestBuildModelsPostCleanupPolicyState(t *testing.T) {
	root := t.TempDir()
	mustWriteContextFile(t, root, "rules/custom.yaml", "rules: []\n")
	mustWriteContextFile(t, root, "skills/legacy/tool.go", "package legacy\n")
	document, err := Build(context.Background(), root, analyze.AnalysisResult{
		RepoShape:    "single-project",
		Technologies: []analyze.DetectedTechnology{{ID: "go", Confidence: analyze.ConfidenceHigh, Evidence: []string{".go files"}}},
	}, BuildOptions{
		PlannedPolicies: []string{"AGENTS.md", "rules/security-global.yaml"},
		ExcludedPaths:   []string{"rules", "skills"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(document.Content, []byte("rules/custom.yaml")) || bytes.Contains(document.Content, []byte("skills/legacy")) {
		t.Fatalf("cleanup targets leaked into post-init map:\n%s", document.Content)
	}
	if !bytes.Contains(document.Content, []byte("rules/security-global.yaml")) {
		t.Fatalf("planned policy missing from map:\n%s", document.Content)
	}
	if !bytes.Contains(document.Content, []byte("- Technologies: none detected")) {
		t.Fatalf("technology from removed legacy content leaked into map:\n%s", document.Content)
	}
}

func TestDecideProtectsManualAndUnknownContent(t *testing.T) {
	root := t.TempDir()
	mustWriteContextFile(t, root, "go.mod", "module example.com/project\n")
	document, err := Build(context.Background(), root, analyze.AnalysisResult{RepoShape: "single-project"}, BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if action, err := Decide(nil, document, false); err != nil || action != ActionCreate {
		t.Fatalf("expected create, action=%q err=%v", action, err)
	}
	if _, err := Decide([]byte("user file\n"), document, false); !errors.Is(err, ErrUnknownContext) {
		t.Fatalf("expected unknown context error, got %v", err)
	}
	if action, err := Decide([]byte("user file\n"), document, true); err != nil || action != ActionUpdate {
		t.Fatalf("expected forced update, action=%q err=%v", action, err)
	}
	future := bytes.Replace(document.Content, []byte(`"schema_version":1`), []byte(`"schema_version":2`), 1)
	if _, err := Decide(future, document, false); !errors.Is(err, ErrUnknownContext) {
		t.Fatalf("expected future schema protection, got %v", err)
	}

	modified := append([]byte(nil), document.Content...)
	modified = append(modified, []byte("manual edit\n")...)
	if _, err := Decide(modified, document, false); !errors.Is(err, ErrModifiedContext) {
		t.Fatalf("expected modified context error, got %v", err)
	}
	if action, err := Decide(modified, document, true); err != nil || action != ActionUpdate {
		t.Fatalf("expected forced update, action=%q err=%v", action, err)
	}
}

func TestBuildHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Build(ctx, t.TempDir(), analyze.AnalysisResult{}, BuildOptions{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func mustWriteContextFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

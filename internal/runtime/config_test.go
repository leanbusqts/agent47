package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectConfigUsesExplicitRepoRootAndHome(t *testing.T) {
	repoRoot := t.TempDir()
	mustWriteFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), "manifest\n")
	mustWriteFile(t, filepath.Join(repoRoot, "AGENTS.md"), "agents\n")
	mustWriteFile(t, filepath.Join(repoRoot, "VERSION"), "vtest\n")

	agent47Home := filepath.Join(t.TempDir(), ".agent47")
	t.Setenv("AGENT47_REPO_ROOT", repoRoot)
	t.Setenv("AGENT47_HOME", agent47Home)
	t.Setenv("AGENT47_TEMPLATE_SOURCE", "")

	cfg, err := DetectConfig(filepath.Join(repoRoot, "bin", "afs"))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.RepoRoot != repoRoot {
		t.Fatalf("expected repo root %s, got %s", repoRoot, cfg.RepoRoot)
	}
	if cfg.Agent47Home != agent47Home {
		t.Fatalf("expected agent47 home %s, got %s", agent47Home, cfg.Agent47Home)
	}
	if cfg.TemplateMode != TemplateModeFilesystem {
		t.Fatalf("expected filesystem mode, got %s", cfg.TemplateMode)
	}
	if cfg.Version != "vtest" {
		t.Fatalf("expected version vtest, got %s", cfg.Version)
	}
}

func TestDetectTemplateModeHonorsExplicitOverride(t *testing.T) {
	t.Setenv("AGENT47_TEMPLATE_SOURCE", string(TemplateModeEmbedded))
	if mode := detectTemplateMode("/repo"); mode != TemplateModeEmbedded {
		t.Fatalf("expected embedded mode, got %s", mode)
	}

	t.Setenv("AGENT47_TEMPLATE_SOURCE", string(TemplateModeFilesystem))
	if mode := detectTemplateMode(""); mode != TemplateModeFilesystem {
		t.Fatalf("expected filesystem mode, got %s", mode)
	}
}

func TestLooksLikeRepoRootRequiresTemplatesAndAgents(t *testing.T) {
	root := t.TempDir()
	if looksLikeRepoRoot(root) {
		t.Fatal("did not expect empty temp dir to look like repo root")
	}

	mustWriteFile(t, filepath.Join(root, "templates", "manifest.txt"), "manifest\n")
	if looksLikeRepoRoot(root) {
		t.Fatal("did not expect repo root without AGENTS.md to pass")
	}

	mustWriteFile(t, filepath.Join(root, "AGENTS.md"), "agents\n")
	if !looksLikeRepoRoot(root) {
		t.Fatal("expected repo root with templates and AGENTS.md to pass")
	}
}

func mustWriteFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

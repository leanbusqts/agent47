package templates

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leanbusqts/agent47/internal/runtime"
)

func TestNewLoaderRequiresRepoRootInFilesystemMode(t *testing.T) {
	_, err := NewLoader(runtime.TemplateModeFilesystem, "")
	if err == nil {
		t.Fatal("expected filesystem loader error")
	}
}

func TestNewLoaderRejectsUnknownMode(t *testing.T) {
	_, err := NewLoader(runtime.TemplateMode("mystery"), "")
	if err == nil {
		t.Fatal("expected unknown mode error")
	}
	if !strings.Contains(err.Error(), "unknown template mode") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewLoaderFilesystemUsesBasePlusBundleCatalogWhenPresent(t *testing.T) {
	repoRoot := t.TempDir()
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), "root-manifest\n")
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "base", "manifest.txt"), "base-manifest\n")
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "base", "prompts", "agent-prompt.txt"), "base-prompt\n")
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "bundles", "project-cli", "rules", "rules-cli.yaml"), "bundle-rule\n")
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "bundles", "project-cli", "skills", "cli-design", "SKILL.md"), "bundle-skill\n")
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "skills", "legacy-skill", "SKILL.md"), "legacy-skill\n")

	loader, err := NewLoader(runtime.TemplateModeFilesystem, repoRoot)
	if err != nil {
		t.Fatal(err)
	}

	data, err := loader.Source.ReadFile("manifest.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte("base-manifest\n")) {
		t.Fatalf("expected base manifest, got %q", string(data))
	}

	skill, err := loader.Source.ReadFile("skills/cli-design/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(skill, []byte("bundle-skill\n")) {
		t.Fatalf("expected bundle skill through catalog source, got %q", string(skill))
	}

	rule, err := loader.Source.ReadFile("rules/rules-cli.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rule, []byte("bundle-rule\n")) {
		t.Fatalf("expected bundle rule through catalog source, got %q", string(rule))
	}

	if _, err := loader.Source.ReadFile("skills/legacy-skill/SKILL.md"); err == nil {
		t.Fatal("expected catalog source to reject legacy flat skill fallback")
	}

	entries, err := loader.Source.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == "base" || entry.Name() == "bundles" {
			t.Fatal("did not expect internal layout directory in merged root")
		}
	}
}

func TestIsNotExistRecognizesWrappedErrNotExist(t *testing.T) {
	err := errors.Join(fs.ErrNotExist, errors.New("wrapped"))
	if !isNotExist(err) {
		t.Fatal("expected wrapped fs.ErrNotExist to be recognized")
	}
	if isNotExist(errors.New("different")) {
		t.Fatal("did not expect unrelated error to be recognized")
	}
}

func TestBundleSourceAssemblesBaseAndBundleContentWithoutLegacyCopies(t *testing.T) {
	repoRoot := t.TempDir()
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), "root-manifest\n")
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "base", "manifest.txt"), "base-manifest\n")
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "base", "AGENTS.md"), "base-agents\n")
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "base", "skills", "analyze", "SKILL.md"), "base-skill\n")
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "bundles", "project-cli", "rules", "rules-cli.yaml"), "bundle-rule\n")
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "bundles", "project-cli", "skills", "cli-design", "SKILL.md"), "bundle-skill\n")

	loader, err := NewLoader(runtime.TemplateModeFilesystem, repoRoot)
	if err != nil {
		t.Fatal(err)
	}

	source := loader.BundleSource([]string{"base", "project-cli"})

	agents, err := source.ReadFile("AGENTS.md")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(agents, []byte("base-agents\n")) {
		t.Fatalf("expected base AGENTS content, got %q", string(agents))
	}

	rule, err := source.ReadFile("rules/rules-cli.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rule, []byte("bundle-rule\n")) {
		t.Fatalf("expected bundle rule content, got %q", string(rule))
	}

	skill, err := source.ReadFile("skills/cli-design/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(skill, []byte("bundle-skill\n")) {
		t.Fatalf("expected bundle skill content, got %q", string(skill))
	}

	entries, err := source.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == "base" || entry.Name() == "bundles" || entry.Name() == "manifest.txt" {
			t.Fatalf("did not expect internal layout directory %q in assembled root", entry.Name())
		}
	}
}

func TestBundleSourceDoesNotFallBackToLegacyFlatTree(t *testing.T) {
	repoRoot := t.TempDir()
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), "root-manifest\n")
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "base", "manifest.txt"), "base-manifest\n")
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "base", "AGENTS.md"), "base-agents\n")
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "rules", "legacy-only.yaml"), "legacy-only\n")

	loader, err := NewLoader(runtime.TemplateModeFilesystem, repoRoot)
	if err != nil {
		t.Fatal(err)
	}

	source := loader.BundleSource([]string{"base"})
	if _, err := source.ReadFile("rules/legacy-only.yaml"); err == nil {
		t.Fatal("expected bundle source to reject legacy-only payload fallback")
	}
}

func mustWriteTemplateFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

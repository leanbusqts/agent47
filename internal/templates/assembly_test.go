package templates

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leanbusqts/agent47/internal/runtime"
)

func TestAssembleManifestIncludesBundleRuleTemplates(t *testing.T) {
	loader, err := NewLoader(runtime.TemplateModeFilesystem, repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}

	got, err := AssembleManifest(loader.RawSource, []string{"base", "project-cli", "project-scripts", "shared-cli-behavior", "shared-testing"})
	if err != nil {
		t.Fatal(err)
	}

	if !got.ContainsRuleTemplate("rules-cli.yaml") {
		t.Fatalf("expected project cli rule template, got %v", got.RuleTemplates)
	}
	if !got.ContainsRuleTemplate("rules-scripts.yaml") {
		t.Fatalf("expected project scripts rule template, got %v", got.RuleTemplates)
	}
	if !got.ContainsRuleTemplate("shared-cli-behavior.yaml") {
		t.Fatalf("expected shared cli behavior rule template, got %v", got.RuleTemplates)
	}
	if !got.ContainsRuleTemplate("shared-testing.yaml") {
		t.Fatalf("expected shared testing rule template, got %v", got.RuleTemplates)
	}
}

func TestAssembleManifestAcceptsPartialBundleManifests(t *testing.T) {
	repoRoot := t.TempDir()
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), validAssemblyManifest())
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "base", "manifest.txt"), validAssemblyManifest())
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "bundles", "project-cli", "manifest.txt"), partialCliBundleManifest())
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "bundles", "shared-cli-behavior", "manifest.txt"), partialSharedBundleManifest("shared-cli-behavior"))
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "bundles", "shared-testing", "manifest.txt"), partialSharedBundleManifest("shared-testing"))
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "bundles", "project-cli", "rules", "rules-cli.yaml"), "rule\n")
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "bundles", "project-cli", "skills", "cli-design", "SKILL.md"), "skill\n")
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "bundles", "shared-cli-behavior", "rules", "shared-cli-behavior.yaml"), "shared-cli\n")
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "bundles", "shared-testing", "rules", "shared-testing.yaml"), "shared-testing\n")

	loader, err := NewLoader(runtime.TemplateModeFilesystem, repoRoot)
	if err != nil {
		t.Fatal(err)
	}

	got, err := AssembleManifest(loader.RawSource, []string{"base", "project-cli", "shared-cli-behavior", "shared-testing"})
	if err != nil {
		t.Fatal(err)
	}

	if !got.ContainsRuleTemplate("rules-cli.yaml") {
		t.Fatalf("expected project cli rule template, got %v", got.RuleTemplates)
	}
	if !got.ContainsRuleTemplate("shared-cli-behavior.yaml") || !got.ContainsRuleTemplate("shared-testing.yaml") {
		t.Fatalf("expected shared rule templates, got %v", got.RuleTemplates)
	}
	if len(got.ManagedTargets) != 4 {
		t.Fatalf("expected managed targets to be inherited from base, got %v", got.ManagedTargets)
	}
}

func TestAssembleManifestFailsWhenBundleManifestMissing(t *testing.T) {
	repoRoot := t.TempDir()
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), validAssemblyManifest())

	loader, err := NewLoader(runtime.TemplateModeFilesystem, repoRoot)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := AssembleManifest(loader.RawSource, []string{"base", "project-cli"}); err == nil {
		t.Fatal("expected missing bundle manifest error")
	} else {
		var missing MissingBundleManifestError
		if !errors.As(err, &missing) {
			t.Fatalf("expected missing bundle manifest error, got %v", err)
		}
	}
}

func TestAssembleManifestRejectsEmptyBundleManifest(t *testing.T) {
	repoRoot := t.TempDir()
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), validAssemblyManifest())
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "base", "manifest.txt"), validAssemblyManifest())
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "bundles", "project-cli", "manifest.txt"), "")

	loader, err := NewLoader(runtime.TemplateModeFilesystem, repoRoot)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := AssembleManifest(loader.RawSource, []string{"base", "project-cli"}); err == nil {
		t.Fatal("expected empty bundle manifest error")
	} else {
		var invalid InvalidBundleManifestError
		if !errors.As(err, &invalid) {
			t.Fatalf("expected invalid bundle manifest error, got %v", err)
		}
	}
}

func TestValidateAssemblyAllowsIdenticalDuplicateContent(t *testing.T) {
	repoRoot := t.TempDir()
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), validAssemblyManifest())
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "base", "manifest.txt"), validAssemblyManifest())
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "base", "rules", "shared.yaml"), "same\n")
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "bundles", "project-cli", "rules", "shared.yaml"), "same\n")

	loader, err := NewLoader(runtime.TemplateModeFilesystem, repoRoot)
	if err != nil {
		t.Fatal(err)
	}

	if err := ValidateAssembly(loader.RawSource, []string{"base", "project-cli"}); err != nil {
		t.Fatalf("expected duplicate identical content to be allowed, got %v", err)
	}
}

func TestValidateAssemblyFailsForConflictingBundleContent(t *testing.T) {
	repoRoot := t.TempDir()
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), validAssemblyManifest())
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "base", "manifest.txt"), validAssemblyManifest())
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "base", "rules", "shared.yaml"), "base\n")
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "bundles", "project-cli", "rules", "shared.yaml"), "cli\n")

	loader, err := NewLoader(runtime.TemplateModeFilesystem, repoRoot)
	if err != nil {
		t.Fatal(err)
	}

	err = ValidateAssembly(loader.RawSource, []string{"base", "project-cli"})
	if err == nil {
		t.Fatal("expected conflicting content error")
	}
	if !strings.Contains(err.Error(), "assembly conflict for rules/shared.yaml") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAssemblyFailsForFileDirectoryConflict(t *testing.T) {
	repoRoot := t.TempDir()
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), validAssemblyManifest())
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "base", "manifest.txt"), validAssemblyManifest())
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "base", "rules", "shared.yaml"), "base\n")
	mustWriteAssemblyFile(t, filepath.Join(repoRoot, "templates", "bundles", "project-cli", "rules", "shared.yaml", "nested.txt"), "nested\n")

	loader, err := NewLoader(runtime.TemplateModeFilesystem, repoRoot)
	if err != nil {
		t.Fatal(err)
	}

	err = ValidateAssembly(loader.RawSource, []string{"base", "project-cli"})
	if err == nil {
		t.Fatal("expected file-directory conflict error")
	}
	if !strings.Contains(err.Error(), "assembly conflict for rules/shared.yaml") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func mustWriteAssemblyFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func validAssemblyManifest() string {
	return `[rule_templates]
security-global.yaml

[managed_targets]
AGENTS.md
rules/*.yaml
skills/*
skills/AVAILABLE_SKILLS.xml

[preserved_targets]
README.md
specs/spec.yml
SNAPSHOT.md
SPEC.md

[required_template_files]
AGENTS.md
manifest.txt
specs/spec.yml

[required_template_dirs]
rules
skills
specs
`
}

func partialCliBundleManifest() string {
	return `[rule_templates]
rules-cli.yaml

[required_template_files]
rules/rules-cli.yaml
skills/cli-design/SKILL.md
`
}

func partialSharedBundleManifest(rule string) string {
	return `[rule_templates]
` + rule + `.yaml

[required_template_files]
rules/` + rule + `.yaml
`
}

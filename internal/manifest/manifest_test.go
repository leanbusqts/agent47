package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseManifestFromTemplates(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "templates", "manifest.txt"))
	if err != nil {
		t.Fatal(err)
	}

	got, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	if !got.ContainsRuleTemplate("security-shell.yaml") {
		t.Fatalf("expected security-shell.yaml in rule templates: %#v", got.RuleTemplates)
	}
	if len(got.ManagedTargets) != 4 {
		t.Fatalf("expected 4 managed targets, got %d", len(got.ManagedTargets))
	}
}

func TestValidateRejectsMissingSections(t *testing.T) {
	t.Parallel()

	err := (Manifest{}).Validate()
	if err == nil {
		t.Fatal("expected validation error")
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

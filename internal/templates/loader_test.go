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
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "base", "NOTICE.md"), "base-notice\n")
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "bundles", "project-cli", "rules", "rules-cli.yaml"), "bundle-rule\n")
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "bundles", "project-cli", "docs", "cli.md"), "bundle-doc\n")
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "docs", "legacy.md"), "legacy-doc\n")

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

	doc, err := loader.Source.ReadFile("docs/cli.md")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(doc, []byte("bundle-doc\n")) {
		t.Fatalf("expected bundle document through catalog source, got %q", string(doc))
	}

	rule, err := loader.Source.ReadFile("rules/rules-cli.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rule, []byte("bundle-rule\n")) {
		t.Fatalf("expected bundle rule through catalog source, got %q", string(rule))
	}

	if _, err := loader.Source.ReadFile("docs/legacy.md"); err == nil {
		t.Fatal("expected catalog source to reject legacy flat-tree fallback")
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
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "base", "NOTICE.md"), "base-notice\n")
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "bundles", "project-cli", "rules", "rules-cli.yaml"), "bundle-rule\n")
	mustWriteTemplateFile(t, filepath.Join(repoRoot, "templates", "bundles", "project-cli", "docs", "cli.md"), "bundle-doc\n")

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

	doc, err := source.ReadFile("docs/cli.md")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(doc, []byte("bundle-doc\n")) {
		t.Fatalf("expected bundle document content, got %q", string(doc))
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

func TestRootFilteredSourceHidesOnlyNamedRootEntries(t *testing.T) {
	root := t.TempDir()
	mustWriteTemplateFile(t, filepath.Join(root, "visible.txt"), "visible\n")
	mustWriteTemplateFile(t, filepath.Join(root, "hidden", "nested.txt"), "nested\n")
	mustWriteTemplateFile(t, filepath.Join(root, "nested", "hidden"), "not hidden below root\n")
	source := NewRootFilteredSource(NewFilesystemSource(root), "hidden")

	if _, err := source.ReadFile("hidden"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected hidden root file to be absent, got %v", err)
	}
	if _, err := source.Stat("hidden"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected hidden root entry to be absent, got %v", err)
	}
	data, err := source.ReadFile("nested/hidden")
	if err != nil || string(data) != "not hidden below root\n" {
		t.Fatalf("nested entry must remain visible: %q, %v", data, err)
	}
	entries, err := source.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == "hidden" {
			t.Fatal("hidden root entry leaked through ReadDir")
		}
	}
	nested, err := source.ReadDir("nested")
	if err != nil || len(nested) != 1 || nested[0].Name() != "hidden" {
		t.Fatalf("nested directory must be delegated unchanged: %+v, %v", nested, err)
	}
}

func TestTemplateErrorsDescribePathAndOptionalDetail(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{MissingTemplateError{Path: "AGENTS.md"}, "template not found: AGENTS.md"},
		{MissingBundleManifestError{Path: "bundles/project-cli/manifest.txt"}, "missing bundle manifest: bundles/project-cli/manifest.txt"},
		{InvalidBundleManifestError{Path: "manifest.txt"}, "invalid bundle manifest: manifest.txt"},
		{InvalidBundleManifestError{Path: "manifest.txt", Detail: "bad section"}, "invalid bundle manifest manifest.txt: bad section"},
		{AssemblyConflictError{Path: "rules/a.yaml"}, "assembly conflict for rules/a.yaml"},
		{AssemblyConflictError{Path: "rules/a.yaml", Detail: "different owners"}, "assembly conflict for rules/a.yaml: different owners"},
	}
	for _, test := range cases {
		if got := test.err.Error(); got != test.want {
			t.Fatalf("unexpected error text: got %q want %q", got, test.want)
		}
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

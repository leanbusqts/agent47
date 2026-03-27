package templates

import (
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

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

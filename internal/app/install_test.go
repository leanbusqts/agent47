package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func TestRunInstallInternalRejectsUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))

	status := root.Run(context.Background(), runtime.Config{Version: "vtest"}, []string{internalInstallCommand, "--bogus"})
	if status == 0 {
		t.Fatal("expected non-zero status")
	}
	if !strings.Contains(stderr.String(), "Unknown internal install flag") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunInstallInternalReportsInstallInitializationFailure(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))

	status := root.Run(context.Background(), runtime.Config{
		Version:      "vtest",
		TemplateMode: runtime.TemplateModeFilesystem,
	}, []string{internalInstallCommand})
	if status == 0 {
		t.Fatal("expected non-zero status")
	}
	if !strings.Contains(stderr.String(), "Failed to initialize install service") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunInstallInternalSucceedsWithEmbeddedTemplates(t *testing.T) {
	baseDir := t.TempDir()
	homeDir := filepath.Join(baseDir, "home")
	agentHome := filepath.Join(homeDir, ".agent47")
	userBinDir := filepath.Join(homeDir, "bin")
	if err := os.MkdirAll(userBinDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))
	status := root.Run(context.Background(), runtime.Config{
		OS:             "darwin",
		Version:        "vtest",
		TemplateMode:   runtime.TemplateModeEmbedded,
		ExecutablePath: os.Args[0],
		HomeDir:        homeDir,
		UserBinDir:     userBinDir,
		Agent47Home:    agentHome,
	}, []string{internalInstallCommand, "--non-interactive"})
	if status != 0 {
		t.Fatalf("expected success, got %d stderr=%s", status, stderr.String())
	}
}

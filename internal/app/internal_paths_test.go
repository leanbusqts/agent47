package app

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func TestRunAddAgentPromptRejectsUnexpectedArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))

	status := root.Run(context.Background(), runtime.Config{Version: "vtest"}, []string{"add-agent-prompt", "unexpected"})
	if status == 0 {
		t.Fatal("expected non-zero status")
	}
	if !strings.Contains(stdout.String(), "Usage: add-agent-prompt [--force]") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestRunAddSSPromptPrintsPromptWithoutClipboardTool(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))

	status := root.Run(context.Background(), runtime.Config{
		Version:      "vtest",
		TemplateMode: runtime.TemplateModeEmbedded,
	}, []string{"add-ss-prompt"})
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr.String())
	}
	if !strings.Contains(stdout.String(), "SNAPSHOT") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestRunUninstallRejectsUnexpectedArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))

	status := root.Run(context.Background(), runtime.Config{Version: "vtest"}, []string{"uninstall", "unexpected"})
	if status == 0 {
		t.Fatal("expected non-zero status")
	}
	if !strings.Contains(stdout.String(), "Unknown command: uninstall unexpected") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestRunMapsAddSSPromptExecutableName(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))

	status := root.Run(context.Background(), runtime.Config{
		Version:        "vtest",
		TemplateMode:   runtime.TemplateModeEmbedded,
		ExecutablePath: "add-ss-prompt",
	}, nil)
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr.String())
	}
	if !strings.Contains(stdout.String(), "SNAPSHOT") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

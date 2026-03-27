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

func TestRunAddAgentPromptCreatesFile(t *testing.T) {
	baseDir := t.TempDir()
	repoRoot := filepath.Join(baseDir, "repo")
	workDir := filepath.Join(baseDir, "work")
	copyDirFromRepo(t, filepath.Join(repoRoot, "templates"), "templates")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))

	wd, _ := os.Getwd()
	_ = os.Chdir(workDir)
	defer func() { _ = os.Chdir(wd) }()

	status := root.Run(context.Background(), runtime.Config{
		Version:      "vtest",
		TemplateMode: runtime.TemplateModeFilesystem,
		RepoRoot:     repoRoot,
	}, []string{"add-agent-prompt"})
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr.String())
	}

	data, err := os.ReadFile(filepath.Join(workDir, "prompts", "agent-prompt.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Use `AGENTS.md` as the single source of policy.") {
		t.Fatalf("unexpected prompt content: %s", string(data))
	}
}

func TestRunAddSSPromptRejectsUnexpectedArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))

	status := root.Run(context.Background(), runtime.Config{Version: "vtest"}, []string{"add-ss-prompt", "unexpected"})
	if status == 0 {
		t.Fatal("expected non-zero status")
	}
	if !strings.Contains(stdout.String(), "Usage: add-ss-prompt") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

package prompts

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/leanbusqts/agent47/internal/cli"
	runtimecfg "github.com/leanbusqts/agent47/internal/runtime"
)

func TestAddSSPromptUsesSupportedClipboardTool(t *testing.T) {
	toolName, toolBody := clipboardTestTool()

	toolDir := t.TempDir()
	toolPath := filepath.Join(toolDir, toolName)
	if runtime.GOOS == "windows" {
		toolPath += ".cmd"
	}
	if err := os.WriteFile(toolPath, []byte(toolBody), 0o755); err != nil {
		t.Fatal(err)
	}

	outputFile := filepath.Join(t.TempDir(), "clipboard.txt")
	t.Setenv("TARGET_FILE", outputFile)
	t.Setenv("PATH", toolDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	service, err := New(runtimecfg.Config{TemplateMode: runtimecfg.TemplateModeEmbedded}, cli.NewOutput(&stdout, &stderr))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.AddSSPrompt(); err != nil {
		t.Fatalf("AddSSPrompt failed: %v", err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "SNAPSHOT") {
		t.Fatalf("unexpected clipboard content: %s", string(data))
	}
	if !strings.Contains(stdout.String(), "copied to clipboard") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestAddAgentPromptCreatesPromptFile(t *testing.T) {
	workDir := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	service, err := New(runtimecfg.Config{TemplateMode: runtimecfg.TemplateModeEmbedded}, cli.NewOutput(&stdout, &stderr))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.AddAgentPrompt(workDir, false); err != nil {
		t.Fatalf("AddAgentPrompt failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(workDir, "prompts", "agent-prompt.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Use `AGENTS.md` as the single source of policy.") {
		t.Fatalf("unexpected prompt content: %s", string(data))
	}
}

func TestAddAgentPromptSkipsExistingFileWithoutForce(t *testing.T) {
	workDir := t.TempDir()
	target := filepath.Join(workDir, "prompts", "agent-prompt.txt")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("existing prompt\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	service, err := New(runtimecfg.Config{TemplateMode: runtimecfg.TemplateModeEmbedded}, cli.NewOutput(&stdout, &stderr))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.AddAgentPrompt(workDir, false); err != nil {
		t.Fatalf("AddAgentPrompt failed: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "existing prompt\n" {
		t.Fatalf("expected prompt to remain unchanged, got %s", string(data))
	}
}

func TestAddSSPromptFallsBackToStdoutWithoutClipboardTool(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	service, err := New(runtimecfg.Config{TemplateMode: runtimecfg.TemplateModeEmbedded}, cli.NewOutput(&stdout, &stderr))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.AddSSPrompt(); err != nil {
		t.Fatalf("AddSSPrompt failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "SNAPSHOT") {
		t.Fatalf("expected prompt output, got %s", stdout.String())
	}
}

func TestAddSSPromptUsesXclipWhenEarlierToolsMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("xclip is unix-only")
	}

	toolDir := t.TempDir()
	writeClipboardTool(t, filepath.Join(toolDir, "xclip"), "#!/bin/sh\necho xclip >> \"$TARGET_FILE\"\ncat >> \"$TARGET_FILE\"\n")

	outputFile := filepath.Join(t.TempDir(), "clipboard.txt")
	t.Setenv("TARGET_FILE", outputFile)
	t.Setenv("PATH", toolDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	service, err := New(runtimecfg.Config{TemplateMode: runtimecfg.TemplateModeEmbedded}, cli.NewOutput(&stdout, &stderr))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.AddSSPrompt(); err != nil {
		t.Fatalf("AddSSPrompt failed: %v", err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "xclip\n") {
		t.Fatalf("expected xclip marker, got %s", string(data))
	}
}

func TestAddSSPromptPrefersPbcopyOverLaterClipboardTools(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("clipboard precedence differs on windows")
	}

	toolDir := t.TempDir()
	writeClipboardTool(t, filepath.Join(toolDir, "pbcopy"), "#!/bin/sh\necho pbcopy >> \"$TARGET_FILE\"\ncat >> \"$TARGET_FILE\"\n")
	writeClipboardTool(t, filepath.Join(toolDir, "xclip"), "#!/bin/sh\necho xclip >> \"$TARGET_FILE\"\ncat >> \"$TARGET_FILE\"\n")

	outputFile := filepath.Join(t.TempDir(), "clipboard.txt")
	t.Setenv("TARGET_FILE", outputFile)
	t.Setenv("PATH", toolDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	service, err := New(runtimecfg.Config{TemplateMode: runtimecfg.TemplateModeEmbedded}, cli.NewOutput(&stdout, &stderr))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.AddSSPrompt(); err != nil {
		t.Fatalf("AddSSPrompt failed: %v", err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "pbcopy\n") {
		t.Fatalf("expected pbcopy marker, got %s", string(data))
	}
}

func clipboardTestTool() (string, string) {
	if runtime.GOOS == "windows" {
		return "clip", "@echo off\r\nmore > \"%TARGET_FILE%\"\r\n"
	}

	return "wl-copy", "#!/bin/sh\ncat > \"$TARGET_FILE\"\n"
}

func writeClipboardTool(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

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

func TestRunDoctorReportsMissingAfsInPath(t *testing.T) {
	baseDir := t.TempDir()
	repoRoot := filepath.Join(baseDir, "repo")
	copyDirFromRepo(t, filepath.Join(repoRoot, "templates"), "templates")

	homeDir := filepath.Join(baseDir, "home")
	if err := os.MkdirAll(filepath.Join(homeDir, ".agent47", "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyDir(filepath.Join(repoRoot, "templates"), filepath.Join(homeDir, ".agent47", "templates")); err != nil {
		t.Fatal(err)
	}

	oldPath := os.Getenv("PATH")
	oldVersionURL := os.Getenv("AGENT47_VERSION_URL")
	t.Setenv("PATH", "/usr/bin:/bin")
	t.Setenv("AGENT47_VERSION_URL", oldVersionURL)
	defer os.Setenv("PATH", oldPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))

	status := root.Run(context.Background(), runtime.Config{
		Version:         "vtest",
		TemplateMode:    runtime.TemplateModeFilesystem,
		RepoRoot:        repoRoot,
		HomeDir:         homeDir,
		UserBinDir:      filepath.Join(homeDir, "bin"),
		Agent47Home:     filepath.Join(homeDir, ".agent47"),
		UpdateCacheFile: filepath.Join(homeDir, ".agent47", "cache", "update.cache"),
	}, []string{"doctor"})
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr.String())
	}
	if !strings.Contains(stdout.String(), "afs not in PATH") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Skipping update check by default") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestRunDoctorCheckUpdateUsesRemoteVersion(t *testing.T) {
	baseDir := t.TempDir()
	repoRoot := filepath.Join(baseDir, "repo")
	copyDirFromRepo(t, filepath.Join(repoRoot, "templates"), "templates")

	homeDir := filepath.Join(baseDir, "home")
	templateDir := filepath.Join(homeDir, ".agent47", "templates")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyDir(filepath.Join(repoRoot, "templates"), templateDir); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", "/usr/bin:/bin")
	t.Setenv("AGENT47_VERSION_URL", "file://"+filepath.Join(repoRoot, "VERSION"))
	if err := os.WriteFile(filepath.Join(repoRoot, "VERSION"), []byte("vtest\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))

	status := root.Run(context.Background(), runtime.Config{
		Version:         "vtest",
		TemplateMode:    runtime.TemplateModeFilesystem,
		RepoRoot:        repoRoot,
		HomeDir:         homeDir,
		UserBinDir:      filepath.Join(homeDir, "bin"),
		Agent47Home:     filepath.Join(homeDir, ".agent47"),
		UpdateCacheFile: filepath.Join(homeDir, ".agent47", "cache", "update.cache"),
	}, []string{"doctor", "--check-update"})
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Up to date") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestRunDoctorUsesWindowsInstallHints(t *testing.T) {
	baseDir := t.TempDir()
	repoRoot := filepath.Join(baseDir, "repo")
	copyDirFromRepo(t, filepath.Join(repoRoot, "templates"), "templates")

	homeDir := filepath.Join(baseDir, "home")
	templateDir := filepath.Join(homeDir, "AppData", "Local", "agent47", "templates")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyDir(filepath.Join(repoRoot, "templates"), templateDir); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", `C:\Windows\System32`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))

	status := root.Run(context.Background(), runtime.Config{
		OS:              "windows",
		Version:         "vtest",
		TemplateMode:    runtime.TemplateModeFilesystem,
		RepoRoot:        repoRoot,
		HomeDir:         homeDir,
		UserBinDir:      filepath.Join(homeDir, "AppData", "Local", "agent47", "bin"),
		Agent47Home:     filepath.Join(homeDir, "AppData", "Local", "agent47"),
		UpdateCacheFile: filepath.Join(homeDir, "AppData", "Local", "agent47", "cache", "update.cache"),
	}, []string{"doctor"})
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr.String())
	}

	output := stdout.String()
	if strings.Contains(output, "Fix: run ./install.sh") {
		t.Fatalf("unexpected unix install hint in output: %s", output)
	}
	if !strings.Contains(output, "Fix: rerun install.ps1 or add the managed bin to PATH") {
		t.Fatalf("expected windows install hint in output: %s", output)
	}
}

func TestRunDoctorRejectsUnexpectedFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))

	status := root.Run(context.Background(), runtime.Config{Version: "vtest"}, []string{"doctor", "--bogus"})
	if status == 0 {
		t.Fatal("expected non-zero status")
	}
	if !strings.Contains(stdout.String(), "Usage: afs doctor") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

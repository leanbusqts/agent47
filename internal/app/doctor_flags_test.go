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

func TestRunDoctorAllowsCombinedFlags(t *testing.T) {
	baseDir := t.TempDir()
	repoRoot := filepath.Join(baseDir, "repo")
	copyDirFromRepo(t, filepath.Join(repoRoot, "templates"), "templates")
	if err := os.WriteFile(filepath.Join(repoRoot, "VERSION"), []byte("vtest\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	homeDir := filepath.Join(baseDir, "home")
	templateDir := filepath.Join(homeDir, ".agent47", "templates")
	managedBinDir := filepath.Join(homeDir, ".agent47", "bin")
	userBinDir := filepath.Join(homeDir, "bin")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(managedBinDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(userBinDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyDir(filepath.Join(repoRoot, "templates"), templateDir); err != nil {
		t.Fatal(err)
	}

	managedAfs := filepath.Join(managedBinDir, "afs")
	if err := os.WriteFile(managedAfs, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"afs", "add-agent", "add-agent-prompt", "add-ss-prompt"} {
		if err := os.Symlink(managedAfs, filepath.Join(userBinDir, name)); err != nil {
			t.Fatal(err)
		}
	}

	t.Setenv("PATH", userBinDir)
	t.Setenv("AGENT47_VERSION_URL", "file://"+filepath.Join(repoRoot, "VERSION"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))

	status := root.Run(context.Background(), runtime.Config{
		Version:         "vtest",
		TemplateMode:    runtime.TemplateModeFilesystem,
		RepoRoot:        repoRoot,
		HomeDir:         homeDir,
		UserBinDir:      userBinDir,
		Agent47Home:     filepath.Join(homeDir, ".agent47"),
		UpdateCacheFile: filepath.Join(homeDir, ".agent47", "cache", "update.cache"),
	}, []string{"doctor", "--check-update", "--fail-on-warn"})
	if status != 0 {
		t.Fatalf("expected status 0, got %d: stdout=%s stderr=%s", status, stdout.String(), stderr.String())
	}
	if strings.Contains(stdout.String(), "Usage: afs doctor") {
		t.Fatalf("unexpected usage output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Up to date") {
		t.Fatalf("expected update check output, got %s", stdout.String())
	}
}

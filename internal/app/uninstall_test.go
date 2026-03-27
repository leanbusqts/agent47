package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/install"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func TestRunUninstallRemovesManagedRuntime(t *testing.T) {
	baseDir := t.TempDir()
	repoRoot := repoRoot(t)

	homeDir := filepath.Join(baseDir, "home")
	userBin := filepath.Join(homeDir, "bin")
	agentHome := filepath.Join(homeDir, ".agent47")
	if err := os.MkdirAll(userBin, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := runtime.Config{
		Version:         "vtest",
		TemplateMode:    runtime.TemplateModeFilesystem,
		RepoRoot:        repoRoot,
		HomeDir:         homeDir,
		UserBinDir:      userBin,
		Agent47Home:     agentHome,
		UpdateCacheFile: filepath.Join(agentHome, "cache", "update.cache"),
		ExecutablePath:  os.Args[0],
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	out := cli.NewOutput(&stdout, &stderr)
	installer, err := install.New(cfg, out)
	if err != nil {
		t.Fatal(err)
	}
	if err := installer.Install(context.Background(), cfg, install.InstallOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(agentHome, "cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentHome, "cache", "update.cache"), []byte("cached"), 0o644); err != nil {
		t.Fatal(err)
	}

	root := NewRoot(cli.NewOutput(&stdout, &stderr))
	status := root.Run(context.Background(), cfg, []string{"uninstall"})
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr.String())
	}

	assertNotExists(t, filepath.Join(userBin, "add-agent"))
	assertNotExists(t, filepath.Join(userBin, "add-agent-prompt"))
	assertNotExists(t, filepath.Join(userBin, "add-ss-prompt"))
	assertNotExists(t, filepath.Join(agentHome, "templates"))
	assertNotExists(t, filepath.Join(agentHome, "scripts"))
	assertNotExists(t, filepath.Join(agentHome, "cache"))
}

package runtime

import (
	"path/filepath"
	"testing"

	"github.com/leanbusqts/agent47/internal/platform"
)

func TestDetectConfigFallsBackToEmbeddedOutsideRepo(t *testing.T) {
	t.Setenv("AGENT47_REPO_ROOT", "")
	t.Setenv("AGENT47_HOME", filepath.Join(t.TempDir(), ".agent47"))
	t.Setenv("AGENT47_TEMPLATE_SOURCE", "")

	cfg, err := DetectConfig(filepath.Join(t.TempDir(), "bin", "afs"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TemplateMode != TemplateModeEmbedded {
		t.Fatalf("expected embedded mode, got %s", cfg.TemplateMode)
	}
	if cfg.RepoRoot != "" {
		t.Fatalf("expected empty repo root, got %s", cfg.RepoRoot)
	}
}

func TestDetectRepoRootWalksParents(t *testing.T) {
	repoRoot := t.TempDir()
	mustWriteFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), "manifest\n")
	mustWriteFile(t, filepath.Join(repoRoot, "AGENTS.md"), "agents\n")

	start := filepath.Join(repoRoot, "cmd", "afs", "nested", "afs")
	if got := detectRepoRoot(start); got != repoRoot {
		t.Fatalf("expected repo root %s, got %s", repoRoot, got)
	}
}

func TestDetectConfigUsesRepoRootDetectedFromExecutablePath(t *testing.T) {
	repoRoot := t.TempDir()
	mustWriteFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), "manifest\n")
	mustWriteFile(t, filepath.Join(repoRoot, "AGENTS.md"), "agents\n")
	mustWriteFile(t, filepath.Join(repoRoot, "VERSION"), "vtest\n")
	t.Setenv("AGENT47_REPO_ROOT", "")
	t.Setenv("AGENT47_HOME", filepath.Join(t.TempDir(), ".agent47"))
	t.Setenv("AGENT47_TEMPLATE_SOURCE", "")

	cfg, err := DetectConfig(filepath.Join(repoRoot, "cmd", "afs", "afs"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RepoRoot != repoRoot {
		t.Fatalf("expected repo root %s, got %s", repoRoot, cfg.RepoRoot)
	}
	if cfg.TemplateMode != TemplateModeFilesystem {
		t.Fatalf("expected filesystem mode, got %s", cfg.TemplateMode)
	}
}

func TestDetectConfigUsesAbsoluteExecutablePathAndUserBin(t *testing.T) {
	repoRoot := t.TempDir()
	mustWriteFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), "manifest\n")
	mustWriteFile(t, filepath.Join(repoRoot, "AGENTS.md"), "agents\n")
	mustWriteFile(t, filepath.Join(repoRoot, "VERSION"), "vtest\n")
	t.Setenv("AGENT47_REPO_ROOT", repoRoot)
	t.Setenv("AGENT47_HOME", filepath.Join(t.TempDir(), ".agent47"))

	cfg, err := DetectConfig(filepath.Join(repoRoot, "bin", "..", "bin", "afs"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ExecutablePath != filepath.Join(repoRoot, "bin", "afs") {
		t.Fatalf("expected absolute executable path, got %s", cfg.ExecutablePath)
	}
	if cfg.UserBinDir != filepath.Join(cfg.HomeDir, "bin") {
		t.Fatalf("unexpected user bin dir: %s", cfg.UserBinDir)
	}
}

func TestDetectConfigUsesWindowsPathsWhenPlatformIsWindows(t *testing.T) {
	repoRoot := t.TempDir()
	localAppData := filepath.Join(t.TempDir(), "LocalAppData")
	mustWriteFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), "manifest\n")
	mustWriteFile(t, filepath.Join(repoRoot, "AGENTS.md"), "agents\n")
	mustWriteFile(t, filepath.Join(repoRoot, "VERSION"), "vtest\n")
	t.Setenv("AGENT47_REPO_ROOT", repoRoot)
	t.Setenv("AGENT47_HOME", "")
	t.Setenv("LOCALAPPDATA", localAppData)
	runtimeIsWindows = func() bool { return true }
	runtimeOS = func() string { return "windows" }
	defer func() {
		runtimeIsWindows = platform.IsWindows
		runtimeOS = platform.OS
	}()

	cfg, err := DetectConfig(filepath.Join(repoRoot, "bin", "afs.exe"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Agent47Home != filepath.Join(localAppData, "agent47") {
		t.Fatalf("unexpected agent47 home: %s", cfg.Agent47Home)
	}
	if cfg.UserBinDir != filepath.Join(cfg.Agent47Home, "bin") {
		t.Fatalf("unexpected user bin dir: %s", cfg.UserBinDir)
	}
	if cfg.OS != "windows" {
		t.Fatalf("unexpected os: %s", cfg.OS)
	}
}

func TestDetectRepoRootIgnoresInvalidExplicitRepoRoot(t *testing.T) {
	repoRoot := t.TempDir()
	invalid := t.TempDir()
	mustWriteFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), "manifest\n")
	mustWriteFile(t, filepath.Join(repoRoot, "AGENTS.md"), "agents\n")
	t.Setenv("AGENT47_REPO_ROOT", invalid)

	start := filepath.Join(repoRoot, "bin", "afs")
	if got := detectRepoRoot(start); got != repoRoot {
		t.Fatalf("expected fallback repo root %s, got %s", repoRoot, got)
	}
}

func TestDetectConfigRejectsUnsafeAgent47HomeAtHomeDir(t *testing.T) {
	repoRoot := t.TempDir()
	mustWriteFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), "manifest\n")
	mustWriteFile(t, filepath.Join(repoRoot, "AGENTS.md"), "agents\n")
	t.Setenv("AGENT47_REPO_ROOT", repoRoot)
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("AGENT47_HOME", homeDir)

	if _, err := DetectConfig(filepath.Join(repoRoot, "bin", "afs")); err == nil {
		t.Fatal("expected unsafe AGENT47_HOME error")
	}
}

func TestDetectConfigRejectsManagedBinCollisionOnUnix(t *testing.T) {
	homeDir := t.TempDir()
	agentHome := filepath.Join(homeDir, ".agent47-home")
	userBinDir := filepath.Join(homeDir, "bin")
	if _, err := validateAgent47Home(homeDir, "", homeDir, userBinDir, false); err == nil {
		t.Fatal("expected home collision rejection")
	}
	if _, err := validateAgent47Home(homeDir, "", userBinDir, userBinDir, false); err == nil {
		t.Fatal("expected user bin collision rejection")
	}
	if _, err := validateAgent47Home(homeDir, "", agentHome, filepath.Join(agentHome, "bin"), false); err == nil {
		t.Fatal("expected managed/published bin collision rejection")
	}
}

func TestDetectConfigRejectsLocalAppDataRootOnWindows(t *testing.T) {
	homeDir := t.TempDir()
	localAppData := filepath.Join(t.TempDir(), "LocalAppData")
	if _, err := validateAgent47Home(homeDir, localAppData, localAppData, filepath.Join(localAppData, "agent47", "bin"), true); err == nil {
		t.Fatal("expected LOCALAPPDATA root rejection")
	}
}

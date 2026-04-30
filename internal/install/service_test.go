package install

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/manifest"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func TestInstallPublishesSymlinkAndHelpers(t *testing.T) {
	cfg := testConfig(t)

	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Install(context.Background(), cfg, InstallOptions{Force: true}); err != nil {
		t.Fatal(err)
	}

	assertExists(t, filepath.Join(cfg.Agent47Home, "bin", "afs"))
	assertExists(t, filepath.Join(cfg.UserBinDir, "add-agent"))
	assertExists(t, filepath.Join(cfg.UserBinDir, "add-agent-prompt"))
	assertSymlinkTarget(t, filepath.Join(cfg.UserBinDir, "afs"), filepath.Join(cfg.Agent47Home, "bin", "afs"))
}

func TestInstallHonorsCanceledContext(t *testing.T) {
	cfg := testConfig(t)
	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := service.Install(ctx, cfg, InstallOptions{Force: true}); err != context.Canceled {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestInstallWithoutForcePreservesExistingManagedAndUserFiles(t *testing.T) {
	cfg := testConfig(t)
	mustWriteFile(t, filepath.Join(cfg.Agent47Home, "bin", "afs"), "old-launcher\n")
	mustWriteFile(t, filepath.Join(cfg.UserBinDir, "add-agent"), "old-user-helper\n")
	oldTarget := filepath.Join(cfg.HomeDir, "old-afs")
	mustWriteFile(t, oldTarget, "old-afs\n")
	if err := os.Symlink(oldTarget, filepath.Join(cfg.UserBinDir, "afs")); err != nil {
		t.Fatal(err)
	}

	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Install(context.Background(), cfg, InstallOptions{}); err != nil {
		t.Fatal(err)
	}

	assertFileContent(t, filepath.Join(cfg.Agent47Home, "bin", "afs"), "old-launcher\n")
	assertFileContent(t, filepath.Join(cfg.UserBinDir, "add-agent"), "old-user-helper\n")
	assertSymlinkTarget(t, filepath.Join(cfg.UserBinDir, "afs"), oldTarget)
}

func TestInstallRollsBackPublishedScriptsOnCopyFailure(t *testing.T) {
	cfg := testConfig(t)
	mustWriteFile(t, filepath.Join(cfg.UserBinDir, "add-agent"), "old-add-agent\n")
	mustWriteFile(t, filepath.Join(cfg.UserBinDir, "add-agent-prompt"), "old-add-agent-prompt\n")

	t.Setenv("AGENT47_ENABLE_TEST_HOOKS", "true")
	t.Setenv("AGENT47_FAIL_SYMLINK_TARGET", filepath.Join(cfg.UserBinDir, "add-agent-prompt"))

	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Install(context.Background(), cfg, InstallOptions{Force: true}); err == nil {
		t.Fatal("expected install failure")
	}

	assertFileContent(t, filepath.Join(cfg.UserBinDir, "add-agent"), "old-add-agent\n")
	assertFileContent(t, filepath.Join(cfg.UserBinDir, "add-agent-prompt"), "old-add-agent-prompt\n")
	assertNotExists(t, filepath.Join(cfg.UserBinDir, "afs"))
}

func TestInstallPreservesExistingSymlinkOnSwapFailure(t *testing.T) {
	cfg := testConfig(t)
	oldTarget := filepath.Join(cfg.HomeDir, "old-afs")
	mustWriteFile(t, oldTarget, "old\n")
	if err := os.Symlink(oldTarget, filepath.Join(cfg.UserBinDir, "afs")); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AGENT47_ENABLE_TEST_HOOKS", "true")
	t.Setenv("AGENT47_FAIL_SYMLINK_TARGET", filepath.Join(cfg.UserBinDir, "afs"))

	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Install(context.Background(), cfg, InstallOptions{Force: true}); err == nil {
		t.Fatal("expected install failure")
	}

	assertSymlinkTarget(t, filepath.Join(cfg.UserBinDir, "afs"), oldTarget)
}

func TestInstallRestoresTemplatesOnSwapFailure(t *testing.T) {
	cfg := testConfig(t)
	oldTemplates := filepath.Join(cfg.Agent47Home, "templates")
	mustWriteFile(t, filepath.Join(oldTemplates, "AGENTS.md"), "old template\n")
	marker := filepath.Join(cfg.HomeDir, "swap-marker")

	t.Setenv("AGENT47_ENABLE_TEST_HOOKS", "true")
	t.Setenv("AGENT47_FAIL_DIR_SWAP_TARGET", oldTemplates)
	t.Setenv("AGENT47_FAIL_DIR_SWAP_MARKER", marker)

	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Install(context.Background(), cfg, InstallOptions{Force: true}); err == nil {
		t.Fatal("expected install failure")
	}

	assertFileContent(t, filepath.Join(oldTemplates, "AGENTS.md"), "old template\n")
}

func TestUninstallRemovesTemplateBackups(t *testing.T) {
	cfg := testConfig(t)
	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Install(context.Background(), cfg, InstallOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	if err := service.Install(context.Background(), cfg, InstallOptions{Force: true}); err != nil {
		t.Fatal(err)
	}

	matches, err := filepath.Glob(filepath.Join(cfg.Agent47Home, "templates.bak.*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatal("expected template backups after forced reinstall")
	}

	if err := service.Uninstall(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}

	assertNotExists(t, cfg.Agent47Home)
}

func TestInstallRejectsUnsafeRuntimePaths(t *testing.T) {
	cfg := testConfig(t)
	cfg.Agent47Home = cfg.HomeDir

	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Install(context.Background(), cfg, InstallOptions{Force: true}); err == nil {
		t.Fatal("expected unsafe runtime path failure")
	}
}

func TestUninstallPreservesUnmanagedUserFiles(t *testing.T) {
	cfg := testConfig(t)
	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}

	mustWriteFile(t, filepath.Join(cfg.UserBinDir, "add-agent"), "user helper\n")
	mustWriteFile(t, filepath.Join(cfg.UserBinDir, "afs"), "user afs\n")

	if err := service.Uninstall(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, filepath.Join(cfg.UserBinDir, "add-agent"), "user helper\n")
	assertFileContent(t, filepath.Join(cfg.UserBinDir, "afs"), "user afs\n")
}

func TestInstallOutputsParityMarkers(t *testing.T) {
	cfg := testConfig(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	service, err := New(cfg, cli.NewOutput(stdout, stderr))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Install(context.Background(), cfg, InstallOptions{Force: true}); err != nil {
		t.Fatal(err)
	}

	output := stdout.String()
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if !strings.Contains(output, "[*] Installing agent47...") {
		t.Fatalf("expected install banner in output, got %q", output)
	}
	if !strings.Contains(output, "[OK] afs installation complete") {
		t.Fatalf("expected final install marker in output, got %q", output)
	}
}

func TestForceInstallWarnsBeforeTemplateOverwrite(t *testing.T) {
	cfg := testConfig(t)
	mustWriteFile(t, filepath.Join(cfg.Agent47Home, "templates", "AGENTS.md"), "old template\n")
	stdout := &bytes.Buffer{}

	service, err := New(cfg, cli.NewOutput(stdout, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Install(context.Background(), cfg, InstallOptions{Force: true}); err != nil {
		t.Fatal(err)
	}

	output := stdout.String()
	if !strings.Contains(output, "[WARN] Overwriting existing templates at "+filepath.Join(cfg.Agent47Home, "templates")) {
		t.Fatalf("expected overwrite warning in output, got %q", output)
	}
	if !strings.Contains(output, "[INFO] Backup created: "+filepath.Join(cfg.Agent47Home, "templates.bak.")) {
		t.Fatalf("expected backup marker in output, got %q", output)
	}
}

func TestInstallPublishesWindowsLauncherAndCmdHelpers(t *testing.T) {
	cfg := testWindowsConfig(t)

	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Install(context.Background(), cfg, InstallOptions{Force: true}); err != nil {
		t.Fatal(err)
	}

	assertExists(t, filepath.Join(cfg.Agent47Home, "bin", "afs.exe"))
	assertExists(t, filepath.Join(cfg.UserBinDir, "add-agent.cmd"))
	assertExists(t, filepath.Join(cfg.UserBinDir, "add-agent-prompt.cmd"))
	assertExists(t, filepath.Join(cfg.UserBinDir, "add-ss-prompt.cmd"))
	assertFileContainsString(t, filepath.Join(cfg.UserBinDir, "add-agent.cmd"), "\"%~dp0afs.exe\" add-agent %*")
}

func TestWindowsUninstallRemovesExeAndCmdHelpers(t *testing.T) {
	cfg := testWindowsConfig(t)

	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Install(context.Background(), cfg, InstallOptions{Force: true}); err != nil {
		t.Fatal(err)
	}

	if err := service.Uninstall(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}

	assertNotExists(t, filepath.Join(cfg.Agent47Home, "bin", "afs.exe"))
	assertNotExists(t, filepath.Join(cfg.UserBinDir, "add-agent.cmd"))
	assertNotExists(t, filepath.Join(cfg.UserBinDir, "add-agent-prompt.cmd"))
	assertNotExists(t, filepath.Join(cfg.UserBinDir, "add-ss-prompt.cmd"))
}

func TestWindowsInstallRollsBackPublishedCmdHelpersOnWriteFailure(t *testing.T) {
	cfg := testWindowsConfig(t)
	mustWriteFile(t, filepath.Join(cfg.UserBinDir, "add-agent.cmd"), "old-add-agent\r\n")
	mustWriteFile(t, filepath.Join(cfg.UserBinDir, "add-agent-prompt.cmd"), "old-add-agent-prompt\r\n")

	t.Setenv("AGENT47_ENABLE_TEST_HOOKS", "true")
	t.Setenv("AGENT47_FAIL_WRITE_TARGET", filepath.Join(cfg.UserBinDir, "add-agent-prompt.cmd"))

	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Install(context.Background(), cfg, InstallOptions{Force: true}); err == nil {
		t.Fatal("expected install failure")
	}

	assertFileContent(t, filepath.Join(cfg.UserBinDir, "add-agent.cmd"), "old-add-agent\r\n")
	assertFileContent(t, filepath.Join(cfg.UserBinDir, "add-agent-prompt.cmd"), "old-add-agent-prompt\r\n")
	assertNotExists(t, filepath.Join(cfg.UserBinDir, "add-agent"))
	assertNotExists(t, filepath.Join(cfg.UserBinDir, "add-agent-prompt"))
}

func TestWindowsInstallRestoresPublishedAfsOnWriteFailure(t *testing.T) {
	cfg := testWindowsConfig(t)
	mustWriteFile(t, filepath.Join(cfg.UserBinDir, "afs.exe"), "old-afs.exe\r\n")

	t.Setenv("AGENT47_ENABLE_TEST_HOOKS", "true")
	t.Setenv("AGENT47_FAIL_COPY_TARGET", filepath.Join(cfg.UserBinDir, "afs.exe"))

	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Install(context.Background(), cfg, InstallOptions{Force: true}); err == nil {
		t.Fatal("expected install failure")
	}

	assertFileContent(t, filepath.Join(cfg.UserBinDir, "afs.exe"), "old-afs.exe\r\n")
}

func TestPreflightFailsWhenExecutablePathMissing(t *testing.T) {
	cfg := testConfig(t)
	cfg.ExecutablePath = filepath.Join(t.TempDir(), "missing-afs")
	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}

	err = service.preflight(mustManifest(t), cfg)
	if err == nil {
		t.Fatal("expected preflight failure")
	}
}

func TestPreflightFailsWhenRequiredTemplateDirMissing(t *testing.T) {
	cfg := testConfig(t)
	repoRoot := t.TempDir()
	mustWriteFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), "placeholder\n")
	cfg.RepoRoot = repoRoot
	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}

	m := mustManifest(t)
	m.RequiredTemplateDirs = append(m.RequiredTemplateDirs, "skills")
	if err := service.preflight(m, cfg); err == nil {
		t.Fatal("expected missing template dir failure")
	}
}

func TestCopyTemplateTreeCopiesNestedFiles(t *testing.T) {
	cfg := testConfig(t)
	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(t.TempDir(), "copied")
	if err := service.copyTemplateTree(service.Loader.RawSource, "base/rules", dst); err != nil {
		t.Fatal(err)
	}
	assertExists(t, filepath.Join(dst, "rules-backend.yaml"))
}

func TestCopyTemplateTreeFailsForMissingSource(t *testing.T) {
	cfg := testConfig(t)
	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.copyTemplateTree(service.Loader.RawSource, "missing-dir", filepath.Join(t.TempDir(), "dst")); err == nil {
		t.Fatal("expected missing template copy failure")
	}
}

func TestPublishUserEntryUnixWarnsWhenExistingWithoutForce(t *testing.T) {
	cfg := testConfig(t)
	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(cfg.UserBinDir, "afs")
	mustWriteFile(t, target, "existing")
	managed := filepath.Join(cfg.Agent47Home, "bin", "afs")
	mustWriteFile(t, managed, "managed")

	if err := service.publishUserEntry(cfg, InstallOptions{}); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, target, "existing")
}

func TestPublishUserEntryWindowsNoopWhenManagedEqualsTarget(t *testing.T) {
	cfg := testWindowsConfig(t)
	cfg.UserBinDir = filepath.Join(cfg.Agent47Home, "bin")
	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.publishUserEntry(cfg, InstallOptions{}); err != nil {
		t.Fatal(err)
	}
}

func TestPublishUserEntryWindowsWarnsWhenExistingWithoutForce(t *testing.T) {
	cfg := testWindowsConfig(t)
	cfg.UserBinDir = filepath.Join(cfg.HomeDir, "bin")
	if err := os.MkdirAll(cfg.UserBinDir, 0o755); err != nil {
		t.Fatal(err)
	}
	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(cfg.UserBinDir, "afs.exe")
	managed := filepath.Join(cfg.Agent47Home, "bin", "afs.exe")
	mustWriteFile(t, target, "existing")
	mustWriteFile(t, managed, "managed")

	if err := service.publishUserEntry(cfg, InstallOptions{}); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, target, "existing")
}

func TestPublishUserEntryWindowsRefreshesExistingLauncher(t *testing.T) {
	cfg := testWindowsConfig(t)
	cfg.UserBinDir = filepath.Join(cfg.HomeDir, "bin")
	if err := os.MkdirAll(cfg.UserBinDir, 0o755); err != nil {
		t.Fatal(err)
	}
	service, err := New(cfg, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(cfg.UserBinDir, "afs.exe")
	managed := filepath.Join(cfg.Agent47Home, "bin", "afs.exe")
	mustWriteFile(t, target, "existing")
	mustWriteFile(t, managed, "managed")

	if err := service.publishUserEntry(cfg, InstallOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, target, "managed")
	if _, err := os.Stat(target + ".bak.install"); !os.IsNotExist(err) {
		t.Fatalf("expected backup to be removed, got err=%v", err)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

func testConfig(t *testing.T) runtime.Config {
	t.Helper()

	baseDir := t.TempDir()
	homeDir := filepath.Join(baseDir, "home")
	userBinDir := filepath.Join(homeDir, "bin")
	agentHome := filepath.Join(homeDir, ".agent47")
	if err := os.MkdirAll(userBinDir, 0o755); err != nil {
		t.Fatal(err)
	}

	return runtime.Config{
		Version:         "vtest",
		TemplateMode:    runtime.TemplateModeFilesystem,
		RepoRoot:        repoRoot(t),
		ExecutablePath:  os.Args[0],
		HomeDir:         homeDir,
		UserBinDir:      userBinDir,
		Agent47Home:     agentHome,
		UpdateCacheFile: filepath.Join(agentHome, "cache", "update.cache"),
	}
}

func testWindowsConfig(t *testing.T) runtime.Config {
	t.Helper()

	baseDir := t.TempDir()
	homeDir := filepath.Join(baseDir, "home")
	agentHome := filepath.Join(homeDir, "AppData", "Local", "agent47")
	userBinDir := filepath.Join(agentHome, "bin")
	if err := os.MkdirAll(userBinDir, 0o755); err != nil {
		t.Fatal(err)
	}

	return runtime.Config{
		OS:              "windows",
		Version:         "vtest",
		TemplateMode:    runtime.TemplateModeFilesystem,
		RepoRoot:        repoRoot(t),
		ExecutablePath:  os.Args[0],
		HomeDir:         homeDir,
		UserBinDir:      userBinDir,
		Agent47Home:     agentHome,
		UpdateCacheFile: filepath.Join(agentHome, "cache", "update.cache"),
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

func mustWriteFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected path to exist: %s", path)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected path to not exist: %s", path)
	}
}

func assertFileContent(t *testing.T, path string, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != expected {
		t.Fatalf("expected %s to equal %q, got %q", path, expected, string(data))
	}
}

func assertSymlinkTarget(t *testing.T, linkPath string, expectedTarget string) {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(linkPath)
	if err != nil {
		t.Fatal(err)
	}
	expectedResolved, err := filepath.EvalSymlinks(expectedTarget)
	if err != nil {
		expectedResolved = expectedTarget
	}
	if resolved != expectedResolved {
		t.Fatalf("expected symlink %s to point to %s, got %s", linkPath, expectedResolved, resolved)
	}
}

func assertFileContainsString(t *testing.T, path string, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), expected) {
		t.Fatalf("expected %s to contain %q, got %q", path, expected, string(data))
	}
}

func mustManifest(t *testing.T) manifest.Manifest {
	t.Helper()
	return manifest.Manifest{
		RuleTemplates:         []string{"rules-backend.yaml"},
		ManagedTargets:        []string{"templates"},
		PreservedTargets:      []string{"VERSION"},
		RequiredTemplateFiles: []string{"AGENTS.md"},
		RequiredTemplateDirs:  []string{"rules"},
	}
}

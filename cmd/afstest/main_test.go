package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/leanbusqts/agent47/internal/testutil"
)

func TestDetectRepoRootFromFindsParentRepo(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "AGENTS.md"), []byte("# AGENTS\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	nested := filepath.Join(repoRoot, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	detected, err := testutil.DetectRepoRootFrom(nested)
	if err != nil {
		t.Fatal(err)
	}
	if detected != repoRoot {
		t.Fatalf("expected repo root %s, got %s", repoRoot, detected)
	}
}

func TestDetectRepoRootFromFailsOutsideRepo(t *testing.T) {
	start := t.TempDir()
	if _, err := testutil.DetectRepoRootFrom(start); err == nil {
		t.Fatal("expected DetectRepoRootFrom to fail outside a repo")
	}
}

func TestCollectTestPathsReturnsArgsAsIs(t *testing.T) {
	repoRoot := t.TempDir()
	want := []string{"tests/unit/a.bats", "tests/unit/b.bats"}
	got, err := collectTestPaths(repoRoot, want)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected test paths: %v", got)
	}
}

func TestCollectTestPathsFindsOnlyBatsAndSkipsVendor(t *testing.T) {
	repoRoot := t.TempDir()
	mustWriteMainTestFile(t, filepath.Join(repoRoot, "tests", "unit", "a.bats"), "")
	mustWriteMainTestFile(t, filepath.Join(repoRoot, "tests", "vendor", "ignored.bats"), "")
	mustWriteMainTestFile(t, filepath.Join(repoRoot, "tests", "unit", "note.txt"), "")

	got, err := collectTestPaths(repoRoot, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || filepath.Base(got[0]) != "a.bats" {
		t.Fatalf("unexpected collected paths: %v", got)
	}
}

func TestCopyFileCreatesParentDirectoryAndPreservesMode(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src.txt")
	dst := filepath.Join(root, "nested", "dst.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o640); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(src, dst, 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("unexpected permissions: %v", info.Mode().Perm())
	}
}

func TestCopyDirCopiesNestedTree(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	mustWriteMainTestFile(t, filepath.Join(src, "a.txt"), "a")
	mustWriteMainTestFile(t, filepath.Join(src, "nested", "b.txt"), "b")

	if err := copyDir(src, dst); err != nil {
		t.Fatal(err)
	}
	assertMainTestFileExists(t, filepath.Join(dst, "a.txt"))
	assertMainTestFileExists(t, filepath.Join(dst, "nested", "b.txt"))
}

func TestSeedTestRuntimeCopiesTemplatesVersionAndLauncher(t *testing.T) {
	repoRoot := t.TempDir()
	homeDir := filepath.Join(t.TempDir(), "home")
	agentHome := filepath.Join(homeDir, ".agent47")
	launcher := filepath.Join(t.TempDir(), "afs")
	mustWriteMainTestFile(t, filepath.Join(repoRoot, "templates", "AGENTS.md"), "agents")
	mustWriteMainTestFile(t, filepath.Join(repoRoot, "VERSION"), "vtest\n")
	mustWriteMainTestFile(t, launcher, "launcher")

	if err := seedTestRuntime(repoRoot, homeDir, agentHome, launcher); err != nil {
		t.Fatal(err)
	}
	assertMainTestFileExists(t, filepath.Join(agentHome, "templates", "AGENTS.md"))
	assertMainTestFileExists(t, filepath.Join(agentHome, "VERSION"))
	assertMainTestFileExists(t, filepath.Join(agentHome, "bin", "afs"))
}

func TestSeedTestRuntimeReplacesExistingTemplates(t *testing.T) {
	repoRoot := t.TempDir()
	homeDir := filepath.Join(t.TempDir(), "home")
	agentHome := filepath.Join(homeDir, ".agent47")
	launcher := filepath.Join(t.TempDir(), "afs")
	mustWriteMainTestFile(t, filepath.Join(repoRoot, "templates", "AGENTS.md"), "fresh")
	mustWriteMainTestFile(t, filepath.Join(repoRoot, "VERSION"), "vtest\n")
	mustWriteMainTestFile(t, launcher, "launcher")
	mustWriteMainTestFile(t, filepath.Join(agentHome, "templates", "stale.txt"), "stale")

	if err := seedTestRuntime(repoRoot, homeDir, agentHome, launcher); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(agentHome, "templates", "stale.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected stale template to be removed, got err=%v", err)
	}
}

func TestResolveBatsBinPrefersEnvThenVendor(t *testing.T) {
	repoRoot := t.TempDir()
	envBin := filepath.Join(t.TempDir(), "bats")
	mustWriteMainTestFile(t, envBin, "#!/bin/sh\n")
	t.Setenv("BATS_BIN", envBin)

	got, err := resolveBatsBin(repoRoot, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got != envBin {
		t.Fatalf("expected env bats bin, got %s", got)
	}

	t.Setenv("BATS_BIN", filepath.Join(repoRoot, "missing-bats"))
	vendorBin := filepath.Join(repoRoot, "tests", "vendor", "bats", "bin", "bats")
	mustWriteMainTestFile(t, vendorBin, "#!/bin/sh\n")
	got, err = resolveBatsBin(repoRoot, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got != vendorBin {
		t.Fatalf("expected vendor bats bin, got %s", got)
	}
}

func TestResolveGoBinUsesLookPath(t *testing.T) {
	restoreAFSTestHooks()
	defer restoreAFSTestHooks()
	afstestLookPath = func(file string) (string, error) {
		if file == "go" {
			return "/usr/bin/go", nil
		}
		return "", errors.New("missing")
	}

	got, err := resolveGoBin()
	if err != nil {
		t.Fatal(err)
	}
	if got != "/usr/bin/go" {
		t.Fatalf("unexpected go bin: %s", got)
	}
}

func TestResolveGoBinFallsBackToKnownLocations(t *testing.T) {
	restoreAFSTestHooks()
	defer restoreAFSTestHooks()
	afstestLookPath = func(string) (string, error) { return "", errors.New("missing") }
	afstestFileExists = func(path string) bool { return path == "/usr/local/bin/go" }

	got, err := resolveGoBin()
	if err != nil {
		t.Fatal(err)
	}
	if got != "/usr/local/bin/go" {
		t.Fatalf("unexpected go bin: %s", got)
	}
}

func TestResolveGoBinFallsBackToWindowsScoopLocation(t *testing.T) {
	restoreAFSTestHooks()
	defer restoreAFSTestHooks()
	t.Setenv("USERPROFILE", `C:\Users\lean`)
	want := filepath.Join(`C:\Users\lean`, "scoop", "apps", "go", "current", "bin", "go.exe")
	afstestLookPath = func(string) (string, error) { return "", errors.New("missing") }
	afstestFileExists = func(path string) bool { return path == want }

	got, err := resolveGoBin()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("unexpected go bin: %s", got)
	}
}

func TestResolveGoBinFallsBackToProgramFilesOnWindows(t *testing.T) {
	restoreAFSTestHooks()
	defer restoreAFSTestHooks()
	afstestLookPath = func(string) (string, error) { return "", errors.New("missing") }
	afstestFileExists = func(path string) bool { return path == `C:\Program Files\Go\bin\go.exe` }

	got, err := resolveGoBin()
	if err != nil {
		t.Fatal(err)
	}
	if got != `C:\Program Files\Go\bin\go.exe` {
		t.Fatalf("unexpected go bin: %s", got)
	}
}

func TestResolveGoBinFailsWhenNoGoFound(t *testing.T) {
	restoreAFSTestHooks()
	defer restoreAFSTestHooks()
	afstestLookPath = func(string) (string, error) { return "", errors.New("missing") }
	afstestFileExists = func(string) bool { return false }

	if _, err := resolveGoBin(); err == nil {
		t.Fatal("expected go lookup failure")
	}
}

func TestDetectRepoRootUsesWorkingDirectory(t *testing.T) {
	repoRoot := t.TempDir()
	mustWriteMainTestFile(t, filepath.Join(repoRoot, "go.mod"), "module test\n")
	mustWriteMainTestFile(t, filepath.Join(repoRoot, "AGENTS.md"), "agents\n")
	nested := filepath.Join(repoRoot, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(wd) }()

	got, err := detectRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("expected detected repo root")
	}
}

func TestBuildManagedLauncherReturnsErrorWhenGoMissing(t *testing.T) {
	restoreAFSTestHooks()
	defer restoreAFSTestHooks()
	afstestLookPath = func(string) (string, error) { return "", errors.New("missing") }
	afstestFileExists = func(string) bool { return false }

	if err := buildManagedLauncher(t.TempDir(), filepath.Join(t.TempDir(), "afs")); err == nil {
		t.Fatal("expected build launcher failure")
	}
}

func TestBuildManagedLauncherInvokesGoBuild(t *testing.T) {
	restoreAFSTestHooks()
	defer restoreAFSTestHooks()
	repoRoot := t.TempDir()
	outputPath := filepath.Join(t.TempDir(), "tools", "afs")
	afstestLookPath = func(string) (string, error) { return "/usr/bin/go", nil }
	afstestExecCommand = afstestHelperCommand(t, "success")

	if err := buildManagedLauncher(repoRoot, outputPath); err != nil {
		t.Fatalf("expected build launcher success, got %v", err)
	}
}

func TestResolveBatsBinUsesSystemPath(t *testing.T) {
	restoreAFSTestHooks()
	defer restoreAFSTestHooks()
	repoRoot := t.TempDir()
	afstestLookPath = func(file string) (string, error) {
		if file == "bats" {
			return "/usr/bin/bats", nil
		}
		return "", errors.New("missing")
	}

	got, err := resolveBatsBin(repoRoot, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got != "/usr/bin/bats" {
		t.Fatalf("unexpected bats bin: %s", got)
	}
}

func TestResolveBatsBinInstallsTemporaryVendoredCopy(t *testing.T) {
	restoreAFSTestHooks()
	defer restoreAFSTestHooks()
	repoRoot := t.TempDir()
	testTmpRoot := t.TempDir()
	t.Setenv("BATS_BIN", "")
	afstestLookPath = func(string) (string, error) { return "", errors.New("missing") }

	installScript := filepath.Join(repoRoot, "tests", "vendor", "bats", "install.sh")
	mustWriteMainTestFile(t, installScript, "#!/bin/sh\nset -eu\nmkdir -p \"$1/bin\"\nprintf '#!/bin/sh\\nexit 0\\n' > \"$1/bin/bats\"\nchmod +x \"$1/bin/bats\"\n")

	got, err := resolveBatsBin(repoRoot, testTmpRoot)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(testTmpRoot, "tools", "bats", "bin", "bats") {
		t.Fatalf("unexpected bats path: %s", got)
	}
	assertMainTestFileExists(t, got)
}

func TestResolveBatsBinFailsWhenVendoredInstallerFails(t *testing.T) {
	restoreAFSTestHooks()
	defer restoreAFSTestHooks()
	repoRoot := t.TempDir()
	t.Setenv("BATS_BIN", "")
	afstestLookPath = func(string) (string, error) { return "", errors.New("missing") }
	mustWriteMainTestFile(t, filepath.Join(repoRoot, "tests", "vendor", "bats", "install.sh"), "#!/bin/sh\nexit 9\n")

	if _, err := resolveBatsBin(repoRoot, t.TempDir()); err == nil {
		t.Fatal("expected vendored bats install failure")
	}
}

func TestRunReturnsOneWhenRepoRootFails(t *testing.T) {
	restoreAFSTestHooks()
	defer restoreAFSTestHooks()
	var stderr bytes.Buffer
	afstestStderr = &stderr
	afstestDetectRepoRoot = func() (string, error) { return "", errors.New("boom") }

	if got := run(nil); got != 1 {
		t.Fatalf("expected status 1, got %d", got)
	}
}

func TestRunReturnsOneWhenTempRootCreationFails(t *testing.T) {
	restoreAFSTestHooks()
	defer restoreAFSTestHooks()
	afstestDetectRepoRoot = func() (string, error) { return "/repo", nil }
	afstestMkdirTemp = func(string, string) (string, error) { return "", errors.New("temp fail") }

	if got := run(nil); got != 1 {
		t.Fatalf("expected status 1, got %d", got)
	}
}

func TestRunReturnsOneWhenBuildLauncherFails(t *testing.T) {
	restoreAFSTestHooks()
	defer restoreAFSTestHooks()
	afstestDetectRepoRoot = func() (string, error) { return "/repo", nil }
	afstestMkdirTemp = func(string, string) (string, error) { return t.TempDir(), nil }
	afstestBuildLauncher = func(string, string) error { return errors.New("build fail") }

	if got := run(nil); got != 1 {
		t.Fatalf("expected status 1, got %d", got)
	}
}

func TestRunReturnsOneWhenSeedRuntimeFails(t *testing.T) {
	restoreAFSTestHooks()
	defer restoreAFSTestHooks()
	afstestDetectRepoRoot = func() (string, error) { return "/repo", nil }
	afstestMkdirTemp = func(string, string) (string, error) { return t.TempDir(), nil }
	afstestBuildLauncher = func(string, string) error { return nil }
	afstestSeedRuntime = func(string, string, string, string) error { return errors.New("seed fail") }

	if got := run(nil); got != 1 {
		t.Fatalf("expected status 1, got %d", got)
	}
}

func TestRunReturnsOneWhenResolveBatsFails(t *testing.T) {
	restoreAFSTestHooks()
	defer restoreAFSTestHooks()
	afstestDetectRepoRoot = func() (string, error) { return "/repo", nil }
	afstestMkdirTemp = func(string, string) (string, error) { return t.TempDir(), nil }
	afstestBuildLauncher = func(string, string) error { return nil }
	afstestSeedRuntime = func(string, string, string, string) error { return nil }
	afstestResolveBatsBin = func(string, string) (string, error) { return "", errors.New("missing bats") }

	if got := run(nil); got != 1 {
		t.Fatalf("expected status 1, got %d", got)
	}
}

func TestRunReturnsOneWhenCollectPathsFails(t *testing.T) {
	restoreAFSTestHooks()
	defer restoreAFSTestHooks()
	afstestDetectRepoRoot = func() (string, error) { return "/repo", nil }
	afstestMkdirTemp = func(string, string) (string, error) { return t.TempDir(), nil }
	afstestBuildLauncher = func(string, string) error { return nil }
	afstestSeedRuntime = func(string, string, string, string) error { return nil }
	afstestResolveBatsBin = func(string, string) (string, error) { return "bats", nil }
	afstestCollectTestPaths = func(string, []string) ([]string, error) { return nil, errors.New("walk failed") }

	if got := run(nil); got != 1 {
		t.Fatalf("expected status 1, got %d", got)
	}
}

func TestRunReturnsZeroWhenNoTestsFound(t *testing.T) {
	restoreAFSTestHooks()
	defer restoreAFSTestHooks()
	var stdout, stderr bytes.Buffer
	afstestStdout = &stdout
	afstestStderr = &stderr
	afstestDetectRepoRoot = func() (string, error) { return "/repo", nil }
	afstestMkdirTemp = func(string, string) (string, error) { return t.TempDir(), nil }
	afstestBuildLauncher = func(string, string) error { return nil }
	afstestSeedRuntime = func(string, string, string, string) error { return nil }
	afstestResolveBatsBin = func(string, string) (string, error) { return "bats", nil }
	afstestCollectTestPaths = func(string, []string) ([]string, error) { return nil, nil }

	if got := run(nil); got != 0 {
		t.Fatalf("expected status 0, got %d", got)
	}
}

func TestRunReturnsBatsExitCode(t *testing.T) {
	restoreAFSTestHooks()
	defer restoreAFSTestHooks()
	var stdout, stderr bytes.Buffer
	afstestStdout = &stdout
	afstestStderr = &stderr
	afstestDetectRepoRoot = func() (string, error) { return "/repo", nil }
	afstestMkdirTemp = func(string, string) (string, error) { return t.TempDir(), nil }
	afstestBuildLauncher = func(string, string) error { return nil }
	afstestSeedRuntime = func(string, string, string, string) error { return nil }
	afstestResolveBatsBin = func(string, string) (string, error) { return "bats", nil }
	afstestCollectTestPaths = func(string, []string) ([]string, error) { return []string{"a.bats"}, nil }
	afstestExecCommand = afstestHelperCommand(t, "bats-fail")

	if got := run(nil); got != 6 {
		t.Fatalf("expected bats exit code 6, got %d", got)
	}
}

func TestRunReturnsOneWhenBatsCommandFailsGenerically(t *testing.T) {
	restoreAFSTestHooks()
	defer restoreAFSTestHooks()
	afstestDetectRepoRoot = func() (string, error) { return "/repo", nil }
	afstestMkdirTemp = func(string, string) (string, error) { return t.TempDir(), nil }
	afstestBuildLauncher = func(string, string) error { return nil }
	afstestSeedRuntime = func(string, string, string, string) error { return nil }
	afstestResolveBatsBin = func(string, string) (string, error) { return "bats", nil }
	afstestCollectTestPaths = func(string, []string) ([]string, error) { return []string{"a.bats"}, nil }
	afstestExecCommand = func(string, ...string) *exec.Cmd { return exec.Command("/path/does/not/exist") }

	if got := run(nil); got != 1 {
		t.Fatalf("expected status 1, got %d", got)
	}
}

func TestRunReturnsZeroWhenBatsSucceeds(t *testing.T) {
	restoreAFSTestHooks()
	defer restoreAFSTestHooks()
	var stdout, stderr bytes.Buffer
	afstestStdout = &stdout
	afstestStderr = &stderr
	afstestDetectRepoRoot = func() (string, error) { return "/repo", nil }
	afstestMkdirTemp = func(string, string) (string, error) { return t.TempDir(), nil }
	afstestBuildLauncher = func(string, string) error { return nil }
	afstestSeedRuntime = func(string, string, string, string) error { return nil }
	afstestResolveBatsBin = func(string, string) (string, error) { return "bats", nil }
	afstestCollectTestPaths = func(string, []string) ([]string, error) { return []string{"a.bats"}, nil }
	afstestExecCommand = afstestHelperCommand(t, "success")

	if got := run(nil); got != 0 {
		t.Fatalf("expected status 0, got %d stderr=%s", got, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Using bats binary")) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestAFSTestHelperProcess(t *testing.T) {
	args := os.Args
	if len(args) < 5 || args[1] != "-test.run=TestAFSTestHelperProcess" {
		return
	}
	if args[3] == "bats-fail" {
		os.Exit(6)
	}
	os.Exit(0)
}

func afstestHelperCommand(t *testing.T, mode string) func(string, ...string) *exec.Cmd {
	t.Helper()
	return func(name string, args ...string) *exec.Cmd {
		cmdArgs := []string{"-test.run=TestAFSTestHelperProcess", "--", mode, name}
		cmdArgs = append(cmdArgs, args...)
		cmd := exec.Command(os.Args[0], cmdArgs...)
		return cmd
	}
}

func restoreAFSTestHooks() {
	afstestDetectRepoRoot = detectRepoRoot
	afstestMkdirTemp = os.MkdirTemp
	afstestRemoveAll = os.RemoveAll
	afstestBuildLauncher = buildManagedLauncher
	afstestSeedRuntime = seedTestRuntime
	afstestResolveBatsBin = resolveBatsBin
	afstestCollectTestPaths = collectTestPaths
	afstestExecCommand = exec.Command
	afstestLookPath = exec.LookPath
	afstestFileExists = testutil.FileExists
	afstestStdout = os.Stdout
	afstestStderr = os.Stderr
}

func mustWriteMainTestFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func assertMainTestFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %s", path)
	}
}

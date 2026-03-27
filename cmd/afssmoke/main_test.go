package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	goRuntime "runtime"
	"strings"
	"testing"

	"github.com/leanbusqts/agent47/internal/testutil"
)

func TestBuildSmokeCommandsReturnsExpectedPaths(t *testing.T) {
	repoRoot := filepath.Join("repo", "root")
	homeDir := filepath.Join("home")
	userBinDir := filepath.Join("bin")

	smokeOS = "windows"
	cmd, doctorPath := buildSmokeCommands(repoRoot, homeDir, userBinDir)
	if !strings.Contains(strings.Join(cmd.Args, " "), "install.ps1") {
		t.Fatalf("expected windows install.ps1 command, got %v", cmd.Args)
	}
	if doctorPath != filepath.Join(userBinDir, "afs.exe") {
		t.Fatalf("unexpected doctor path: %s", doctorPath)
	}

	smokeOS = "darwin"
	if cmd.Path != filepath.Join(repoRoot, "install.sh") {
		cmd, doctorPath = buildSmokeCommands(repoRoot, homeDir, userBinDir)
	}
	if cmd.Path != filepath.Join(repoRoot, "install.sh") {
		t.Fatalf("unexpected install path: %s", cmd.Path)
	}
	if doctorPath != filepath.Join(homeDir, "bin", "afs") {
		t.Fatalf("unexpected doctor path: %s", doctorPath)
	}
	restoreSmokeHooks()
}

func TestRunReturnsOneWhenGetwdFails(t *testing.T) {
	restoreSmokeHooks()
	defer restoreSmokeHooks()
	smokeGetwd = func() (string, error) { return "", errors.New("boom") }

	if got := run(); got != 1 {
		t.Fatalf("expected status 1, got %d", got)
	}
}

func TestRunReturnsInstallExitCode(t *testing.T) {
	restoreSmokeHooks()
	defer restoreSmokeHooks()
	var stdout, stderr bytes.Buffer
	smokeStdout = &stdout
	smokeStderr = &stderr
	smokeGetwd = func() (string, error) { return "/repo", nil }
	smokeDetectRepoRootFrom = func(string) (string, error) { return "/repo", nil }
	smokeMkdirTemp = func(string, string) (string, error) { return t.TempDir(), nil }
	smokeExecCommand = smokeHelperCommand(t, "install-fail")

	if got := run(); got != 7 {
		t.Fatalf("expected install exit code 7, got %d", got)
	}
}

func TestRunReturnsOneWhenRepoRootDetectionFails(t *testing.T) {
	restoreSmokeHooks()
	defer restoreSmokeHooks()
	smokeGetwd = func() (string, error) { return "/repo", nil }
	smokeDetectRepoRootFrom = func(string) (string, error) { return "", errors.New("missing repo") }

	if got := run(); got != 1 {
		t.Fatalf("expected status 1, got %d", got)
	}
}

func TestRunReturnsOneWhenCreateTempFails(t *testing.T) {
	restoreSmokeHooks()
	defer restoreSmokeHooks()
	smokeGetwd = func() (string, error) { return "/repo", nil }
	smokeDetectRepoRootFrom = func(string) (string, error) { return "/repo", nil }
	smokeMkdirTemp = func(string, string) (string, error) { return "", errors.New("no temp") }

	if got := run(); got != 1 {
		t.Fatalf("expected status 1, got %d", got)
	}
}

func TestRunReturnsOneWhenPrepareHomeFails(t *testing.T) {
	restoreSmokeHooks()
	defer restoreSmokeHooks()
	smokeGetwd = func() (string, error) { return "/repo", nil }
	smokeDetectRepoRootFrom = func(string) (string, error) { return "/repo", nil }
	smokeMkdirTemp = func(string, string) (string, error) { return t.TempDir(), nil }
	smokeMkdirAll = func(string, os.FileMode) error { return errors.New("mkdir failed") }

	if got := run(); got != 1 {
		t.Fatalf("expected status 1, got %d", got)
	}
}

func TestRunReturnsDoctorExitCode(t *testing.T) {
	restoreSmokeHooks()
	defer restoreSmokeHooks()
	var stdout, stderr bytes.Buffer
	smokeStdout = &stdout
	smokeStderr = &stderr
	smokeGetwd = func() (string, error) { return "/repo", nil }
	smokeDetectRepoRootFrom = func(string) (string, error) { return "/repo", nil }
	smokeMkdirTemp = func(string, string) (string, error) { return t.TempDir(), nil }
	smokeExecCommand = smokeHelperCommand(t, "doctor-fail")

	if got := run(); got != 9 {
		t.Fatalf("expected doctor exit code 9, got %d", got)
	}
}

func TestRunReturnsOneWhenInstallCommandFailsGenerically(t *testing.T) {
	restoreSmokeHooks()
	defer restoreSmokeHooks()
	smokeGetwd = func() (string, error) { return "/repo", nil }
	smokeDetectRepoRootFrom = func(string) (string, error) { return "/repo", nil }
	smokeMkdirTemp = func(string, string) (string, error) { return t.TempDir(), nil }
	smokeExecCommand = func(string, ...string) *exec.Cmd {
		return exec.Command("/definitely-missing-binary")
	}

	if got := run(); got != 1 {
		t.Fatalf("expected generic failure status 1, got %d", got)
	}
}

func TestRunReturnsOneWhenDoctorCommandFailsGenerically(t *testing.T) {
	restoreSmokeHooks()
	defer restoreSmokeHooks()
	smokeGetwd = func() (string, error) { return "/repo", nil }
	smokeDetectRepoRootFrom = func(string) (string, error) { return "/repo", nil }
	smokeMkdirTemp = func(string, string) (string, error) { return t.TempDir(), nil }
	smokeExecCommand = func(name string, args ...string) *exec.Cmd {
		if strings.Contains(name, "install") || name == "powershell" {
			return smokeHelperCommand(t, "success")(name, args...)
		}
		return exec.Command("/definitely-missing-binary")
	}

	if got := run(); got != 1 {
		t.Fatalf("expected generic failure status 1, got %d", got)
	}
}

func TestRunPrintsSuccessWhenInstallAndDoctorSucceed(t *testing.T) {
	restoreSmokeHooks()
	defer restoreSmokeHooks()
	var stdout, stderr bytes.Buffer
	smokeStdout = &stdout
	smokeStderr = &stderr
	smokeGetwd = func() (string, error) { return "/repo", nil }
	smokeDetectRepoRootFrom = func(string) (string, error) { return "/repo", nil }
	smokeMkdirTemp = func(string, string) (string, error) { return t.TempDir(), nil }
	smokeExecCommand = smokeHelperCommand(t, "success")

	if got := run(); got != 0 {
		t.Fatalf("expected success status, got %d stderr=%s", got, stderr.String())
	}
	if !strings.Contains(stdout.String(), "[OK] smoke install succeeded") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestSmokeHelperProcess(t *testing.T) {
	args := os.Args
	if len(args) < 6 || args[1] != "-test.run=TestSmokeHelperProcess" {
		return
	}
	role := args[3]
	mode := args[4]
	if role == "install" {
		if mode == "install-fail" {
			os.Exit(7)
		}
		os.Exit(0)
	}
	if mode == "doctor-fail" {
		os.Exit(9)
	}
	os.Stdout.WriteString("[OK] doctor ran\n")
	os.Exit(0)
}

func smokeHelperCommand(t *testing.T, mode string) func(string, ...string) *exec.Cmd {
	t.Helper()
	return func(name string, args ...string) *exec.Cmd {
		role := "doctor"
		if strings.Contains(name, "install") || name == "powershell" {
			role = "install"
		}
		cmdArgs := []string{"-test.run=TestSmokeHelperProcess", "--", role, mode, name}
		cmdArgs = append(cmdArgs, args...)
		cmd := exec.Command(os.Args[0], cmdArgs...)
		return cmd
	}
}

func restoreSmokeHooks() {
	smokeOS = goRuntime.GOOS
	smokeGetwd = os.Getwd
	smokeDetectRepoRootFrom = testutil.DetectRepoRootFrom
	smokeMkdirTemp = os.MkdirTemp
	smokeMkdirAll = os.MkdirAll
	smokeRemoveAll = os.RemoveAll
	smokeExecCommand = exec.Command
	smokeStdout = os.Stdout
	smokeStderr = os.Stderr
}

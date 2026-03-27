package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	goRuntime "runtime"

	"github.com/leanbusqts/agent47/internal/testutil"
)

var (
	smokeOS                = goRuntime.GOOS
	smokeGetwd             = os.Getwd
	smokeDetectRepoRootFrom = testutil.DetectRepoRootFrom
	smokeMkdirTemp         = os.MkdirTemp
	smokeMkdirAll          = os.MkdirAll
	smokeRemoveAll         = os.RemoveAll
	smokeExecCommand       = exec.Command
	smokeStdout  io.Writer = os.Stdout
	smokeStderr  io.Writer = os.Stderr
)

func main() {
	os.Exit(run())
}

func run() int {
	wd, err := smokeGetwd()
	if err != nil {
		fmt.Fprintf(smokeStderr, "[ERR] failed to detect repo root: %v\n", err)
		return 1
	}
	repoRoot, err := smokeDetectRepoRootFrom(wd)
	if err != nil {
		fmt.Fprintf(smokeStderr, "[ERR] failed to detect repo root: %v\n", err)
		return 1
	}

	smokeRoot, err := smokeMkdirTemp(os.TempDir(), "afs-smoke-")
	if err != nil {
		fmt.Fprintf(smokeStderr, "[ERR] failed to create smoke root: %v\n", err)
		return 1
	}
	defer smokeRemoveAll(smokeRoot)

	homeDir := filepath.Join(smokeRoot, "home")
	agentHome := filepath.Join(homeDir, ".agent47")
	userBinDir := filepath.Join(homeDir, "bin")
	if smokeOS == "windows" {
		agentHome = filepath.Join(homeDir, "AppData", "Local", "agent47")
		userBinDir = filepath.Join(agentHome, "bin")
	}
	if err := smokeMkdirAll(userBinDir, 0o755); err != nil {
		fmt.Fprintf(smokeStderr, "[ERR] failed to prepare smoke home: %v\n", err)
		return 1
	}

	env := append(os.Environ(),
		"HOME="+homeDir,
		"USERPROFILE="+homeDir,
		"LOCALAPPDATA="+filepath.Join(homeDir, "AppData", "Local"),
		"AGENT47_HOME="+agentHome,
		"PATH="+userBinDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)

	installCmd, doctorPath := buildSmokeCommands(repoRoot, homeDir, userBinDir)
	installCmd.Args = append(installCmd.Args, "--force", "--non-interactive")
	installCmd.Env = env
	installCmd.Stdout = smokeStdout
	installCmd.Stderr = smokeStderr
	if err := installCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(smokeStderr, "[ERR] failed to run install: %v\n", err)
		return 1
	}

	doctorCmd := smokeExecCommand(doctorPath, "doctor", "--fail-on-warn")
	doctorCmd.Env = env
	var doctorOut bytes.Buffer
	doctorCmd.Stdout = &doctorOut
	doctorCmd.Stderr = smokeStderr
	if err := doctorCmd.Run(); err != nil {
		fmt.Fprint(smokeStdout, doctorOut.String())
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(smokeStderr, "[ERR] failed to run doctor: %v\n", err)
		return 1
	}

	fmt.Fprint(smokeStdout, doctorOut.String())
	fmt.Fprintln(smokeStdout, "[OK] smoke install succeeded")
	return 0
}

func buildSmokeCommands(repoRoot, homeDir, userBinDir string) (*exec.Cmd, string) {
	if smokeOS == "windows" {
		return smokeExecCommand(
			"powershell",
			"-NoProfile",
			"-ExecutionPolicy",
			"Bypass",
			"-File",
			filepath.Join(repoRoot, "install.ps1"),
		), filepath.Join(userBinDir, "afs.exe")
	}

	return smokeExecCommand(filepath.Join(repoRoot, "install.sh")), filepath.Join(homeDir, "bin", "afs")
}

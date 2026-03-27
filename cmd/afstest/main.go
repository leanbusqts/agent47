package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/leanbusqts/agent47/internal/testutil"
)

var (
	afstestDetectRepoRoot             = detectRepoRoot
	afstestMkdirTemp                  = os.MkdirTemp
	afstestRemoveAll                  = os.RemoveAll
	afstestBuildLauncher              = buildManagedLauncher
	afstestSeedRuntime                = seedTestRuntime
	afstestResolveBatsBin             = resolveBatsBin
	afstestCollectTestPaths           = collectTestPaths
	afstestExecCommand                = exec.Command
	afstestLookPath                   = exec.LookPath
	afstestFileExists                 = testutil.FileExists
	afstestStdout           io.Writer = os.Stdout
	afstestStderr           io.Writer = os.Stderr
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	repoRoot, err := afstestDetectRepoRoot()
	if err != nil {
		fmt.Fprintf(afstestStderr, "[ERR] %v\n", err)
		return 1
	}

	testTmpRoot, err := afstestMkdirTemp(os.TempDir(), "afs-test-")
	if err != nil {
		fmt.Fprintf(afstestStderr, "[ERR] failed to create temp root: %v\n", err)
		return 1
	}
	defer afstestRemoveAll(testTmpRoot)

	homeDir := filepath.Join(testTmpRoot, "home")
	agentHome := filepath.Join(homeDir, ".agent47")
	testLauncher := filepath.Join(testTmpRoot, "tools", "afs")
	if err := afstestBuildLauncher(repoRoot, testLauncher); err != nil {
		fmt.Fprintf(afstestStderr, "[ERR] failed to build test launcher: %v\n", err)
		return 1
	}

	if err := afstestSeedRuntime(repoRoot, homeDir, agentHome, testLauncher); err != nil {
		fmt.Fprintf(afstestStderr, "[ERR] failed to seed test runtime: %v\n", err)
		return 1
	}

	batsBin, err := afstestResolveBatsBin(repoRoot, testTmpRoot)
	if err != nil {
		fmt.Fprintf(afstestStderr, "[ERR] %v\n", err)
		return 1
	}

	testPaths, err := afstestCollectTestPaths(repoRoot, args)
	if err != nil {
		fmt.Fprintf(afstestStderr, "[ERR] failed to collect test paths: %v\n", err)
		return 1
	}
	if len(testPaths) == 0 {
		fmt.Fprintln(afstestStdout, "[WARN] No tests found under", filepath.Join(repoRoot, "tests"))
		return 0
	}

	fmt.Fprintf(afstestStdout, "[INFO] Running tests with HOME=%s and AGENT47_HOME=%s\n", homeDir, agentHome)
	fmt.Fprintf(afstestStdout, "[INFO] Using bats binary: %s\n", batsBin)

	cmd := afstestExecCommand(batsBin, testPaths...)
	cmd.Stdout = afstestStdout
	cmd.Stderr = afstestStderr
	cmd.Env = append(os.Environ(),
		"HOME="+homeDir,
		"AGENT47_HOME="+agentHome,
		"PATH="+filepath.Join(homeDir, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
		"BATS_LIB_PATH="+filepath.Join(repoRoot, "tests", "helpers")+string(os.PathListSeparator)+filepath.Join(repoRoot, "tests"),
		"AGENT47_STAGE_ROOT="+testTmpRoot,
		"TEST_TMP_ROOT="+testTmpRoot,
		"TEST_AFS_LAUNCHER="+testLauncher,
		"ROOT_DIR="+repoRoot,
	)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(afstestStderr, "[ERR] failed to run bats: %v\n", err)
		return 1
	}
	return 0
}

func detectRepoRoot() (string, error) {
	return testutil.DetectRepoRoot()
}

func seedTestRuntime(repoRoot, homeDir, agentHome, launcherPath string) error {
	if err := os.MkdirAll(filepath.Join(homeDir, "bin"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(agentHome, "bin"), 0o755); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(agentHome, "templates")); err != nil {
		return err
	}
	if err := copyDir(filepath.Join(repoRoot, "templates"), filepath.Join(agentHome, "templates")); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(repoRoot, "VERSION"), filepath.Join(agentHome, "VERSION"), 0o644); err != nil {
		return err
	}
	if err := copyFile(launcherPath, filepath.Join(agentHome, "bin", "afs"), 0o755); err != nil {
		return err
	}
	return nil
}

func buildManagedLauncher(repoRoot, outputPath string) error {
	goBin, err := resolveGoBin()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}

	cmd := afstestExecCommand(goBin, "build", "-o", outputPath, "./cmd/afs")
	cmd.Dir = repoRoot
	cmd.Stdout = nil
	cmd.Stderr = afstestStderr
	cmd.Env = os.Environ()
	return cmd.Run()
}

func resolveGoBin() (string, error) {
	if goBin, err := afstestLookPath("go"); err == nil {
		return goBin, nil
	}
	for _, candidate := range goFallbackCandidates() {
		if afstestFileExists(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("failed to locate go toolchain")
}

func goFallbackCandidates() []string {
	candidates := []string{"/opt/homebrew/bin/go", "/usr/local/bin/go", "/usr/bin/go"}
	userProfile := os.Getenv("USERPROFILE")
	if userProfile != "" {
		candidates = append(candidates, filepath.Join(userProfile, "scoop", "apps", "go", "current", "bin", "go.exe"))
	}
	return append(candidates, `C:\Program Files\Go\bin\go.exe`)
}

func resolveBatsBin(repoRoot, testTmpRoot string) (string, error) {
	if batsBin := os.Getenv("BATS_BIN"); batsBin != "" {
		if info, err := os.Stat(batsBin); err == nil && info.Mode().IsRegular() {
			return batsBin, nil
		}
	}

	vendorBin := filepath.Join(repoRoot, "tests", "vendor", "bats", "bin", "bats")
	if info, err := os.Stat(vendorBin); err == nil && info.Mode().IsRegular() {
		return vendorBin, nil
	}

	if systemBin, err := afstestLookPath("bats"); err == nil {
		return systemBin, nil
	}

	vendorInstall := filepath.Join(repoRoot, "tests", "vendor", "bats", "install.sh")
	if info, err := os.Stat(vendorInstall); err == nil && info.Mode().IsRegular() {
		fmt.Println("[INFO] bats not found; installing a temporary copy from tests/vendor/bats")
		tempBatsRoot := filepath.Join(testTmpRoot, "tools", "bats")
		if err := os.MkdirAll(tempBatsRoot, 0o755); err != nil {
			return "", err
		}
		cmd := afstestExecCommand("bash", vendorInstall, tempBatsRoot)
		cmd.Stdout = nil
		cmd.Stderr = afstestStderr
		if err := cmd.Run(); err != nil {
			return "", err
		}
		return filepath.Join(tempBatsRoot, "bin", "bats"), nil
	}

	return "", fmt.Errorf("bats not found. Set BATS_BIN, install bats on PATH, or restore tests/vendor/bats")
}

func collectTestPaths(repoRoot string, args []string) ([]string, error) {
	if len(args) > 0 {
		return args, nil
	}

	var paths []string
	err := filepath.WalkDir(filepath.Join(repoRoot, "tests"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path == filepath.Join(repoRoot, "tests", "vendor") {
			return filepath.SkipDir
		}
		if !d.IsDir() && filepath.Ext(path) == ".bats" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func copyFile(src, dst string, perm os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, perm)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target, info.Mode())
	})
}

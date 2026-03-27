package install

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/leanbusqts/agent47/internal/cli"
	runtimecfg "github.com/leanbusqts/agent47/internal/runtime"
)

func TestRunWindowsPostInstallSkipsPathWarningWhenRequested(t *testing.T) {
	cfg := runtimecfg.Config{
		OS:         "windows",
		HomeDir:    t.TempDir(),
		UserBinDir: `C:\Users\Test\AppData\Local\agent47\bin`,
	}

	var stdout bytes.Buffer
	out := cli.NewOutput(&stdout, &stdout)
	err := RunPostInstall(context.Background(), cfg, out, PostInstallOptions{SkipPathCheck: true})
	if err != nil {
		t.Fatal(err)
	}

	output := stdout.String()
	if strings.Contains(output, "managed bin not in PATH") {
		t.Fatalf("expected no PATH warning, got %q", output)
	}
	if !strings.Contains(output, "[OK] afs installed") {
		t.Fatalf("expected installed marker, got %q", output)
	}
}

func TestSamePathIsCaseInsensitiveOnWindows(t *testing.T) {
	if !samePath("windows", `C:\USERS\Test\AppData\Local\AGENT47\BIN`, `c:\users\test\appdata\local\agent47\bin`) {
		t.Fatal("expected samePath to match case-insensitively on windows")
	}
}

func TestPathContainsMatchesUnixExactly(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "bin")
	t.Setenv("PATH", strings.Join([]string{dir, "/usr/bin"}, string(os.PathListSeparator)))

	if !PathContains("darwin", dir) {
		t.Fatal("expected unix pathContains match")
	}
	if PathContains("darwin", strings.ToUpper(dir)) {
		t.Fatal("expected unix pathContains to remain case-sensitive")
	}
}

func TestRunPostInstallUnixPathAlreadyPresent(t *testing.T) {
	homeDir := t.TempDir()
	userBinDir := filepath.Join(homeDir, "bin")
	t.Setenv("PATH", userBinDir)

	var stdout bytes.Buffer
	out := cli.NewOutput(&stdout, &stdout)
	err := RunPostInstall(context.Background(), runtimecfg.Config{
		OS:         "darwin",
		HomeDir:    homeDir,
		UserBinDir: userBinDir,
	}, out, PostInstallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "~/bin in PATH") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestRunPostInstallUnixNonInteractiveWithoutPath(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("PATH", "/usr/bin:/bin")
	t.Setenv("SHELL", "/bin/zsh")

	var stdout bytes.Buffer
	out := cli.NewOutput(&stdout, &stdout)
	err := RunPostInstall(context.Background(), runtimecfg.Config{
		OS:         "darwin",
		HomeDir:    homeDir,
		UserBinDir: filepath.Join(homeDir, "bin"),
	}, out, PostInstallOptions{NonInteractive: true})
	if err != nil {
		t.Fatal(err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Non-interactive install; skipping shell rc update") {
		t.Fatalf("unexpected output: %s", output)
	}
	if !strings.Contains(output, ".zshrc") {
		t.Fatalf("expected suggested shell rc file in output: %s", output)
	}
}

func TestDetectShellRCFilePrefersBashProfileThenBashrcThenProfile(t *testing.T) {
	homeDir := t.TempDir()
	bashProfile := filepath.Join(homeDir, ".bash_profile")
	bashrc := filepath.Join(homeDir, ".bashrc")

	got := detectShellRCFile("/bin/bash", homeDir)
	wantDefault := filepath.Join(homeDir, ".profile")
	if runtime.GOOS == "darwin" {
		wantDefault = bashProfile
	}
	if got != wantDefault {
		t.Fatalf("unexpected default rc file: %s", got)
	}
	if err := os.WriteFile(bashrc, []byte("export PATH\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = detectShellRCFile("/bin/bash", homeDir)
	if runtime.GOOS == "darwin" {
		if got != bashProfile {
			t.Fatalf("expected .bash_profile on darwin, got %s", got)
		}
	} else if got != bashrc {
		t.Fatalf("expected .bashrc, got %s", got)
	}
	if err := os.WriteFile(bashProfile, []byte("export PATH\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := detectShellRCFile("/bin/bash", homeDir); got != bashProfile {
		t.Fatalf("expected .bash_profile, got %s", got)
	}
}

func TestAppendPathExportCreatesBackupAndDoesNotDuplicate(t *testing.T) {
	homeDir := t.TempDir()
	rcFile := filepath.Join(homeDir, ".zshrc")
	if err := os.WriteFile(rcFile, []byte("# existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	out := cli.NewOutput(&stdout, &stdout)
	if err := appendPathExport(rcFile, out); err != nil {
		t.Fatal(err)
	}
	if err := appendPathExport(rcFile, out); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "export PATH=\"$HOME/bin:$PATH\"") != 1 {
		t.Fatalf("expected single export line, got %s", string(data))
	}
	if _, err := os.Stat(rcFile + ".bak"); err != nil {
		t.Fatalf("expected backup file: %v", err)
	}
}

func TestHasTTYReturnsFalseForRegularFile(t *testing.T) {
	file, err := os.Create(filepath.Join(t.TempDir(), "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	if hasTTY(file) {
		t.Fatal("did not expect regular file to be a tty")
	}
}

func TestRunWindowsPostInstallWarnsWhenManagedBinMissingFromPath(t *testing.T) {
	t.Setenv("PATH", `C:\Windows\System32`)

	var stdout bytes.Buffer
	out := cli.NewOutput(&stdout, &stdout)
	err := RunPostInstall(context.Background(), runtimecfg.Config{
		OS:         "windows",
		HomeDir:    t.TempDir(),
		UserBinDir: `C:\Users\Test\AppData\Local\agent47\bin`,
	}, out, PostInstallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "managed bin not in PATH") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestRunWindowsPostInstallAcceptsManagedBinInPath(t *testing.T) {
	t.Setenv("PATH", strings.Join([]string{`Users\Test\AppData\Local\agent47\bin`}, string(os.PathListSeparator)))

	var stdout bytes.Buffer
	out := cli.NewOutput(&stdout, &stdout)
	err := RunPostInstall(context.Background(), runtimecfg.Config{
		OS:         "windows",
		HomeDir:    t.TempDir(),
		UserBinDir: `Users\Test\AppData\Local\agent47\bin`,
	}, out, PostInstallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "managed bin in PATH") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestDetectShellRCFileHandlesZshAndDefault(t *testing.T) {
	homeDir := t.TempDir()
	if got := detectShellRCFile("/bin/zsh", homeDir); got != filepath.Join(homeDir, ".zshrc") {
		t.Fatalf("unexpected zsh rc file: %s", got)
	}
	if got := detectShellRCFile("/bin/fish", homeDir); got != filepath.Join(homeDir, ".profile") {
		t.Fatalf("unexpected default rc file: %s", got)
	}
}

func TestRunPostInstallInteractiveYesAppendsExport(t *testing.T) {
	restorePostInstallHooks()
	defer restorePostInstallHooks()
	homeDir := t.TempDir()
	rcFile := filepath.Join(homeDir, ".zshrc")
	t.Setenv("PATH", "/usr/bin:/bin")
	t.Setenv("SHELL", "/bin/zsh")
	postInstallHasTTY = func(*os.File) bool { return true }
	postInstallReadReply = func(io.Reader) (string, error) { return "yes\n", nil }

	var stdout bytes.Buffer
	out := cli.NewOutput(&stdout, &stdout)
	err := RunPostInstall(context.Background(), runtimecfg.Config{
		OS:         "darwin",
		HomeDir:    homeDir,
		UserBinDir: filepath.Join(homeDir, "bin"),
	}, out, PostInstallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "export PATH=\"$HOME/bin:$PATH\"") {
		t.Fatalf("unexpected rc file content: %s", string(data))
	}
}

func TestRunPostInstallInteractiveReadErrorReturnsFailure(t *testing.T) {
	restorePostInstallHooks()
	defer restorePostInstallHooks()
	homeDir := t.TempDir()
	t.Setenv("PATH", "/usr/bin:/bin")
	t.Setenv("SHELL", "/bin/zsh")
	postInstallHasTTY = func(*os.File) bool { return true }
	postInstallReadReply = func(io.Reader) (string, error) { return "", errors.New("read failed") }

	var stdout bytes.Buffer
	out := cli.NewOutput(&stdout, &stdout)
	err := RunPostInstall(context.Background(), runtimecfg.Config{
		OS:         "darwin",
		HomeDir:    homeDir,
		UserBinDir: filepath.Join(homeDir, "bin"),
	}, out, PostInstallOptions{})
	if err == nil {
		t.Fatal("expected read error")
	}
}

func restorePostInstallHooks() {
	postInstallHasTTY = hasTTY
	postInstallReadReply = func(r io.Reader) (string, error) {
		return bufio.NewReader(r).ReadString('\n')
	}
	postInstallStdin = os.Stdin
}

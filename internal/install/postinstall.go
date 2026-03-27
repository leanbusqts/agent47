package install

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/platform"
	"github.com/leanbusqts/agent47/internal/runtime"
)

var (
	postInstallHasTTY = hasTTY
	postInstallReadReply = func(r io.Reader) (string, error) {
		return bufio.NewReader(r).ReadString('\n')
	}
	postInstallStdin io.Reader = os.Stdin
)

type PostInstallOptions struct {
	NonInteractive bool
	SkipPathCheck  bool
}

func RunPostInstall(_ context.Context, cfg runtime.Config, out cli.Output, opts PostInstallOptions) error {
	if cfg.OS == "windows" {
		return runWindowsPostInstall(cfg, out, opts)
	}

	if PathContains(cfg.OS, cfg.UserBinDir) {
		out.OK("~/bin in PATH")
		out.OK("afs installed")
		out.Info("Run: afs doctor")
		return nil
	}

	out.Warn("~/bin not in PATH")
	out.Info("Add this to your shell config:")
	out.Printf("export PATH=\"$HOME/bin:$PATH\"\n")

	rcFile := detectShellRCFile(os.Getenv("SHELL"), cfg.HomeDir)
	if rcFile != "" {
		out.Info("Suggested shell rc file: %s", rcFile)
	}

	if opts.NonInteractive || !postInstallHasTTY(os.Stdin) || !postInstallHasTTY(os.Stdout) {
		out.Warn("Non-interactive install; skipping shell rc update")
		out.OK("afs installed")
		out.Info("Run: afs doctor")
		return nil
	}

	out.Printf("Add it now? [y/N]: ")
	reply, err := postInstallReadReply(postInstallStdin)
	if err != nil && err.Error() != "EOF" {
		return err
	}

	switch strings.TrimSpace(strings.ToLower(reply)) {
	case "y", "yes":
		if rcFile == "" {
			return fmt.Errorf("could not determine shell rc file")
		}
		if err := appendPathExport(rcFile, out); err != nil {
			return err
		}
	default:
		out.Warn("Skipped adding to %s; update manually if needed", rcFile)
	}

	out.OK("afs installed")
	out.Info("Run: afs doctor")
	return nil
}

func detectShellRCFile(shellPath, homeDir string) string {
	shellName := filepath.Base(shellPath)
	switch shellName {
	case "zsh":
		return filepath.Join(homeDir, ".zshrc")
	case "bash":
		bashProfile := filepath.Join(homeDir, ".bash_profile")
		if _, err := os.Stat(bashProfile); err == nil {
			return bashProfile
		}
		if platform.OS() == "darwin" {
			return bashProfile
		}
		bashrc := filepath.Join(homeDir, ".bashrc")
		if _, err := os.Stat(bashrc); err == nil {
			return bashrc
		}
		return filepath.Join(homeDir, ".profile")
	default:
		return filepath.Join(homeDir, ".profile")
	}
}

func appendPathExport(rcFile string, out cli.Output) error {
	exportLine := "export PATH=\"$HOME/bin:$PATH\""
	if data, err := os.ReadFile(rcFile); err == nil && strings.Contains(string(data), exportLine) {
		out.OK("~/bin export already present in %s", rcFile)
		return nil
	}

	if info, err := os.Stat(rcFile); err == nil && !info.IsDir() {
		backup := rcFile + ".bak"
		data, readErr := os.ReadFile(rcFile)
		if readErr != nil {
			return readErr
		}
		if writeErr := os.WriteFile(backup, data, info.Mode()); writeErr != nil {
			return writeErr
		}
		out.Info("Backup created: %s", backup)
	}

	file, err := os.OpenFile(rcFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := fmt.Fprintln(file, exportLine); err != nil {
		return err
	}
	out.OK("Added to %s", rcFile)
	return nil
}

func hasTTY(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && (info.Mode()&os.ModeCharDevice) != 0
}

func runWindowsPostInstall(cfg runtime.Config, out cli.Output, opts PostInstallOptions) error {
	if opts.SkipPathCheck {
		out.OK("afs installed")
		out.Info("Run: afs doctor")
		return nil
	}

	if PathContains(cfg.OS, cfg.UserBinDir) {
		out.OK("managed bin in PATH")
		out.OK("afs installed")
		out.Info("Run: afs doctor")
		return nil
	}

	out.Warn("managed bin not in PATH")
	out.Info("Add this directory to your user PATH:")
	out.Printf("%s\n", cfg.UserBinDir)
	out.OK("afs installed")
	out.Info("Run: afs doctor")
	return nil
}

func PathContains(osName, dir string) bool {
	cleanDir := filepath.Clean(dir)
	for _, item := range filepath.SplitList(os.Getenv("PATH")) {
		if samePath(osName, item, cleanDir) {
			return true
		}
	}
	return false
}

func samePath(osName, left, right string) bool {
	leftClean := filepath.Clean(left)
	rightClean := filepath.Clean(right)
	if osName == "windows" {
		return strings.EqualFold(leftClean, rightClean)
	}
	return leftClean == rightClean
}

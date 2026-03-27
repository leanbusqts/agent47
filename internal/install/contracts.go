package install

import (
	"fmt"

	"github.com/leanbusqts/agent47/internal/runtime"
)

var helperCommands = []string{"add-agent", "add-agent-prompt", "add-ss-prompt"}

func HelperCommands() []string {
	return append([]string(nil), helperCommands...)
}

func ReinstallHint(cfg runtime.Config) string {
	if cfg.OS == "windows" {
		return "Fix: rerun install.ps1 or add the managed bin to PATH"
	}
	return "Fix: rerun install.sh or reinstall agent47"
}

func UpdateInstructions(cfg runtime.Config) string {
	if cfg.OS == "windows" {
		if cfg.RepoRoot != "" {
			return fmt.Sprintf("Update via: git -C \"%s\" pull and rerun install.ps1", cfg.RepoRoot)
		}
		return "Update via: re-download agent47 and rerun install.ps1"
	}
	if cfg.RepoRoot != "" {
		return fmt.Sprintf("Update via: git -C \"%s\" pull && ./install.sh", cfg.RepoRoot)
	}
	return "Update via: re-download agent47 and rerun install.sh"
}

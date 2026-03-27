package runtime

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/leanbusqts/agent47/internal/platform"
	"github.com/leanbusqts/agent47/internal/version"
)

type TemplateMode string

const (
	TemplateModeEmbedded   TemplateMode = "embedded"
	TemplateModeFilesystem TemplateMode = "filesystem"
)

type Config struct {
	OS              string
	HomeDir         string
	UserBinDir      string
	Agent47Home     string
	CacheDir        string
	UpdateCacheFile string
	Version         string
	TemplateMode    TemplateMode
	RepoRoot        string
	ExecutablePath  string
}

var (
	runtimeOS        = platform.OS
	runtimeIsWindows = platform.IsWindows
)

func DetectConfig(executablePath string) (Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}

	localAppData := os.Getenv("LOCALAPPDATA")
	agent47Home := os.Getenv("AGENT47_HOME")
	if agent47Home == "" {
		if runtimeIsWindows() {
			if localAppData == "" {
				localAppData = homeDir
			}
			agent47Home = filepath.Join(localAppData, "agent47")
		} else {
			agent47Home = filepath.Join(homeDir, ".agent47")
		}
	}

	userBinDir := filepath.Join(homeDir, "bin")
	if runtimeIsWindows() {
		userBinDir = filepath.Join(agent47Home, "bin")
	}

	agent47Home, err = validateAgent47Home(homeDir, localAppData, agent47Home, userBinDir, runtimeIsWindows())
	if err != nil {
		return Config{}, err
	}

	absExecutablePath, err := filepath.Abs(executablePath)
	if err != nil {
		return Config{}, err
	}

	repoRoot := detectRepoRoot(absExecutablePath)
	templateMode := detectTemplateMode(repoRoot)

	return Config{
		OS:              runtimeOS(),
		HomeDir:         homeDir,
		UserBinDir:      userBinDir,
		Agent47Home:     agent47Home,
		CacheDir:        filepath.Join(agent47Home, "cache"),
		UpdateCacheFile: filepath.Join(agent47Home, "cache", "update.cache"),
		Version:         version.Current(repoRoot, agent47Home),
		TemplateMode:    templateMode,
		RepoRoot:        repoRoot,
		ExecutablePath:  absExecutablePath,
	}, nil
}

func detectTemplateMode(repoRoot string) TemplateMode {
	switch os.Getenv("AGENT47_TEMPLATE_SOURCE") {
	case string(TemplateModeEmbedded):
		return TemplateModeEmbedded
	case string(TemplateModeFilesystem):
		return TemplateModeFilesystem
	}

	if repoRoot != "" {
		return TemplateModeFilesystem
	}

	return TemplateModeEmbedded
}

func detectRepoRoot(executablePath string) string {
	if repoRoot := os.Getenv("AGENT47_REPO_ROOT"); repoRoot != "" && looksLikeRepoRoot(repoRoot) {
		return repoRoot
	}

	current := filepath.Dir(executablePath)
	for {
		if looksLikeRepoRoot(current) {
			return current
		}

		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func looksLikeRepoRoot(root string) bool {
	if root == "" {
		return false
	}

	manifestPath := filepath.Join(root, "templates", "manifest.txt")
	_, manifestErr := os.Stat(manifestPath)
	if manifestErr != nil {
		return false
	}

	agentsPath := filepath.Join(root, "AGENTS.md")
	_, agentsErr := os.Stat(agentsPath)
	return agentsErr == nil
}

func validateAgent47Home(homeDir, localAppData, agent47Home, userBinDir string, windows bool) (string, error) {
	cleanHome, err := filepath.Abs(homeDir)
	if err != nil {
		return "", err
	}
	cleanAgentHome, err := filepath.Abs(agent47Home)
	if err != nil {
		return "", err
	}
	cleanUserBin, err := filepath.Abs(userBinDir)
	if err != nil {
		return "", err
	}
	cleanLocalAppData := ""
	if localAppData != "" {
		cleanLocalAppData, err = filepath.Abs(localAppData)
		if err != nil {
			return "", err
		}
	}

	switch {
	case sameRuntimePath(windows, cleanAgentHome, cleanHome):
		return "", fmt.Errorf("unsafe AGENT47_HOME: runtime home cannot be the same as HOME")
	case sameRuntimePath(windows, cleanAgentHome, cleanUserBin):
		return "", fmt.Errorf("unsafe AGENT47_HOME: runtime home cannot be the same as the published bin directory")
	case !windows && sameRuntimePath(windows, filepath.Join(cleanAgentHome, "bin"), cleanUserBin):
		return "", fmt.Errorf("unsafe AGENT47_HOME: managed bin would collide with the published bin directory")
	case windows && cleanLocalAppData != "" && sameRuntimePath(windows, cleanAgentHome, cleanLocalAppData):
		return "", fmt.Errorf("unsafe AGENT47_HOME: runtime home cannot be the same as LOCALAPPDATA")
	}

	return cleanAgentHome, nil
}

func sameRuntimePath(windows bool, left, right string) bool {
	leftClean := filepath.Clean(left)
	rightClean := filepath.Clean(right)
	if windows {
		return equalFold(leftClean, rightClean)
	}
	return leftClean == rightClean
}

func equalFold(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		l := left[i]
		r := right[i]
		if 'A' <= l && l <= 'Z' {
			l += 'a' - 'A'
		}
		if 'A' <= r && r <= 'Z' {
			r += 'a' - 'A'
		}
		if l != r {
			return false
		}
	}
	return true
}

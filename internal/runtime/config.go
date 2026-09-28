package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	repoRoot, explicitRepoRoot := detectRepoRoot(absExecutablePath)
	templateMode := detectTemplateMode(repoRoot)
	versionRepoRoot := ""
	if explicitRepoRoot || shouldUseRepoVersion(absExecutablePath, repoRoot) {
		versionRepoRoot = repoRoot
	}

	return Config{
		OS:              runtimeOS(),
		HomeDir:         homeDir,
		UserBinDir:      userBinDir,
		Agent47Home:     agent47Home,
		CacheDir:        filepath.Join(agent47Home, "cache"),
		UpdateCacheFile: filepath.Join(agent47Home, "cache", "update.cache"),
		Version:         version.Current(versionRepoRoot, agent47Home),
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

func detectRepoRoot(executablePath string) (string, bool) {
	if repoRoot := os.Getenv("AGENT47_REPO_ROOT"); repoRoot != "" && looksLikeRepoRoot(repoRoot) {
		return repoRoot, true
	}

	current := filepath.Dir(executablePath)
	for {
		if looksLikeRepoRoot(current) {
			return current, false
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}

func shouldUseRepoVersion(executablePath, repoRoot string) bool {
	if repoRoot == "" {
		return false
	}

	rel, err := filepath.Rel(repoRoot, executablePath)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	if rel == "." {
		return true
	}
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return false
	}
	return true
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
	cleanAgentHome, err := ValidateAgent47HomePath(homeDir, agent47Home, userBinDir, windows)
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

	if windows && cleanLocalAppData != "" && pathContains(windows, cleanAgentHome, cleanLocalAppData) {
		return "", fmt.Errorf("unsafe AGENT47_HOME: runtime home cannot be the same as LOCALAPPDATA")
	}

	return cleanAgentHome, nil
}

// ValidateAgent47HomePath constrains the runtime to a dedicated directory and
// rejects existing symlink components before lifecycle code performs any IO.
func ValidateAgent47HomePath(homeDir, agent47Home, userBinDir string, windows bool) (string, error) {
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

	volumeRoot := filepath.Clean(filepath.VolumeName(cleanAgentHome) + string(filepath.Separator))
	switch {
	case sameRuntimePath(windows, cleanAgentHome, volumeRoot):
		return "", fmt.Errorf("unsafe AGENT47_HOME: runtime home cannot be a filesystem root")
	case sameRuntimePath(windows, filepath.Dir(cleanAgentHome), volumeRoot):
		return "", fmt.Errorf("unsafe AGENT47_HOME: runtime home must be a dedicated nested directory")
	case pathContains(windows, cleanAgentHome, cleanHome):
		return "", fmt.Errorf("unsafe AGENT47_HOME: runtime home cannot contain HOME")
	case sameRuntimePath(windows, cleanAgentHome, cleanUserBin):
		return "", fmt.Errorf("unsafe AGENT47_HOME: runtime home cannot be the same as the published bin directory")
	case !windows && pathContains(windows, cleanAgentHome, cleanUserBin):
		return "", fmt.Errorf("unsafe AGENT47_HOME: runtime home cannot contain the published bin directory")
	}

	agentAnchor := filepath.Dir(cleanAgentHome)
	if pathContains(windows, cleanHome, cleanAgentHome) {
		agentAnchor = cleanHome
	}
	if err := rejectSymlinkComponents(agentAnchor, cleanAgentHome); err != nil {
		return "", fmt.Errorf("unsafe AGENT47_HOME: %w", err)
	}
	userBinAnchor := filepath.Dir(cleanUserBin)
	if pathContains(windows, cleanHome, cleanUserBin) {
		userBinAnchor = cleanHome
	} else if pathContains(windows, cleanAgentHome, cleanUserBin) {
		userBinAnchor = cleanAgentHome
	}
	if err := rejectSymlinkComponents(userBinAnchor, cleanUserBin); err != nil {
		return "", fmt.Errorf("unsafe published bin directory: %w", err)
	}

	return cleanAgentHome, nil
}

func rejectSymlinkComponents(anchor, path string) error {
	cleanAnchor := filepath.Clean(anchor)
	cleanPath := filepath.Clean(path)
	rel, err := filepath.Rel(cleanAnchor, cleanPath)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("path escapes validation anchor: %s", cleanAnchor)
	}

	current := cleanAnchor
	if info, statErr := os.Lstat(current); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path component is a symlink: %s", current)
		}
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	for _, component := range strings.Split(rel, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				return nil
			}
			return statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path component is a symlink: %s", current)
		}
	}
	return nil
}

func pathContains(windows bool, parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	if rel == "." {
		return true
	}
	if filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	if windows {
		return !strings.HasPrefix(strings.ToLower(rel), ".."+string(filepath.Separator))
	}
	return true
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

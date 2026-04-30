package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	goRuntime "runtime"
	"sort"
	"strings"

	"github.com/leanbusqts/agent47/internal/testutil"
)

var (
	afsverifyOS              = goRuntime.GOOS
	afsverifyDetectRepoRoot  = detectRepoRoot
	afsverifyNewInstalledEnv = newInstalledEnv
	afsverifyScenarios       = func() []scenario {
		return []scenario{
			{name: "install", run: verifyFreshInstall},
			{name: "help-doctor", run: verifyInstalledHelpAndDoctor},
			{name: "doctor-update-check", run: verifyInstalledDoctorUpdateCheck},
			{name: "add-agent", run: verifyInstalledAddAgent},
			{name: "add-agent-force", run: verifyInstalledAddAgentForce},
			{name: "add-agent-only-skills", run: verifyInstalledOnlySkills},
			{name: "add-agent-only-skills-force", run: verifyInstalledOnlySkillsForce},
			{name: "add-agent-prompt", run: verifyInstalledAddAgentPrompt},
			{name: "add-ss-prompt", run: verifyInstalledAddSSPrompt},
			{name: "reinstall-force", run: verifyForceReinstall},
			{name: "uninstall", run: verifyUninstallCleanup},
		}
	}
	afsverifyStdout io.Writer = os.Stdout
	afsverifyStderr io.Writer = os.Stderr
)

type scenario struct {
	name string
	run  func(*installedEnv) error
}

type installedEnv struct {
	repoRoot     string
	tempRoot     string
	homeDir      string
	agentHome    string
	userBinDir   string
	localAppData string
	baseEnv      []string
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	repoRoot, err := afsverifyDetectRepoRoot()
	if err != nil {
		fmt.Fprintf(afsverifyStderr, "[ERR] %v\n", err)
		return 1
	}

	scenarios := afsverifyScenarios()

	selected, err := selectScenarios(scenarios, args)
	if err != nil {
		fmt.Fprintf(afsverifyStderr, "[ERR] %v\n", err)
		return 1
	}

	for _, sc := range selected {
		fmt.Fprintf(afsverifyStdout, "[INFO] Scenario: %s\n", sc.name)
		env, err := afsverifyNewInstalledEnv(repoRoot)
		if err != nil {
			fmt.Fprintf(afsverifyStderr, "[ERR] failed to prepare installed test env for %s: %v\n", sc.name, err)
			return 1
		}
		if err := sc.run(env); err != nil {
			fmt.Fprintf(afsverifyStderr, "[ERR] scenario %s failed: %v\n", sc.name, err)
			fmt.Fprintf(afsverifyStderr, "[INFO] preserving failed scenario root: %s\n", env.tempRoot)
			return 1
		}
		if err := env.cleanup(); err != nil {
			fmt.Fprintf(afsverifyStderr, "[ERR] failed to clean scenario root %s: %v\n", env.tempRoot, err)
			return 1
		}
		fmt.Fprintf(afsverifyStdout, "[OK] Scenario passed: %s\n", sc.name)
	}

	fmt.Fprintf(afsverifyStdout, "[OK] installed artifact verification passed (%d scenarios)\n", len(selected))
	return 0
}

func newInstalledEnv(repoRoot string) (*installedEnv, error) {
	tempRoot, err := os.MkdirTemp(os.TempDir(), "afs-installed-")
	if err != nil {
		return nil, err
	}

	homeDir := filepath.Join(tempRoot, "home")
	localAppData := filepath.Join(homeDir, "AppData", "Local")
	agentHome := filepath.Join(homeDir, ".agent47")
	userBinDir := filepath.Join(homeDir, "bin")
	if afsverifyOS == "windows" {
		agentHome = filepath.Join(localAppData, "agent47")
		userBinDir = filepath.Join(agentHome, "bin")
	}

	if err := os.MkdirAll(userBinDir, 0o755); err != nil {
		return nil, err
	}

	baseEnv := append(os.Environ(),
		"HOME="+homeDir,
		"USERPROFILE="+homeDir,
		"LOCALAPPDATA="+localAppData,
		"AGENT47_HOME="+agentHome,
		"PATH="+isolatedPath(userBinDir),
	)

	return &installedEnv{
		repoRoot:     repoRoot,
		tempRoot:     tempRoot,
		homeDir:      homeDir,
		agentHome:    agentHome,
		userBinDir:   userBinDir,
		localAppData: localAppData,
		baseEnv:      baseEnv,
	}, nil
}

func verifyFreshInstall(env *installedEnv) error {
	if _, _, err := env.runInstall("--force", "--non-interactive"); err != nil {
		return err
	}

	for _, path := range []string{
		filepath.Join(env.agentHome, "templates"),
		filepath.Join(env.agentHome, "VERSION"),
		env.managedAfsPath(),
		env.helperPath("add-agent"),
		env.helperPath("add-agent-prompt"),
		env.helperPath("add-ss-prompt"),
	} {
		if err := assertExists(path); err != nil {
			return err
		}
	}

	if afsverifyOS != "windows" {
		if err := assertExists(filepath.Join(env.userBinDir, "afs")); err != nil {
			return err
		}
	}

	return nil
}

func verifyInstalledHelpAndDoctor(env *installedEnv) error {
	if _, _, err := env.runInstall("--force", "--non-interactive"); err != nil {
		return err
	}

	helpOut, _, err := env.runPublishedAfs("", "help")
	if err != nil {
		return err
	}
	if !strings.Contains(helpOut, "Core commands:") {
		return fmt.Errorf("installed help output missing core commands: %q", helpOut)
	}

	doctorOut, _, err := env.runPublishedAfs("", "doctor", "--fail-on-warn")
	if err != nil {
		return err
	}
	for _, marker := range []string{
		"[OK] afs in PATH",
		"[OK] Templates installed",
		"[OK] add-agent available",
		"[OK] add-agent-prompt available",
		"[OK] add-ss-prompt available",
	} {
		if !strings.Contains(doctorOut, marker) {
			return fmt.Errorf("doctor output missing %q: %q", marker, doctorOut)
		}
	}

	return nil
}

func verifyInstalledDoctorUpdateCheck(env *installedEnv) error {
	if _, _, err := env.runInstall("--force", "--non-interactive"); err != nil {
		return err
	}

	localVersionBytes, err := os.ReadFile(filepath.Join(env.agentHome, "VERSION"))
	if err != nil {
		return err
	}
	remoteVersionFile := filepath.Join(env.tempRoot, "remote-VERSION")
	if err := os.WriteFile(remoteVersionFile, localVersionBytes, 0o644); err != nil {
		return err
	}

	originalEnv := env.baseEnv
	env.baseEnv = withEnv(env.baseEnv, "AGENT47_VERSION_URL=file://"+remoteVersionFile)
	defer func() {
		env.baseEnv = originalEnv
	}()

	for _, args := range [][]string{
		{"doctor", "--check-update", "--fail-on-warn"},
		{"doctor", "--check-update-force", "--fail-on-warn"},
	} {
		output, _, err := env.runPublishedAfs("", args...)
		if err != nil {
			return err
		}
		if !strings.Contains(output, "Up to date") && !strings.Contains(output, "Git tracking reference is current") {
			return fmt.Errorf("unexpected doctor update-check output: %q", output)
		}
	}

	return nil
}

func verifyInstalledAddAgent(env *installedEnv) error {
	if _, _, err := env.runInstall("--force", "--non-interactive"); err != nil {
		return err
	}

	projectDir := filepath.Join(env.tempRoot, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return err
	}

	if _, _, err := env.runCommand(env.helperPath("add-agent"), projectDir); err != nil {
		return err
	}

	for _, path := range []string{
		filepath.Join(projectDir, "AGENTS.md"),
		filepath.Join(projectDir, "README.md"),
		filepath.Join(projectDir, "rules"),
		filepath.Join(projectDir, "skills", "AVAILABLE_SKILLS.xml"),
	} {
		if err := assertExists(path); err != nil {
			return err
		}
	}

	return nil
}

func verifyInstalledAddAgentForce(env *installedEnv) error {
	if err := verifyInstalledAddAgent(env); err != nil {
		return err
	}

	projectDir := filepath.Join(env.tempRoot, "project")
	if err := os.WriteFile(filepath.Join(projectDir, "AGENTS.md"), []byte("stale-agents\n"), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(projectDir, "rules", "stale.yaml"), []byte("stale\n"), 0o644); err != nil {
		return err
	}

	if _, _, err := env.runCommand(env.helperPath("add-agent"), projectDir, "--force"); err != nil {
		return err
	}

	agentsData, err := os.ReadFile(filepath.Join(projectDir, "AGENTS.md"))
	if err != nil {
		return err
	}
	if strings.Contains(string(agentsData), "stale-agents") {
		return fmt.Errorf("expected force run to refresh AGENTS.md")
	}
	return assertNotExists(filepath.Join(projectDir, "rules", "stale.yaml"))
}

func verifyInstalledOnlySkills(env *installedEnv) error {
	if _, _, err := env.runInstall("--force", "--non-interactive"); err != nil {
		return err
	}

	projectDir := filepath.Join(env.tempRoot, "skills-project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return err
	}

	if _, _, err := env.runPublishedAfs(projectDir, "add-agent", "--only-skills"); err != nil {
		return err
	}

	if err := assertExists(filepath.Join(projectDir, "skills", "AVAILABLE_SKILLS.xml")); err != nil {
		return err
	}
	if err := assertNotExists(filepath.Join(projectDir, "AGENTS.md")); err != nil {
		return err
	}
	if err := assertNotExists(filepath.Join(projectDir, "rules")); err != nil {
		return err
	}

	return nil
}

func verifyInstalledOnlySkillsForce(env *installedEnv) error {
	if err := verifyInstalledOnlySkills(env); err != nil {
		return err
	}

	projectDir := filepath.Join(env.tempRoot, "skills-project")
	customSkillDir := filepath.Join(projectDir, "skills", "custom")
	if err := os.MkdirAll(customSkillDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(customSkillDir, "SKILL.md"), []byte("custom\n"), 0o644); err != nil {
		return err
	}

	if _, _, err := env.runPublishedAfs(projectDir, "add-agent", "--only-skills", "--force"); err != nil {
		return err
	}

	if err := assertExists(filepath.Join(projectDir, "skills", "AVAILABLE_SKILLS.xml")); err != nil {
		return err
	}
	return assertNotExists(customSkillDir)
}

func verifyInstalledAddAgentPrompt(env *installedEnv) error {
	if _, _, err := env.runInstall("--force", "--non-interactive"); err != nil {
		return err
	}

	projectDir := filepath.Join(env.tempRoot, "prompt-project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return err
	}

	if _, _, err := env.runCommand(env.helperPath("add-agent-prompt"), projectDir); err != nil {
		return err
	}

	return assertExists(filepath.Join(projectDir, "prompts", "agent-prompt.txt"))
}

func verifyInstalledAddSSPrompt(env *installedEnv) error {
	if _, _, err := env.runInstall("--force", "--non-interactive"); err != nil {
		return err
	}

	clipboardFile := filepath.Join(env.tempRoot, "clipboard.txt")
	clipboardBin := filepath.Join(env.tempRoot, "clipboard-bin")
	if err := os.MkdirAll(clipboardBin, 0o755); err != nil {
		return err
	}
	if err := installMockClipboardTool(clipboardBin, clipboardFile); err != nil {
		return err
	}

	originalEnv := env.baseEnv
	env.baseEnv = withEnv(env.baseEnv,
		"PATH="+clipboardBin+string(os.PathListSeparator)+"/bin",
		"TARGET_FILE="+clipboardFile,
	)
	defer func() {
		env.baseEnv = originalEnv
	}()

	output, _, err := env.runCommand(env.helperPath("add-ss-prompt"), env.tempRoot)
	if err != nil {
		return err
	}

	if strings.Contains(output, "Snapshot/spec prompt copied to clipboard") {
		data, readErr := os.ReadFile(clipboardFile)
		if readErr != nil {
			return fmt.Errorf("expected clipboard capture file: %w", readErr)
		}
		if !strings.Contains(string(data), "Review this repository") {
			return fmt.Errorf("unexpected clipboard content: %q", string(data))
		}
		return nil
	}
	if !strings.Contains(output, "Review this repository") {
		return fmt.Errorf("unexpected add-ss-prompt output: %q", output)
	}
	return nil
}

func verifyForceReinstall(env *installedEnv) error {
	if _, _, err := env.runInstall("--force", "--non-interactive"); err != nil {
		return err
	}
	if _, _, err := env.runInstall("--force", "--non-interactive"); err != nil {
		return err
	}

	doctorOut, _, err := env.runPublishedAfs("", "doctor", "--fail-on-warn")
	if err != nil {
		return err
	}
	if !strings.Contains(doctorOut, "[OK] Templates installed") {
		return fmt.Errorf("doctor output missing install confirmation after force reinstall: %q", doctorOut)
	}
	return nil
}

func verifyUninstallCleanup(env *installedEnv) error {
	if _, _, err := env.runInstall("--force", "--non-interactive"); err != nil {
		return err
	}
	if _, _, err := env.runPublishedAfs("", "uninstall"); err != nil {
		return err
	}

	if err := assertNotExists(env.agentHome); err != nil {
		return err
	}
	for _, path := range []string{
		env.helperPath("add-agent"),
		env.helperPath("add-agent-prompt"),
		env.helperPath("add-ss-prompt"),
	} {
		if err := assertNotExists(path); err != nil {
			return err
		}
	}
	if afsverifyOS != "windows" {
		if err := assertNotExists(filepath.Join(env.userBinDir, "afs")); err != nil {
			return err
		}
	}
	return nil
}

func (env *installedEnv) runInstall(args ...string) (string, string, error) {
	cmd := env.installCommand(args...)
	cmd.Env = env.baseEnv
	cmd.Dir = env.repoRoot
	return runCombined(cmd)
}

func (env *installedEnv) runPublishedAfs(workDir string, args ...string) (string, string, error) {
	return env.runCommand(env.publishedAfsPath(), workDir, args...)
}

func (env *installedEnv) runCommand(path, workDir string, args ...string) (string, string, error) {
	var cmd *exec.Cmd
	if afsverifyOS == "windows" && strings.EqualFold(filepath.Ext(path), ".cmd") {
		cmdArgs := append([]string{"/c", path}, args...)
		cmd = exec.Command("cmd", cmdArgs...)
	} else {
		cmd = exec.Command(path, args...)
	}
	cmd.Env = env.baseEnv
	cmd.Dir = workDir
	return runCombined(cmd)
}

func (env *installedEnv) installCommand(args ...string) *exec.Cmd {
	if afsverifyOS == "windows" {
		baseArgs := []string{
			"-NoProfile",
			"-ExecutionPolicy",
			"Bypass",
			"-File",
			filepath.Join(env.repoRoot, "install.ps1"),
		}
		return exec.Command("powershell", append(baseArgs, toPowerShellArgs(args)...)...)
	}
	return exec.Command(filepath.Join(env.repoRoot, "install.sh"), args...)
}

func (env *installedEnv) publishedAfsPath() string {
	if afsverifyOS == "windows" {
		return filepath.Join(env.userBinDir, "afs.exe")
	}
	return filepath.Join(env.userBinDir, "afs")
}

func (env *installedEnv) managedAfsPath() string {
	if afsverifyOS == "windows" {
		return filepath.Join(env.agentHome, "bin", "afs.exe")
	}
	return filepath.Join(env.agentHome, "bin", "afs")
}

func (env *installedEnv) helperPath(name string) string {
	if afsverifyOS == "windows" {
		return filepath.Join(env.userBinDir, name+".cmd")
	}
	return filepath.Join(env.userBinDir, name)
}

func detectRepoRoot() (string, error) {
	return testutil.DetectRepoRoot()
}

func selectScenarios(all []scenario, args []string) ([]scenario, error) {
	if len(args) == 0 {
		return all, nil
	}

	index := map[string]scenario{}
	for _, sc := range all {
		index[sc.name] = sc
	}

	var selected []scenario
	for _, arg := range args {
		sc, ok := index[arg]
		if !ok {
			var names []string
			for _, candidate := range all {
				names = append(names, candidate.name)
			}
			sort.Strings(names)
			return nil, fmt.Errorf("unknown scenario %q (available: %s)", arg, strings.Join(names, ", "))
		}
		selected = append(selected, sc)
	}
	return selected, nil
}

func toPowerShellArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		switch arg {
		case "--force":
			out = append(out, "-Force")
		case "--non-interactive":
			out = append(out, "-NonInteractive")
		default:
			out = append(out, arg)
		}
	}
	return out
}

func runCombined(cmd *exec.Cmd) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("%w\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	return stdout.String(), stderr.String(), nil
}

func installMockClipboardTool(binDir, targetFile string) error {
	if afsverifyOS == "windows" {
		path := filepath.Join(binDir, "clip.cmd")
		body := "@echo off\r\nmore > \"%TARGET_FILE%\"\r\n"
		return os.WriteFile(path, []byte(body), 0o755)
	}

	for _, name := range []string{"pbcopy", "wl-copy"} {
		path := filepath.Join(binDir, name)
		body := "#!/bin/sh\ncat > \"$TARGET_FILE\"\n"
		if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
			return err
		}
	}
	_ = targetFile
	return nil
}

func withEnv(base []string, values ...string) []string {
	result := append([]string{}, base...)
	for _, value := range values {
		key, _, ok := strings.Cut(value, "=")
		if !ok {
			result = append(result, value)
			continue
		}

		filtered := result[:0]
		for _, existing := range result {
			existingKey, _, existingOK := strings.Cut(existing, "=")
			if existingOK && existingKey == key {
				continue
			}
			filtered = append(filtered, existing)
		}
		result = append(filtered, value)
	}
	return result
}

func getEnvValue(env []string, key string) string {
	for i := len(env) - 1; i >= 0; i-- {
		item := env[i]
		existingKey, value, ok := strings.Cut(item, "=")
		if ok && existingKey == key {
			return value
		}
	}
	return ""
}

func isolatedPath(userBinDir string) string {
	var segments []string
	seen := map[string]bool{}
	add := func(path string) {
		if path == "" {
			return
		}
		cleaned := filepath.Clean(path)
		if !seen[cleaned] {
			seen[cleaned] = true
			segments = append(segments, cleaned)
		}
	}

	add(userBinDir)
	for _, name := range []string{"go", "git", "powershell", "cmd"} {
		if toolPath, err := exec.LookPath(name); err == nil {
			add(filepath.Dir(toolPath))
		}
	}

	if afsverifyOS == "windows" {
		systemRoot := os.Getenv("SystemRoot")
		add(filepath.Join(systemRoot, "System32"))
		add(filepath.Join(systemRoot, "System32", "WindowsPowerShell", "v1.0"))
		add(systemRoot)
	} else {
		add("/usr/bin")
		add("/bin")
		add("/usr/sbin")
		add("/sbin")
	}

	return strings.Join(segments, string(os.PathListSeparator))
}

func assertExists(path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("expected path to exist: %s", path)
	}
	return nil
}

func assertNotExists(path string) error {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return fmt.Errorf("expected path to not exist: %s", path)
	}
	return nil
}

func (env *installedEnv) cleanup() error {
	return os.RemoveAll(env.tempRoot)
}

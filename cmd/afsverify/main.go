package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	goRuntime "runtime"
	"sort"
	"strings"
	"time"

	"github.com/leanbusqts/agent47/internal/testutil"
)

var (
	afsverifyOS              = goRuntime.GOOS
	afsverifyDetectRepoRoot  = testutil.DetectRepoRoot
	afsverifyNewInstalledEnv = newInstalledEnv
	afsverifyScenarios       = func() []scenario {
		return []scenario{
			{name: "install", run: verifyFreshInstall},
			{name: "help-doctor", run: verifyInstalledHelpAndDoctor},
			{name: "analyze-deep", run: verifyInstalledAnalyzeDeep},
			{name: "map", run: verifyInstalledMap},
			{name: "init", run: verifyInstalledInit},
			{name: "init-force-cleans-legacy", run: verifyInstalledInitForceCleansLegacy},
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
	repoRoot   string
	tempRoot   string
	homeDir    string
	agentHome  string
	userBinDir string
	baseEnv    []string
}

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	repoRoot, err := afsverifyDetectRepoRoot()
	if err != nil {
		fmt.Fprintf(afsverifyStderr, "[ERR] %v\n", err)
		return 1
	}
	selected, err := selectScenarios(afsverifyScenarios(), args)
	if err != nil {
		fmt.Fprintf(afsverifyStderr, "[ERR] %v\n", err)
		return 1
	}
	for _, item := range selected {
		fmt.Fprintf(afsverifyStdout, "[INFO] Scenario: %s\n", item.name)
		env, err := afsverifyNewInstalledEnv(repoRoot)
		if err != nil {
			fmt.Fprintf(afsverifyStderr, "[ERR] prepare %s: %v\n", item.name, err)
			return 1
		}
		if err := item.run(env); err != nil {
			fmt.Fprintf(afsverifyStderr, "[ERR] scenario %s failed: %v\n", item.name, err)
			fmt.Fprintf(afsverifyStderr, "[INFO] preserving failed scenario root: %s\n", env.tempRoot)
			return 1
		}
		if err := env.cleanup(); err != nil {
			fmt.Fprintf(afsverifyStderr, "[ERR] cleanup %s: %v\n", item.name, err)
			return 1
		}
		fmt.Fprintf(afsverifyStdout, "[OK] Scenario passed: %s\n", item.name)
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
	if afsverifyOS != "windows" {
		if err := os.MkdirAll(userBinDir, 0o755); err != nil {
			return nil, err
		}
	}
	return &installedEnv{
		repoRoot: repoRoot, tempRoot: tempRoot, homeDir: homeDir,
		agentHome: agentHome, userBinDir: userBinDir,
		baseEnv: append(os.Environ(),
			"HOME="+homeDir,
			"USERPROFILE="+homeDir,
			"LOCALAPPDATA="+localAppData,
			"AGENT47_HOME="+agentHome,
			"PATH="+isolatedPath(userBinDir),
		),
	}, nil
}

func verifyFreshInstall(env *installedEnv) error {
	if _, _, err := env.runInstall("--force", "--non-interactive"); err != nil {
		return err
	}
	for _, path := range []string{filepath.Join(env.agentHome, "templates"), filepath.Join(env.agentHome, "VERSION"), env.managedAfsPath(), env.publishedAfsPath()} {
		if err := assertExists(path); err != nil {
			return err
		}
	}
	for _, name := range []string{"add-agent", "add-agent-prompt", "add-ss-prompt"} {
		if err := assertNotExists(env.legacyHelperPath(name)); err != nil {
			return err
		}
	}
	return nil
}

func verifyInstalledHelpAndDoctor(env *installedEnv) error {
	if err := verifyFreshInstall(env); err != nil {
		return err
	}
	help, _, err := env.runPublishedAfs("", "help")
	if err != nil {
		return err
	}
	if !strings.Contains(help, "afs init") || !strings.Contains(help, "afs map [--force]") || strings.Contains(help, "add-agent") {
		return fmt.Errorf("unexpected installed help: %q", help)
	}
	for _, args := range [][]string{{"add-agent"}, {"doctor", "--unknown"}, {"map", "--preview"}} {
		cmd := exec.Command(env.publishedAfsPath(), args...)
		cmd.Env = env.baseEnv
		err := cmd.Run()
		exitErr, ok := err.(*exec.ExitError)
		if !ok || exitErr.ExitCode() != 2 {
			return fmt.Errorf("expected usage exit 2 for %v, got %v", args, err)
		}
	}
	doctor, _, err := env.runPublishedAfs("", "doctor", "--fail-on-warn")
	if err != nil {
		return err
	}
	if !strings.Contains(doctor, "[OK] Templates installed") {
		return fmt.Errorf("doctor did not validate templates: %q", doctor)
	}
	doctorJSON, doctorJSONErr, err := env.runPublishedAfs("", "doctor", "--json")
	if err != nil {
		return err
	}
	if strings.TrimSpace(doctorJSONErr) != "" {
		return fmt.Errorf("doctor JSON wrote diagnostics to stderr: %q", doctorJSONErr)
	}
	var report struct {
		SchemaVersion int      `json:"schema_version"`
		Status        string   `json:"status"`
		Checks        []string `json:"checks"`
		Warnings      []string `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(doctorJSON), &report); err != nil {
		return fmt.Errorf("doctor JSON is invalid: %w", err)
	}
	if report.SchemaVersion != 1 || report.Status != "ok" || len(report.Checks) == 0 || report.Warnings == nil {
		return fmt.Errorf("unexpected doctor JSON report: %+v", report)
	}
	return nil
}

func verifyInstalledAnalyzeDeep(env *installedEnv) error {
	if err := verifyFreshInstall(env); err != nil {
		return err
	}
	project := filepath.Join(env.tempRoot, "analyze-project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		return err
	}
	out, _, err := env.runPublishedAfs(project, "analyze", "--deep")
	if err != nil {
		return err
	}
	if !strings.Contains(out, "Agent readiness") {
		return fmt.Errorf("deep analysis output missing readiness section: %q", out)
	}
	return nil
}

func verifyInstalledInit(env *installedEnv) error {
	if err := verifyFreshInstall(env); err != nil {
		return err
	}
	project := filepath.Join(env.tempRoot, "init-project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		return err
	}
	if _, _, err := env.runPublishedAfs(project, "init"); err != nil {
		return err
	}
	for _, path := range []string{filepath.Join(project, "AGENTS.md"), filepath.Join(project, "rules", "security-global.yaml"), filepath.Join(project, ".agent47", "context.md")} {
		if err := assertExists(path); err != nil {
			return err
		}
	}
	for _, path := range []string{filepath.Join(project, "skills"), filepath.Join(project, "prompts"), filepath.Join(project, ".agents")} {
		if err := assertNotExists(path); err != nil {
			return err
		}
	}
	return nil
}

func verifyInstalledMap(env *installedEnv) error {
	if err := verifyFreshInstall(env); err != nil {
		return err
	}
	project := filepath.Join(env.tempRoot, "map-project")
	if err := os.MkdirAll(filepath.Join(project, "cmd", "tool"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.com/project\n"), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(project, "cmd", "tool", "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		return err
	}
	out, _, err := env.runPublishedAfs(project, "map")
	if err != nil || !strings.Contains(out, "Created .agent47/context.md") {
		return fmt.Errorf("installed map create failed: output=%q err=%v", out, err)
	}
	contextPath := filepath.Join(project, ".agent47", "context.md")
	content, err := os.ReadFile(contextPath)
	if err != nil || !bytes.Contains(content, []byte("<!-- afs-context")) {
		return fmt.Errorf("installed map context invalid: %w", err)
	}
	out, _, err = env.runPublishedAfs(project, "map")
	if err != nil || !strings.Contains(out, "is current") {
		return fmt.Errorf("installed map freshness failed: output=%q err=%v", out, err)
	}
	file, err := os.OpenFile(contextPath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	if _, err := file.WriteString("manual edit\n"); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if _, _, err := env.runPublishedAfs(project, "map"); err == nil {
		return errors.New("installed map accepted manual context edit")
	}
	_, _, err = env.runPublishedAfs(project, "map", "--force")
	return err
}

func verifyInstalledInitForceCleansLegacy(env *installedEnv) error {
	if err := verifyInstalledInit(env); err != nil {
		return err
	}
	project := filepath.Join(env.tempRoot, "init-project")
	contextPath := filepath.Join(project, ".agent47", "context.md")
	contextBefore, err := os.ReadFile(contextPath)
	if err != nil {
		return err
	}
	custom := map[string]string{
		filepath.Join(project, "rules", "custom.yaml"):         "custom rule\n",
		filepath.Join(project, "skills", "custom", "SKILL.md"): "custom skill\n",
		filepath.Join(project, "prompts", "agent-prompt.txt"):  "custom prompt\n",
		filepath.Join(project, "specs", "spec.yml"):            "legacy task\n",
		filepath.Join(project, ".agents", "specs", "spec.yml"): "newer legacy task\n",
		filepath.Join(project, "README.md"):                    "project readme\n",
		filepath.Join(project, ".agents", "keep.txt"):          "project state\n",
	}
	for path, body := range custom {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return err
		}
	}
	if _, _, err := env.runPublishedAfs(project, "init", "--force"); err != nil {
		return err
	}
	contextAfter, err := os.ReadFile(contextPath)
	if err != nil || !bytes.Equal(contextBefore, contextAfter) {
		return fmt.Errorf("init --force replaced existing generated context")
	}
	for _, path := range []string{
		filepath.Join(project, "rules", "custom.yaml"),
		filepath.Join(project, "skills"),
		filepath.Join(project, "prompts"),
		filepath.Join(project, "specs", "spec.yml"),
		filepath.Join(project, ".agents", "specs", "spec.yml"),
	} {
		if err := assertNotExists(path); err != nil {
			return err
		}
	}
	for path, want := range map[string]string{
		filepath.Join(project, "README.md"):           custom[filepath.Join(project, "README.md")],
		filepath.Join(project, ".agents", "keep.txt"): custom[filepath.Join(project, ".agents", "keep.txt")],
	} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			return fmt.Errorf("unrelated file changed: %s", path)
		}
	}
	return nil
}

func verifyForceReinstall(env *installedEnv) error {
	if _, _, err := env.runInstall("--force", "--non-interactive"); err != nil {
		return err
	}
	_, _, err := env.runInstall("--force", "--non-interactive")
	return err
}

func verifyUninstallCleanup(env *installedEnv) error {
	if err := verifyFreshInstall(env); err != nil {
		return err
	}
	if _, _, err := env.runPublishedAfs("", "uninstall"); err != nil {
		return err
	}
	if err := assertNotExists(env.agentHome); err != nil {
		return err
	}
	return assertNotExists(env.publishedAfsPath())
}

func (env *installedEnv) runInstall(args ...string) (string, string, error) {
	cmd := env.installCommand(args...)
	cmd.Env, cmd.Dir = env.baseEnv, env.repoRoot
	return runCombined(cmd)
}

func (env *installedEnv) runPublishedAfs(workDir string, args ...string) (string, string, error) {
	cmd := exec.Command(env.publishedAfsPath(), args...)
	cmd.Env, cmd.Dir = env.baseEnv, workDir
	return runCombined(cmd)
}

func (env *installedEnv) installCommand(args ...string) *exec.Cmd {
	if afsverifyOS == "windows" {
		base := []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", filepath.Join(env.repoRoot, "install.ps1")}
		return exec.Command("powershell", append(base, toPowerShellArgs(args)...)...)
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

func (env *installedEnv) legacyHelperPath(name string) string {
	if afsverifyOS == "windows" {
		return filepath.Join(env.userBinDir, name+".cmd")
	}
	return filepath.Join(env.userBinDir, name)
}

func selectScenarios(all []scenario, args []string) ([]scenario, error) {
	if len(args) == 0 {
		return all, nil
	}
	index := map[string]scenario{}
	for _, item := range all {
		index[item.name] = item
	}
	selected := make([]scenario, 0, len(args))
	for _, name := range args {
		item, ok := index[name]
		if !ok {
			names := make([]string, 0, len(all))
			for _, candidate := range all {
				names = append(names, candidate.name)
			}
			sort.Strings(names)
			return nil, fmt.Errorf("unknown scenario %q (available: %s)", name, strings.Join(names, ", "))
		}
		selected = append(selected, item)
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
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("%w\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	return stdout.String(), stderr.String(), nil
}

func isolatedPath(userBinDir string) string {
	paths := []string{userBinDir}
	for _, name := range []string{"go", "git", "powershell", "cmd"} {
		if path, err := exec.LookPath(name); err == nil {
			paths = append(paths, filepath.Dir(path))
		}
	}
	if afsverifyOS != "windows" {
		paths = append(paths, "/usr/bin", "/bin", "/usr/sbin", "/sbin")
	}
	return strings.Join(paths, string(os.PathListSeparator))
}

func assertExists(path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("expected path to exist: %s", path)
	}
	return nil
}

func assertNotExists(path string) error {
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		return fmt.Errorf("expected path to not exist: %s", path)
	}
	return nil
}

func (env *installedEnv) cleanup() error {
	var err error
	for attempt := 0; attempt < 20; attempt++ {
		if err = os.RemoveAll(env.tempRoot); err == nil {
			return nil
		}
		if afsverifyOS != "windows" {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
	return err
}

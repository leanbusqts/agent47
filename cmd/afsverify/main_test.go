package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	goRuntime "runtime"
	"strings"
	"testing"
)

func TestWithEnvOverridesExistingValues(t *testing.T) {
	base := []string{"PATH=/a", "HOME=/home"}
	got := withEnv(base, "PATH=/b", "NEW=value")
	want := []string{"PATH=/b", "HOME=/home", "NEW=value"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected env: got %v want %v", got, want)
	}
}

func TestGetEnvValueReturnsEmptyWhenMissing(t *testing.T) {
	if got := getEnvValue([]string{"PATH=/a"}, "HOME"); got != "" {
		t.Fatalf("expected empty value, got %q", got)
	}
}

func TestWithEnvAppendsRawValueWithoutEquals(t *testing.T) {
	base := []string{"PATH=/a"}
	got := withEnv(base, "RAWVALUE")
	want := []string{"PATH=/a", "RAWVALUE"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected env: got %v want %v", got, want)
	}
}

func TestSelectScenariosRejectsUnknownName(t *testing.T) {
	_, err := selectScenarios([]scenario{{name: "one"}}, []string{"missing"})
	if err == nil {
		t.Fatal("expected unknown scenario error")
	}
}

func TestSelectScenariosReturnsAllWhenArgsEmpty(t *testing.T) {
	all := []scenario{{name: "one"}, {name: "two"}}
	got, err := selectScenarios(all, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, all) {
		t.Fatalf("unexpected scenarios: %#v", got)
	}
}

func TestSelectScenariosFiltersRequestedNames(t *testing.T) {
	all := []scenario{{name: "one"}, {name: "two"}, {name: "three"}}
	got, err := selectScenarios(all, []string{"three", "one"})
	if err != nil {
		t.Fatal(err)
	}
	want := []scenario{{name: "three"}, {name: "one"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected scenarios: %#v", got)
	}
}

func TestToPowerShellArgsMapsKnownFlags(t *testing.T) {
	got := toPowerShellArgs([]string{"--force", "--non-interactive", "--custom"})
	want := []string{"-Force", "-NonInteractive", "--custom"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args: got %v want %v", got, want)
	}
}

func TestInstalledEnvPathHelpers(t *testing.T) {
	env := installedEnv{
		repoRoot:   filepath.Join("repo"),
		userBinDir: filepath.Join("user-bin"),
		agentHome:  filepath.Join("agent-home"),
	}

	afsverifyOS = "windows"
	installCmd := env.installCommand("--force", "--non-interactive")
	if installCmd.Path != "powershell" {
		t.Fatalf("unexpected install command path: %s", installCmd.Path)
	}
	if env.publishedAfsPath() != filepath.Join(env.userBinDir, "afs.exe") {
		t.Fatalf("unexpected published afs path: %s", env.publishedAfsPath())
	}
	if env.managedAfsPath() != filepath.Join(env.agentHome, "bin", "afs.exe") {
		t.Fatalf("unexpected managed afs path: %s", env.managedAfsPath())
	}
	if env.helperPath("add-agent") != filepath.Join(env.userBinDir, "add-agent.cmd") {
		t.Fatalf("unexpected helper path: %s", env.helperPath("add-agent"))
	}

	afsverifyOS = "darwin"
	if installCmd.Path != filepath.Join(env.repoRoot, "install.sh") {
		installCmd = env.installCommand("--force", "--non-interactive")
	}
	if installCmd.Path != filepath.Join(env.repoRoot, "install.sh") {
		t.Fatalf("unexpected install command path: %s", installCmd.Path)
	}
	if env.publishedAfsPath() != filepath.Join(env.userBinDir, "afs") {
		t.Fatalf("unexpected published afs path: %s", env.publishedAfsPath())
	}
	if env.managedAfsPath() != filepath.Join(env.agentHome, "bin", "afs") {
		t.Fatalf("unexpected managed afs path: %s", env.managedAfsPath())
	}
	if env.helperPath("add-agent") != filepath.Join(env.userBinDir, "add-agent") {
		t.Fatalf("unexpected helper path: %s", env.helperPath("add-agent"))
	}
	restoreAFSVerifyHooks()
}

func TestAssertExistsAndAssertNotExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(path, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := assertExists(path); err != nil {
		t.Fatalf("assertExists failed: %v", err)
	}
	if err := assertNotExists(path); err == nil {
		t.Fatal("expected assertNotExists to fail for existing path")
	}
}

func TestAssertExistsFailsForMissingPath(t *testing.T) {
	if err := assertExists(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected missing path to fail")
	}
}

func TestInstallMockClipboardToolCreatesExpectedTooling(t *testing.T) {
	binDir := t.TempDir()

	afsverifyOS = "windows"
	if err := installMockClipboardTool(binDir, filepath.Join(binDir, "clipboard.txt")); err != nil {
		t.Fatalf("installMockClipboardTool failed: %v", err)
	}
	if err := assertExists(filepath.Join(binDir, "clip.cmd")); err != nil {
		t.Fatalf("expected clip.cmd: %v", err)
	}

	afsverifyOS = "darwin"
	if err := installMockClipboardTool(binDir, filepath.Join(binDir, "clipboard.txt")); err != nil {
		t.Fatalf("installMockClipboardTool failed: %v", err)
	}

	if err := assertExists(filepath.Join(binDir, "pbcopy")); err != nil {
		t.Fatalf("expected pbcopy: %v", err)
	}
	if err := assertExists(filepath.Join(binDir, "wl-copy")); err != nil {
		t.Fatalf("expected wl-copy: %v", err)
	}
	restoreAFSVerifyHooks()
}

func TestNewInstalledEnvSetsExpectedFields(t *testing.T) {
	restoreAFSVerifyHooks()
	defer restoreAFSVerifyHooks()
	t.Setenv("PATH", "/usr/bin:/afsverify-host-marker:/bin")
	env, err := newInstalledEnv(filepath.Join("repo"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = env.cleanup() }()

	if env.repoRoot != filepath.Join("repo") {
		t.Fatalf("unexpected repo root: %s", env.repoRoot)
	}
	if env.tempRoot == "" || env.homeDir == "" || env.userBinDir == "" {
		t.Fatalf("expected populated env, got %+v", env)
	}
	if getEnvValue(env.baseEnv, "AGENT47_HOME") != env.agentHome {
		t.Fatalf("expected AGENT47_HOME=%s, got %s", env.agentHome, getEnvValue(env.baseEnv, "AGENT47_HOME"))
	}
	pathValue := getEnvValue(env.baseEnv, "PATH")
	if !strings.HasPrefix(pathValue, env.userBinDir+string(os.PathListSeparator)) {
		t.Fatalf("expected isolated PATH to start with user bin, got %q", pathValue)
	}
	if strings.Contains(pathValue, "afsverify-host-marker") {
		t.Fatalf("expected isolated PATH to drop arbitrary host marker, got %q", pathValue)
	}
}

func TestNewInstalledEnvUsesWindowsLayout(t *testing.T) {
	restoreAFSVerifyHooks()
	defer restoreAFSVerifyHooks()
	afsverifyOS = "windows"

	env, err := newInstalledEnv(filepath.Join("repo"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = env.cleanup() }()

	if !strings.Contains(env.agentHome, filepath.Join("AppData", "Local", "agent47")) {
		t.Fatalf("unexpected windows agent home: %s", env.agentHome)
	}
	if env.userBinDir != filepath.Join(env.agentHome, "bin") {
		t.Fatalf("unexpected windows user bin: %s", env.userBinDir)
	}
}

func TestRunCombinedCapturesSuccessAndFailure(t *testing.T) {
	success := exec.Command("sh", "-c", "printf hello; printf warn >&2")
	stdout, stderr, err := runCombined(success)
	if err != nil {
		t.Fatal(err)
	}
	if stdout != "hello" || stderr != "warn" {
		t.Fatalf("unexpected output: stdout=%q stderr=%q", stdout, stderr)
	}

	failure := exec.Command("sh", "-c", "printf boom; printf err >&2; exit 3")
	_, _, err = runCombined(failure)
	if err == nil {
		t.Fatal("expected runCombined failure")
	}
}

func TestAssertNotExistsSucceedsForMissingPath(t *testing.T) {
	if err := assertNotExists(filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatalf("expected missing path to pass: %v", err)
	}
}

func TestCleanupRemovesTempRoot(t *testing.T) {
	env, err := newInstalledEnv(filepath.Join("repo"))
	if err != nil {
		t.Fatal(err)
	}
	if err := env.cleanup(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(env.tempRoot); !os.IsNotExist(err) {
		t.Fatalf("expected temp root to be removed: %s", env.tempRoot)
	}
}

func TestVerifyScenariosWithFakeInstalledEnv(t *testing.T) {
	if goRuntime.GOOS == "windows" {
		t.Skip("fake installed env helper scripts are unix-only")
	}

	env := newFakeInstalledEnv(t)

	tests := []struct {
		name string
		run  func(*installedEnv) error
	}{
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

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scoped := cloneInstalledEnv(t, env)
			if err := tc.run(scoped); err != nil {
				t.Fatalf("scenario failed: %v", err)
			}
		})
	}
}

func TestVerifyInstalledAddSSPromptAcceptsStdoutFallback(t *testing.T) {
	if goRuntime.GOOS == "windows" {
		t.Skip("fake installed env helper scripts are unix-only")
	}

	env := newCustomInstalledEnv(t, fakeInstalledEnvOptions{ssPromptMode: "stdout-only"})
	if err := verifyInstalledAddSSPrompt(env); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyInstalledHelpAndDoctorFailsWhenDoctorMarkersMissing(t *testing.T) {
	if goRuntime.GOOS == "windows" {
		t.Skip("fake installed env helper scripts are unix-only")
	}

	env := newCustomInstalledEnv(t, fakeInstalledEnvOptions{doctorMode: "missing-markers"})
	if err := verifyInstalledHelpAndDoctor(env); err == nil {
		t.Fatal("expected doctor marker validation failure")
	}
}

func TestVerifyInstalledAddAgentFailsWhenProjectFilesMissing(t *testing.T) {
	if goRuntime.GOOS == "windows" {
		t.Skip("fake installed env helper scripts are unix-only")
	}

	env := newCustomInstalledEnv(t, fakeInstalledEnvOptions{addAgentMode: "missing-files"})
	if err := verifyInstalledAddAgent(env); err == nil {
		t.Fatal("expected add-agent verification failure")
	}
}

func TestVerifyInstalledAddAgentPromptFailsWhenPromptMissing(t *testing.T) {
	if goRuntime.GOOS == "windows" {
		t.Skip("fake installed env helper scripts are unix-only")
	}

	env := newCustomInstalledEnv(t, fakeInstalledEnvOptions{agentPromptMode: "missing-file"})
	if err := verifyInstalledAddAgentPrompt(env); err == nil {
		t.Fatal("expected add-agent-prompt verification failure")
	}
}

func TestRunReturnsOneWhenRepoRootFails(t *testing.T) {
	restoreAFSVerifyHooks()
	defer restoreAFSVerifyHooks()
	var stderr bytes.Buffer
	afsverifyStderr = &stderr
	afsverifyDetectRepoRoot = func() (string, error) { return "", errors.New("boom") }

	if got := run(nil); got != 1 {
		t.Fatalf("expected status 1, got %d", got)
	}
}

func TestRunReturnsOneWhenScenarioFails(t *testing.T) {
	restoreAFSVerifyHooks()
	defer restoreAFSVerifyHooks()
	var stdout, stderr bytes.Buffer
	afsverifyStdout = &stdout
	afsverifyStderr = &stderr
	afsverifyDetectRepoRoot = func() (string, error) { return "/repo", nil }
	afsverifyNewInstalledEnv = func(string) (*installedEnv, error) {
		return &installedEnv{tempRoot: t.TempDir()}, nil
	}
	afsverifyScenarios = func() []scenario {
		return []scenario{{name: "broken", run: func(*installedEnv) error { return errors.New("fail") }}}
	}

	if got := run(nil); got != 1 {
		t.Fatalf("expected status 1, got %d", got)
	}
	if !strings.Contains(stderr.String(), "scenario broken failed") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunReturnsOneWhenInstalledEnvCreationFails(t *testing.T) {
	restoreAFSVerifyHooks()
	defer restoreAFSVerifyHooks()
	afsverifyDetectRepoRoot = func() (string, error) { return "/repo", nil }
	afsverifyNewInstalledEnv = func(string) (*installedEnv, error) { return nil, errors.New("init failed") }

	if got := run(nil); got != 1 {
		t.Fatalf("expected status 1, got %d", got)
	}
}

func TestRunReturnsOneWhenScenarioSelectionFails(t *testing.T) {
	restoreAFSVerifyHooks()
	defer restoreAFSVerifyHooks()
	afsverifyDetectRepoRoot = func() (string, error) { return "/repo", nil }
	afsverifyNewInstalledEnv = func(string) (*installedEnv, error) {
		return &installedEnv{tempRoot: t.TempDir()}, nil
	}

	if got := run([]string{"missing"}); got != 1 {
		t.Fatalf("expected status 1, got %d", got)
	}
}

func TestRunReturnsZeroWhenScenarioSucceeds(t *testing.T) {
	restoreAFSVerifyHooks()
	defer restoreAFSVerifyHooks()
	var stdout, stderr bytes.Buffer
	afsverifyStdout = &stdout
	afsverifyStderr = &stderr
	afsverifyDetectRepoRoot = func() (string, error) { return "/repo", nil }
	afsverifyNewInstalledEnv = func(string) (*installedEnv, error) {
		return &installedEnv{tempRoot: t.TempDir()}, nil
	}
	afsverifyScenarios = func() []scenario {
		return []scenario{{name: "ok", run: func(*installedEnv) error { return nil }}}
	}

	if got := run(nil); got != 0 {
		t.Fatalf("expected status 0, got %d stderr=%s", got, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Scenario passed: ok") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestDetectRepoRootUsesWorkingDirectory(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "AGENTS.md"), []byte("agents\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(repoRoot, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(wd) }()

	got, err := detectRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("expected detected repo root")
	}
}

func restoreAFSVerifyHooks() {
	afsverifyOS = goRuntime.GOOS
	afsverifyDetectRepoRoot = detectRepoRoot
	afsverifyNewInstalledEnv = newInstalledEnv
	afsverifyScenarios = func() []scenario {
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
	afsverifyStdout = os.Stdout
	afsverifyStderr = os.Stderr
}

type fakeInstalledEnvOptions struct {
	doctorMode      string
	addAgentMode    string
	agentPromptMode string
	ssPromptMode    string
}

func newFakeInstalledEnv(t *testing.T) *installedEnv {
	t.Helper()
	return newCustomInstalledEnv(t, fakeInstalledEnvOptions{})
}

func newCustomInstalledEnv(t *testing.T, opts fakeInstalledEnvOptions) *installedEnv {
	t.Helper()
	repoRoot := t.TempDir()
	doctorBody := "printf '[OK] afs in PATH\\n[OK] Templates installed\\n[OK] add-agent available\\n[OK] add-agent-prompt available\\n[OK] add-ss-prompt available\\n'\n"
	if opts.doctorMode == "missing-markers" {
		doctorBody = "printf 'doctor incomplete\\n'\n"
	}
	addAgentBody := "if [ \"${1:-}\" = \"--only-skills\" ]; then\n  if [ \"${2:-}\" = \"--force\" ]; then\n    rm -rf \"$PWD/skills\"\n  fi\n  mkdir -p \"$PWD/skills\"\n  printf '<skills />\\n' > \"$PWD/skills/AVAILABLE_SKILLS.xml\"\n  exit 0\nfi\nif [ \"${1:-}\" = \"--force\" ]; then\n  rm -f \"$PWD/rules/stale.yaml\"\nfi\nmkdir -p \"$PWD/rules\" \"$PWD/skills\"\nprintf 'agents\\n' > \"$PWD/AGENTS.md\"\nprintf 'readme\\n' > \"$PWD/README.md\"\nprintf '<skills />\\n' > \"$PWD/skills/AVAILABLE_SKILLS.xml\"\n"
	if opts.addAgentMode == "missing-files" {
		addAgentBody = "mkdir -p \"$PWD/rules\"\n"
	}
	agentPromptBody := "mkdir -p \"$PWD/prompts\"\nprintf 'prompt\\n' > \"$PWD/prompts/agent-prompt.txt\"\n"
	if opts.agentPromptMode == "missing-file" {
		agentPromptBody = "mkdir -p \"$PWD/prompts\"\n"
	}
	ssPromptBody := "if [ -n \"${TARGET_FILE:-}\" ]; then\n  printf 'Review this repository\\n' > \"${TARGET_FILE}\"\n  printf 'Snapshot/spec prompt copied to clipboard\\n'\nelse\n  printf 'Review this repository\\n'\nfi\n"
	if opts.ssPromptMode == "stdout-only" {
		ssPromptBody = "printf 'Review this repository\\n'\n"
	}

	installScript := `#!/bin/sh
set -eu
agent_home="${AGENT47_HOME}"
user_bin="${HOME}/bin"
mkdir -p "${agent_home}/templates" "${agent_home}/bin" "${user_bin}"
printf 'vtest\n' > "${agent_home}/VERSION"
cat > "${agent_home}/bin/afs" <<'EOS'
#!/bin/sh
set -eu
case "${1:-}" in
  help)
    printf 'Core commands:\n'
    ;;
  add-agent)
    shift
    exec "${HOME}/bin/add-agent" "$@"
    ;;
  doctor)
    shift
    case "${1:-}" in
      --check-update|--check-update-force)
        ` + doctorBody + `
        printf '[OK] Up to date (version vtest)\n'
        ;;
      *)
        ` + doctorBody + `
        ;;
    esac
    ;;
  uninstall)
    rm -rf "${AGENT47_HOME}"
    rm -f "${HOME}/bin/afs" "${HOME}/bin/add-agent" "${HOME}/bin/add-agent-prompt" "${HOME}/bin/add-ss-prompt"
    ;;
esac
EOS
chmod +x "${agent_home}/bin/afs"
ln -sf "${agent_home}/bin/afs" "${user_bin}/afs"
cat > "${user_bin}/add-agent" <<'EOS'
#!/bin/sh
set -eu
` + addAgentBody + `
EOS
chmod +x "${user_bin}/add-agent"
cat > "${user_bin}/add-agent-prompt" <<'EOS'
#!/bin/sh
set -eu
` + agentPromptBody + `
EOS
chmod +x "${user_bin}/add-agent-prompt"
cat > "${user_bin}/add-ss-prompt" <<'EOS'
#!/bin/sh
set -eu
` + ssPromptBody + `
EOS
chmod +x "${user_bin}/add-ss-prompt"
`
	if err := os.WriteFile(filepath.Join(repoRoot, "install.sh"), []byte(installScript), 0o755); err != nil {
		t.Fatal(err)
	}

	env, err := newInstalledEnv(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = env.cleanup() })
	return env
}

func cloneInstalledEnv(t *testing.T, base *installedEnv) *installedEnv {
	t.Helper()
	env, err := newInstalledEnv(base.repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = env.cleanup() })
	return env
}

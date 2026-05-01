package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/runtime"
	"github.com/leanbusqts/agent47/internal/templates"
)

func TestRunAddAgentCreatesCoreFiles(t *testing.T) {
	env := newAddAgentEnv(t)
	status, stdout, _ := env.run(t, "add-agent")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stdout)
	}

	assertFileExists(t, filepath.Join(env.workDir, "AGENTS.md"))
	assertFileExists(t, filepath.Join(env.workDir, "rules", "security-global.yaml"))
	assertFileExists(t, filepath.Join(env.workDir, "rules", "security-shell.yaml"))
	assertFileExists(t, filepath.Join(env.workDir, "skills", "analyze", "SKILL.md"))
	assertFileExists(t, filepath.Join(env.workDir, "skills", "review", "SKILL.md"))
	assertFileExists(t, filepath.Join(env.workDir, "skills", "AVAILABLE_SKILLS.xml"))
	assertFileExists(t, filepath.Join(env.workDir, "README.md"))
	assertFileExists(t, filepath.Join(env.workDir, "prompts", "agent-prompt.txt"))
}

func TestRunAddAgentOnlySkills(t *testing.T) {
	env := newAddAgentEnv(t)
	status, _, _ := env.run(t, "add-agent", "--only-skills")
	if status != 0 {
		t.Fatalf("expected status 0, got %d", status)
	}

	assertFileExists(t, filepath.Join(env.workDir, "skills", "analyze", "SKILL.md"))
	assertFileExists(t, filepath.Join(env.workDir, "skills", "AVAILABLE_SKILLS.xml"))
	assertNotExists(t, filepath.Join(env.workDir, "AGENTS.md"))
	assertNotExists(t, filepath.Join(env.workDir, "rules"))
}

func TestRunAddAgentOnlySkillsPreviewDoesNotWriteFiles(t *testing.T) {
	env := newAddAgentEnv(t)

	status, stdout, stderr := env.run(t, "add-agent", "--only-skills", "--preview")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}
	if !strings.Contains(stdout, "mode: only-skills") {
		t.Fatalf("expected only-skills preview mode, got %s", stdout)
	}
	assertNotExists(t, filepath.Join(env.workDir, "skills"))
	assertNotExists(t, filepath.Join(env.workDir, "AGENTS.md"))
	assertNotExists(t, filepath.Join(env.workDir, "rules"))
}

func TestRunAddAgentOnlySkillsRespectsExplicitBundles(t *testing.T) {
	env := newAddAgentEnv(t)

	status, _, stderr := env.run(t, "add-agent", "--only-skills", "--bundle", "cli", "--yes")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}

	assertFileExists(t, filepath.Join(env.workDir, "skills", "cli-design", "SKILL.md"))
	assertNotExists(t, filepath.Join(env.workDir, "skills", "optimize"))
	assertNotExists(t, filepath.Join(env.workDir, "skills", "refactor"))
	assertNotExists(t, filepath.Join(env.workDir, "AGENTS.md"))
	assertNotExists(t, filepath.Join(env.workDir, "rules"))
}

func TestRunAddAgentOnlySkillsPromptsBeforeWriting(t *testing.T) {
	env := newAddAgentEnv(t)

	restoreConfirmHooks := overrideAddAgentConfirmHooks(
		t,
		func(bool) bool { return true },
		func(*Root) bool { return false },
	)
	defer restoreConfirmHooks()

	status, stdout, stderr := env.run(t, "add-agent", "--only-skills")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}
	if !strings.Contains(stdout, "Aborted before writing.") {
		t.Fatalf("expected abort message, got %s", stdout)
	}

	assertNotExists(t, filepath.Join(env.workDir, "skills"))
	assertNotExists(t, filepath.Join(env.workDir, "AGENTS.md"))
	assertNotExists(t, filepath.Join(env.workDir, "rules"))
}

func TestShouldConfirmWriteReturnsTrueForTTYWithoutYesOrCI(t *testing.T) {
	restoreStatHooks := overrideConfirmStatHooks(
		t,
		func() (os.FileInfo, error) { return fakeTTYFileInfo{}, nil },
		func() (os.FileInfo, error) { return fakeTTYFileInfo{}, nil },
	)
	defer restoreStatHooks()

	if shouldConfirmWrite(true) {
		t.Fatal("expected --yes to bypass confirmation")
	}

	if got := shouldConfirmWrite(false); !got {
		t.Fatal("expected confirmation for interactive TTY")
	}
}

func TestShouldConfirmWriteReturnsFalseOutsideTTYOrInCI(t *testing.T) {
	restoreStatHooks := overrideConfirmStatHooks(
		t,
		func() (os.FileInfo, error) { return fakeRegularFileInfo{}, nil },
		func() (os.FileInfo, error) { return fakeTTYFileInfo{}, nil },
	)
	defer restoreStatHooks()

	if got := shouldConfirmWrite(false); got {
		t.Fatal("expected no confirmation when stdin is not a TTY")
	}

	t.Setenv("CI", "1")
	restoreTTYStatHooks := overrideConfirmStatHooks(
		t,
		func() (os.FileInfo, error) { return fakeTTYFileInfo{}, nil },
		func() (os.FileInfo, error) { return fakeTTYFileInfo{}, nil },
	)
	defer restoreTTYStatHooks()

	if got := shouldConfirmWrite(false); got {
		t.Fatal("expected CI to bypass confirmation")
	}
}

func TestRunAddAgentForcePreservesProjectFilesAndRefreshesManagedArea(t *testing.T) {
	env := newAddAgentEnv(t)
	mustWriteFile(t, filepath.Join(env.workDir, "AGENTS.md"), "old agents\n")
	mustWriteFile(t, filepath.Join(env.workDir, "rules", "rules-backend.yaml"), "old rule\n")
	mustWriteFile(t, filepath.Join(env.workDir, "rules", "custom-rule.yaml"), "stale managed rule\n")
	mustWriteFile(t, filepath.Join(env.workDir, "skills", "custom-skill", "SKILL.md"), "---\nname: custom-skill\ndescription: Local custom skill.\n---\n")
	mustWriteFile(t, filepath.Join(env.workDir, "README.md"), "custom readme\n")
	mustWriteFile(t, filepath.Join(env.workDir, "SPEC.md"), "custom product spec\n")

	status, _, _ := env.run(t, "add-agent", "--force")
	if status != 0 {
		t.Fatalf("expected status 0, got %d", status)
	}

	assertFileContains(t, filepath.Join(env.workDir, "AGENTS.md"), "single source of operating policy")
	assertNotExists(t, filepath.Join(env.workDir, "rules", "rules-backend.yaml"))
	assertNotExists(t, filepath.Join(env.workDir, "rules", "custom-rule.yaml"))
	assertFileContains(t, filepath.Join(env.workDir, "rules", "security-global.yaml"), "Never hardcode secrets")
	assertNotExists(t, filepath.Join(env.workDir, "skills", "custom-skill"))
	assertFileContains(t, filepath.Join(env.workDir, "skills", "analyze", "SKILL.md"), "name: analyze")
	assertFileContains(t, filepath.Join(env.workDir, "README.md"), "custom readme")
	assertFileContains(t, filepath.Join(env.workDir, "SPEC.md"), "custom product spec")
	assertFileExists(t, filepath.Join(env.workDir, "prompts", "agent-prompt.txt"))
}

func TestRunAddAgentForceRollsBackOnInvalidSkillTemplate(t *testing.T) {
	env := newAddAgentEnv(t)
	mustWriteFile(t, filepath.Join(env.workDir, "AGENTS.md"), "existing agents\n")
	mustWriteFile(t, filepath.Join(env.workDir, "rules", "rules-backend.yaml"), "existing rule\n")
	mustWriteFile(t, filepath.Join(env.workDir, "skills", "analyze", "SKILL.md"), "existing skill\n")
	mustWriteFile(t, filepath.Join(env.repoRoot, "templates", "base", "skills", "analyze", "SKILL.md"), "not-valid-frontmatter\n")

	status, _, stderr := env.run(t, "add-agent", "--force")
	if status == 0 {
		t.Fatal("expected non-zero status")
	}
	if !strings.Contains(stderr, "missing frontmatter fence") {
		t.Fatalf("expected validation error, got %s", stderr)
	}

	assertFileContains(t, filepath.Join(env.workDir, "AGENTS.md"), "existing agents")
	assertFileContains(t, filepath.Join(env.workDir, "rules", "rules-backend.yaml"), "existing rule")
	assertFileContains(t, filepath.Join(env.workDir, "skills", "analyze", "SKILL.md"), "existing skill")
}

func TestRunAddAgentOnlySkillsPreservesInvalidExistingSkillWithoutForce(t *testing.T) {
	env := newAddAgentEnv(t)
	mustWriteFile(t, filepath.Join(env.workDir, "skills", "analyze", "SKILL.md"), "invalid-skill-body\n")

	status, stdout, stderr := env.run(t, "add-agent", "--only-skills")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}
	if !strings.Contains(stdout, "preserving existing content") {
		t.Fatalf("expected warning about preserving invalid skill, got %s", stdout)
	}

	assertFileContains(t, filepath.Join(env.workDir, "skills", "analyze", "SKILL.md"), "invalid-skill-body")
	assertFileExists(t, filepath.Join(env.workDir, "skills", "AVAILABLE_SKILLS.xml"))
}

func TestRunAddAgentPreviewShowsConflictFallback(t *testing.T) {
	env := newAddAgentEnv(t)
	mustWriteFile(t, filepath.Join(env.workDir, "package.json"), `{"dependencies":{"react":"1.0.0","express":"1.0.0"}}`)
	if err := os.MkdirAll(filepath.Join(env.workDir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(env.workDir, "api"), 0o755); err != nil {
		t.Fatal(err)
	}

	status, stdout, stderr := env.run(t, "add-agent", "--preview")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}
	if !strings.Contains(stdout, "Multiple project types detected with no supported automatic composition") {
		t.Fatalf("expected conflict warning, got %s", stdout)
	}
	if !strings.Contains(stdout, "bundles: base") {
		t.Fatalf("expected base fallback, got %s", stdout)
	}
}

func TestRunAddAgentRejectsIncompatibleExplicitBundles(t *testing.T) {
	env := newAddAgentEnv(t)

	status, _, stderr := env.run(t, "add-agent", "--preview", "--bundle", "frontend", "--bundle", "backend")
	if status == 0 {
		t.Fatal("expected non-zero status")
	}
	if !strings.Contains(stderr, "explicit bundle selection is incompatible") {
		t.Fatalf("expected incompatible bundle error, got %s", stderr)
	}
}

func TestRunAddAgentAllowsCompatibleExplicitBundles(t *testing.T) {
	env := newAddAgentEnv(t)

	status, stdout, stderr := env.run(t, "add-agent", "--preview", "--bundle", "cli", "--bundle", "scripts")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}
	if !strings.Contains(stdout, "bundles: base, project-cli, project-scripts, shared-cli-behavior, shared-testing") {
		t.Fatalf("expected compatible bundle preview, got %s", stdout)
	}
}

func TestNormalizeBootstrapErrorHandlesNil(t *testing.T) {
	if got := normalizeBootstrapError(nil); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestNormalizeBootstrapErrorAddsConflictGuidance(t *testing.T) {
	got := normalizeBootstrapError(templates.AssemblyConflictError{
		Path:   "rules/shared.yaml",
		Detail: "base and project-cli provide different content",
	})
	if !strings.Contains(got, "Use a supported bundle composition or remove one of the conflicting bundles.") {
		t.Fatalf("expected actionable conflict guidance, got %q", got)
	}
}

func TestNormalizeBootstrapErrorAddsMissingTemplateGuidance(t *testing.T) {
	got := normalizeBootstrapError(templates.MissingTemplateError{Path: "rules/rules-cli.yaml"})
	if !strings.Contains(got, "Template not found: rules/rules-cli.yaml") {
		t.Fatalf("expected template path, got %q", got)
	}
	if !strings.Contains(got, "Restore the missing template asset") {
		t.Fatalf("expected actionable guidance, got %q", got)
	}
}

func TestNormalizeBootstrapErrorAddsSkillsGuidance(t *testing.T) {
	got := normalizeBootstrapError(templates.MissingTemplateError{Path: "skills"})
	if !strings.Contains(got, "Skills templates are missing.") {
		t.Fatalf("expected skills guidance, got %q", got)
	}
}

func TestNormalizeBootstrapErrorAddsInvalidBundleManifestGuidance(t *testing.T) {
	got := normalizeBootstrapError(templates.InvalidBundleManifestError{
		Path:   "bundles/project-cli/manifest.txt",
		Detail: "manifest has no entries",
	})
	if !strings.Contains(got, "invalid bundle manifest bundles/project-cli/manifest.txt") {
		t.Fatalf("expected invalid bundle manifest detail, got %q", got)
	}
	if !strings.Contains(got, "Restore the bundle manifest contents or exclude that bundle") {
		t.Fatalf("expected actionable bundle guidance, got %q", got)
	}
}

type addAgentEnv struct {
	repoRoot string
	workDir  string
	cfg      runtime.Config
	root     *Root
}

func newAddAgentEnv(t *testing.T) addAgentEnv {
	t.Helper()

	baseDir := t.TempDir()
	repoRoot := filepath.Join(baseDir, "repo")
	workDir := filepath.Join(baseDir, "work")

	copyDirFromRepo(t, filepath.Join(repoRoot, "templates"), "templates")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))

	return addAgentEnv{
		repoRoot: repoRoot,
		workDir:  workDir,
		cfg: runtime.Config{
			Version:      "vtest",
			TemplateMode: runtime.TemplateModeFilesystem,
			RepoRoot:     repoRoot,
		},
		root: root,
	}
}

func (e addAgentEnv) run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	e.root = NewRoot(cli.NewOutput(&stdout, &stderr))

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(e.workDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()

	status := e.root.Run(context.Background(), e.cfg, args)
	return status, stdout.String(), stderr.String()
}

func copyDirFromRepo(t *testing.T, dst string, rel string) {
	t.Helper()

	src := filepath.Join(repoRoot(t), rel)
	if err := copyDir(src, dst); err != nil {
		t.Fatal(err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dstPath, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func mustWriteFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %s", path)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected path to not exist: %s", path)
	}
}

func assertFileContains(t *testing.T, path string, fragment string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), fragment) {
		t.Fatalf("expected %s to contain %q, got %s", path, fragment, string(data))
	}
}

func overrideAddAgentConfirmHooks(t *testing.T, should func(bool) bool, confirm func(*Root) bool) func() {
	t.Helper()

	prevShould := shouldConfirmWriteFunc
	prevConfirm := confirmWriteFunc
	shouldConfirmWriteFunc = should
	confirmWriteFunc = confirm

	return func() {
		shouldConfirmWriteFunc = prevShould
		confirmWriteFunc = prevConfirm
	}
}

func overrideConfirmStatHooks(t *testing.T, stdin func() (os.FileInfo, error), stdout func() (os.FileInfo, error)) func() {
	t.Helper()

	prevStdin := stdinStatFunc
	prevStdout := stdoutStatFunc
	stdinStatFunc = stdin
	stdoutStatFunc = stdout

	return func() {
		stdinStatFunc = prevStdin
		stdoutStatFunc = prevStdout
	}
}

type fakeTTYFileInfo struct{}

func (fakeTTYFileInfo) Name() string       { return "tty" }
func (fakeTTYFileInfo) Size() int64        { return 0 }
func (fakeTTYFileInfo) Mode() os.FileMode  { return os.ModeCharDevice }
func (fakeTTYFileInfo) ModTime() time.Time { return time.Time{} }
func (fakeTTYFileInfo) IsDir() bool        { return false }
func (fakeTTYFileInfo) Sys() any           { return nil }

type fakeRegularFileInfo struct{}

func (fakeRegularFileInfo) Name() string       { return "file" }
func (fakeRegularFileInfo) Size() int64        { return 0 }
func (fakeRegularFileInfo) Mode() os.FileMode  { return 0 }
func (fakeRegularFileInfo) ModTime() time.Time { return time.Time{} }
func (fakeRegularFileInfo) IsDir() bool        { return false }
func (fakeRegularFileInfo) Sys() any           { return nil }

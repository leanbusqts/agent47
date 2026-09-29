package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leanbusqts/agent47/internal/analyze"
	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func TestRunInitCreatesPolicyRulesAndGeneratedContext(t *testing.T) {
	env := newInitTestEnv(t)
	mustWriteInitAppFile(t, filepath.Join(env.workDir, "README.md"), "project readme\n")
	mustWriteInitAppFile(t, filepath.Join(env.workDir, "rules", "custom.yaml"), "custom\n")

	status, stdout, stderr := runInitTest(t, env, "--bundle", "cli", "--exclude-bundle", "scripts")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}
	if !strings.HasPrefix(stdout, "[INFO] Analyzing repository...\nPlan\n") || !strings.Contains(stdout, "project-cli") {
		t.Fatalf("expected resolved apply plan, got %s", stdout)
	}
	assertInitAppFileExists(t, filepath.Join(env.workDir, "AGENTS.md"))
	assertInitAppFileExists(t, filepath.Join(env.workDir, "rules", "rules-cli.yaml"))
	assertInitAppFileExists(t, filepath.Join(env.workDir, "rules", "security-global.yaml"))
	assertInitAppFileContains(t, filepath.Join(env.workDir, ".agent47", "context.md"), "<!-- afs-context")
	assertInitAppFileContains(t, filepath.Join(env.workDir, ".agent47", "context.md"), "rules/rules-cli.yaml")
	assertInitAppFileContains(t, filepath.Join(env.workDir, "rules", "custom.yaml"), "custom")
	assertInitAppFileContains(t, filepath.Join(env.workDir, "README.md"), "project readme")
	assertInitAppNotExists(t, filepath.Join(env.workDir, ".agents"))
	assertInitAppNotExists(t, filepath.Join(env.workDir, "skills"))
	assertInitAppNotExists(t, filepath.Join(env.workDir, "prompts"))
}

func TestRunInitForceReplacesLegacyCLIContent(t *testing.T) {
	env := newInitTestEnv(t)
	mustWriteInitAppFile(t, filepath.Join(env.workDir, "AGENTS.md"), "old agents\n")
	mustWriteInitAppFile(t, filepath.Join(env.workDir, "rules", "security-global.yaml"), "old security\n")
	mustWriteInitAppFile(t, filepath.Join(env.workDir, "rules", "unknown.yaml"), "unknown\n")
	mustWriteInitAppFile(t, filepath.Join(env.workDir, "skills", "legacy", "SKILL.md"), "legacy skill\n")
	mustWriteInitAppFile(t, filepath.Join(env.workDir, "prompts", "agent-prompt.txt"), "legacy prompt\n")
	mustWriteInitAppFile(t, filepath.Join(env.workDir, ".agents", "specs", "spec.yml"), "legacy task\n")
	mustWriteInitAppFile(t, filepath.Join(env.workDir, ".agents", "keep.txt"), "keep\n")
	mustWriteInitAppFile(t, filepath.Join(env.workDir, ".agent47", "context.md"), "user context\n")
	mustWriteInitAppFile(t, filepath.Join(env.workDir, "README.md"), "readme\n")

	status, stdout, stderr := runInitTest(t, env, "--force", "--preview")
	if status != 0 {
		t.Fatalf("expected preview status 0, got %d: %s", status, stderr)
	}
	if !strings.HasPrefix(stdout, "[INFO] Analyzing repository...\nPreview\n") {
		t.Fatalf("expected preview heading, got %s", stdout)
	}
	assertInitAppFileContains(t, filepath.Join(env.workDir, "rules", "unknown.yaml"), "unknown")
	assertInitAppFileExists(t, filepath.Join(env.workDir, "skills", "legacy", "SKILL.md"))

	status, stdout, stderr = runInitTest(t, env, "--force")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}
	if !strings.HasPrefix(stdout, "[INFO] Analyzing repository...\nPlan\n") {
		t.Fatalf("expected apply plan heading, got %s", stdout)
	}
	if !strings.Contains(stdout, "  update:\n    AGENTS.md") {
		t.Fatalf("expected update plan, got %s", stdout)
	}
	for _, removed := range []string{".agents/specs/spec.yml", "prompts/", "rules/unknown.yaml", "skills/"} {
		if !strings.Contains(stdout, removed) {
			t.Fatalf("expected removal %s in plan, got %s", removed, stdout)
		}
	}
	assertInitAppFileContains(t, filepath.Join(env.workDir, "AGENTS.md"), "[AG-001]")
	assertInitAppFileContains(t, filepath.Join(env.workDir, "rules", "security-global.yaml"), "SEC-global-001")
	assertInitAppNotExists(t, filepath.Join(env.workDir, "rules", "unknown.yaml"))
	assertInitAppNotExists(t, filepath.Join(env.workDir, "skills"))
	assertInitAppNotExists(t, filepath.Join(env.workDir, "prompts"))
	assertInitAppNotExists(t, filepath.Join(env.workDir, ".agents", "specs", "spec.yml"))
	assertInitAppFileContains(t, filepath.Join(env.workDir, ".agents", "keep.txt"), "keep")
	assertInitAppFileContains(t, filepath.Join(env.workDir, ".agent47", "context.md"), "user context")
	assertInitAppFileContains(t, filepath.Join(env.workDir, "README.md"), "readme")
}

func TestRunInitForceCreatesContextFromPostMigrationState(t *testing.T) {
	env := newInitTestEnv(t)
	mustWriteInitAppFile(t, filepath.Join(env.workDir, "rules", "custom.yaml"), "custom\n")
	mustWriteInitAppFile(t, filepath.Join(env.workDir, "skills", "legacy", "tool.go"), "package legacy\n")

	status, _, stderr := runInitTest(t, env, "--bundle", "cli", "--force")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}
	contextPath := filepath.Join(env.workDir, ".agent47", "context.md")
	assertInitAppFileContains(t, contextPath, "rules/rules-cli.yaml")
	data, err := os.ReadFile(contextPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "rules/custom.yaml") || strings.Contains(string(data), "skills/legacy") {
		t.Fatalf("force-generated context included removed legacy paths:\n%s", data)
	}
}

func TestRunInitAppliesWithoutConfirmationInCI(t *testing.T) {
	env := newInitTestEnv(t)
	t.Setenv("CI", "true")
	status, _, stderr := runInitTest(t, env)
	if status != 0 {
		t.Fatalf("expected init to apply without confirmation, status=%d stderr=%s", status, stderr)
	}
	assertInitAppFileExists(t, filepath.Join(env.workDir, "AGENTS.md"))
}

func TestRunInitReturnsUsageAndOperationalExitCodes(t *testing.T) {
	env := newInitTestEnv(t)
	status, stdout, stderr := runInitTest(t, env, "--bundle")
	if status != 2 || stdout != "" || !strings.Contains(stderr, initUsage) {
		t.Fatalf("expected usage status 2, got status=%d stdout=%s stderr=%s", status, stdout, stderr)
	}

	status, _, stderr = runInitTest(t, env, "--bundle", "missing")
	if status != 2 || !strings.Contains(stderr, "unknown bundle") {
		t.Fatalf("expected usage status 2, got status=%d stderr=%s", status, stderr)
	}
}

func TestRunInitRejectsRemovedYesFlag(t *testing.T) {
	env := newInitTestEnv(t)
	status, stdout, stderr := runInitTest(t, env, "--yes")
	if status != 2 || stdout != "" || !strings.Contains(stderr, initUsage) {
		t.Fatalf("expected removed flag to be a usage error, status=%d stdout=%s stderr=%s", status, stdout, stderr)
	}
}

func TestRunInitRejectsIncludedAndExcludedBundle(t *testing.T) {
	env := newInitTestEnv(t)
	status, stdout, stderr := runInitTest(t, env, "--bundle", "cli", "--exclude-bundle", "cli")
	if status != 2 || stdout != "" || !strings.Contains(stderr, initUsage) {
		t.Fatalf("expected usage status 2, got status=%d stdout=%s stderr=%s", status, stdout, stderr)
	}
}

func TestRunInitPreviewIsDeterministicAndNeverWrites(t *testing.T) {
	env := newInitTestEnv(t)

	status, first, stderr := runInitTest(t, env, "--bundle", "cli", "--preview")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}
	status, second, stderr := runInitTest(t, env, "--bundle", "cli", "--dry-run")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stderr)
	}
	if first != second {
		t.Fatalf("preview and dry-run differ:\npreview:\n%s\ndry-run:\n%s", first, second)
	}
	if !strings.HasPrefix(first, "[INFO] Analyzing repository...\nPreview\n") {
		t.Fatalf("expected preview heading, got %s", first)
	}
	if !strings.Contains(first, "  remove:\n    (none)") {
		t.Fatalf("preview must include an empty removal group: %s", first)
	}
	if !strings.Contains(first, ".agent47/context.md") {
		t.Fatalf("preview must include generated context: %s", first)
	}
	if strings.Index(first, "AGENTS.md") > strings.Index(first, "rules/rules-cli.yaml") {
		t.Fatalf("expected sorted deterministic paths, got %s", first)
	}
	assertInitAppNotExists(t, filepath.Join(env.workDir, "AGENTS.md"))
	assertInitAppNotExists(t, filepath.Join(env.workDir, "rules"))
	assertInitAppNotExists(t, filepath.Join(env.workDir, ".agent47"))
}

func TestCollectInitWarningsSurfacesPolicyAndConflictRisks(t *testing.T) {
	warnings := collectInitWarnings(analyze.AnalysisResult{
		Warnings: []string{"analysis warning"},
		AgentPolicy: analyze.AgentPolicy{
			Warnings:    []string{"repository entry limit reached"},
			AgentsFiles: []analyze.AgentPolicyFile{{Path: "packages/api/AGENTS.md", Scope: "packages/api"}},
			Tools: analyze.AgentPolicyTools{
				Codex: analyze.ToolPolicy{
					State:    analyze.PolicyInvalid,
					Evidence: []string{".codex/config.toml"},
					Warnings: []string{"invalid TOML"},
				},
			},
		},
		UnresolvedConflict:   true,
		ConflictProjectTypes: []string{"backend", "frontend"},
	})

	for _, want := range []string{
		"analysis warning",
		"agent policy warning: repository entry limit reached",
		"nested policy detected: packages/api/AGENTS.md",
		"vendor policy detected: codex (invalid): .codex/config.toml",
		"codex policy warning: invalid TOML",
		"unresolved project-type conflict; using the base bundle: backend, frontend",
	} {
		if !containsStringValue(warnings, want) {
			t.Fatalf("expected warning %q, got %v", want, warnings)
		}
	}
}

type initTestEnv struct {
	repoRoot string
	workDir  string
	cfg      runtime.Config
}

func newInitTestEnv(t *testing.T) initTestEnv {
	t.Helper()
	baseDir := t.TempDir()
	repoRoot := filepath.Join(baseDir, "repo")
	workDir := filepath.Join(baseDir, "work")
	if err := copyInitAppDir(filepath.Join(initAppRepoRoot(t), "templates"), filepath.Join(repoRoot, "templates")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return initTestEnv{
		repoRoot: repoRoot,
		workDir:  workDir,
		cfg: runtime.Config{
			Version:      "vtest",
			TemplateMode: runtime.TemplateModeFilesystem,
			RepoRoot:     repoRoot,
		},
	}
}

func runInitTest(t *testing.T, env initTestEnv, args ...string) (int, string, string) {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))

	oldWorkDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(env.workDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWorkDir) }()

	status := root.runInit(context.Background(), env.cfg, args)
	return status, stdout.String(), stderr.String()
}

func initAppRepoRoot(t *testing.T) string {
	t.Helper()
	workDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(workDir, "..", ".."))
}

func copyInitAppDir(source, destination string) error {
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(source, entry.Name())
		destinationPath := filepath.Join(destination, entry.Name())
		if entry.IsDir() {
			if err := copyInitAppDir(sourcePath, destinationPath); err != nil {
				return err
			}
			continue
		}
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(destinationPath, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func mustWriteInitAppFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertInitAppFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %s", path)
	}
}

func assertInitAppNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected path not to exist: %s", path)
	}
}

func assertInitAppFileContains(t *testing.T, path, fragment string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), fragment) {
		t.Fatalf("expected %s to contain %q, got %s", path, fragment, data)
	}
}

func containsStringValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func TestRunAddAgentCreatesCoreFiles(t *testing.T) {
	env := newAddAgentEnv(t)
	status, stdout, _ := env.run(t, "add-agent")
	if status != 0 {
		t.Fatalf("expected status 0, got %d: %s", status, stdout)
	}

	assertFileExists(t, filepath.Join(env.workDir, "AGENTS.md"))
	assertFileExists(t, filepath.Join(env.workDir, "rules", "rules-backend.yaml"))
	assertFileExists(t, filepath.Join(env.workDir, "skills", "analyze", "SKILL.md"))
	assertFileExists(t, filepath.Join(env.workDir, "skills", "AVAILABLE_SKILLS.xml"))
	assertFileExists(t, filepath.Join(env.workDir, "README.md"))
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
	assertFileContains(t, filepath.Join(env.workDir, "rules", "rules-backend.yaml"), "Controllers and transport adapters handle transport concerns only")
	assertNotExists(t, filepath.Join(env.workDir, "rules", "custom-rule.yaml"))
	assertNotExists(t, filepath.Join(env.workDir, "skills", "custom-skill"))
	assertFileContains(t, filepath.Join(env.workDir, "skills", "analyze", "SKILL.md"), "name: analyze")
	assertFileContains(t, filepath.Join(env.workDir, "README.md"), "custom readme")
	assertFileContains(t, filepath.Join(env.workDir, "SPEC.md"), "custom product spec")
}

func TestRunAddAgentForceRollsBackOnInvalidSkillTemplate(t *testing.T) {
	env := newAddAgentEnv(t)
	mustWriteFile(t, filepath.Join(env.workDir, "AGENTS.md"), "existing agents\n")
	mustWriteFile(t, filepath.Join(env.workDir, "rules", "rules-backend.yaml"), "existing rule\n")
	mustWriteFile(t, filepath.Join(env.workDir, "skills", "analyze", "SKILL.md"), "existing skill\n")
	mustWriteFile(t, filepath.Join(env.repoRoot, "templates", "skills", "analyze", "SKILL.md"), "not-valid-frontmatter\n")

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

func TestNormalizeBootstrapErrorHandlesNil(t *testing.T) {
	if got := normalizeBootstrapError(nil); got != "" {
		t.Fatalf("expected empty string, got %q", got)
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

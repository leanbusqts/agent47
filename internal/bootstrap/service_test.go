package bootstrap

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/fsx"
	"github.com/leanbusqts/agent47/internal/manifest"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func TestPrepareStateHonorsStageRootOverride(t *testing.T) {
	service := Service{FS: fsx.Service{}, Out: cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{})}
	override := t.TempDir()
	t.Setenv("AGENT47_STAGE_ROOT", override)

	st, err := service.prepareState(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(st.root, override) {
		t.Fatalf("expected stage root under %s, got %s", override, st.root)
	}
}

func TestRunHonorsCanceledContext(t *testing.T) {
	workDir := t.TempDir()
	service, err := New(runtime.Config{
		TemplateMode: runtime.TemplateModeFilesystem,
		RepoRoot:     newBootstrapRepoWithSkills(t, map[string]string{"analyze": validSkillBody("analyze")}),
	}, cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := service.Run(ctx, Options{WorkDir: workDir}); err != context.Canceled {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestPrepareStateDefaultsUnderWorkDir(t *testing.T) {
	service := Service{FS: fsx.Service{}, Out: cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{})}
	workDir := t.TempDir()

	st, err := service.prepareState(workDir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(st.root, workDir) {
		t.Fatalf("expected stage root under %s, got %s", workDir, st.root)
	}
}

func TestNewSucceedsWithFilesystemTemplates(t *testing.T) {
	service, err := New(runtime.Config{
		TemplateMode: runtime.TemplateModeFilesystem,
		RepoRoot:     newBootstrapRepo(t, true),
	}, cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}
	if service == nil || service.Loader == nil {
		t.Fatal("expected initialized service")
	}
}

func TestRollbackRestoresSkillsWhenCommitFailsAfterBackup(t *testing.T) {
	workDir := t.TempDir()
	service := Service{Out: cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{})}

	writeTestFile(t, filepath.Join(workDir, "skills", "analyze", "SKILL.md"), "existing skill\n")

	st := state{
		stageRoot:  filepath.Join(t.TempDir(), "stage"),
		backupRoot: filepath.Join(t.TempDir(), "backup"),
	}
	if err := os.MkdirAll(st.backupRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := service.commitSkills(workDir, &st); err == nil {
		t.Fatal("expected commitSkills to fail")
	}
	if err := service.rollback(workDir, &st); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	assertTestFileContains(t, filepath.Join(workDir, "skills", "analyze", "SKILL.md"), "existing skill")
}

func TestRollbackRestoresAgentsWhenCommitFailsAfterBackup(t *testing.T) {
	workDir := t.TempDir()
	service := Service{Out: cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{})}

	writeTestFile(t, filepath.Join(workDir, "AGENTS.md"), "existing agents\n")

	st := state{
		stageRoot:  filepath.Join(t.TempDir(), "stage"),
		backupRoot: filepath.Join(t.TempDir(), "backup"),
	}
	if err := os.MkdirAll(st.backupRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := service.commitAgents(workDir, Options{Force: true}, &st); err == nil {
		t.Fatal("expected commitAgents to fail")
	}
	if err := service.rollback(workDir, &st); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	assertTestFileContains(t, filepath.Join(workDir, "AGENTS.md"), "existing agents")
}

func TestRunRemovesStageRootAfterSuccess(t *testing.T) {
	workDir := t.TempDir()
	service, err := New(runtime.Config{
		TemplateMode: runtime.TemplateModeFilesystem,
		RepoRoot:     repoRoot(t),
	}, cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Run(context.Background(), Options{WorkDir: workDir}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(workDir, ".agent47-stage-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected no leftover stage roots, got %v", matches)
	}
}

func TestRunOnlySkillsSkipsRulesAndAgents(t *testing.T) {
	workDir := t.TempDir()
	service, err := New(runtime.Config{
		TemplateMode: runtime.TemplateModeFilesystem,
		RepoRoot:     newBootstrapRepoWithSkills(t, map[string]string{"analyze": validSkillBody("analyze"), "build": validSkillBody("build")}),
	}, cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Run(context.Background(), Options{WorkDir: workDir, OnlySkills: true}); err != nil {
		t.Fatal(err)
	}
	assertTestFileContains(t, filepath.Join(workDir, "skills", "AVAILABLE_SKILLS.xml"), "skills")
	if _, err := os.Stat(filepath.Join(workDir, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("did not expect AGENTS.md, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(workDir, "rules")); !os.IsNotExist(err) {
		t.Fatalf("did not expect rules directory, err=%v", err)
	}
}

func TestRunOnlySkillsIgnoresBrokenManifest(t *testing.T) {
	workDir := t.TempDir()
	repoRoot := t.TempDir()
	writeTestFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), "[broken]\n")
	writeTestFile(t, filepath.Join(repoRoot, "templates", "skills", "analyze", "SKILL.md"), validSkillBody("analyze"))
	service, err := New(runtime.Config{
		TemplateMode: runtime.TemplateModeFilesystem,
		RepoRoot:     repoRoot,
	}, cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Run(context.Background(), Options{WorkDir: workDir, OnlySkills: true}); err != nil {
		t.Fatal(err)
	}
	assertTestFileContains(t, filepath.Join(workDir, "skills", "AVAILABLE_SKILLS.xml"), "analyze")
}

func TestRunUsesCurrentDirectoryWhenWorkDirEmpty(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	workDir := t.TempDir()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(wd) }()

	service, err := New(runtime.Config{
		TemplateMode: runtime.TemplateModeFilesystem,
		RepoRoot:     newBootstrapRepoWithSkills(t, map[string]string{"analyze": validSkillBody("analyze"), "build": validSkillBody("build")}),
	}, cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Run(context.Background(), Options{OnlySkills: true}); err != nil {
		t.Fatal(err)
	}
	assertTestFileContains(t, filepath.Join(workDir, "skills", "AVAILABLE_SKILLS.xml"), "skills")
}

func TestRunRollsBackAndCleansStageRootOnFailure(t *testing.T) {
	workDir := t.TempDir()
	writeTestFile(t, filepath.Join(workDir, "skills", "analyze", "SKILL.md"), "existing\n")
	service, err := New(runtime.Config{
		TemplateMode: runtime.TemplateModeFilesystem,
		RepoRoot:     newBootstrapRepoWithSkills(t, map[string]string{"broken": "invalid\n"}),
	}, cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}

	err = service.Run(context.Background(), Options{WorkDir: workDir, Force: true})
	if err == nil {
		t.Fatal("expected run failure")
	}
	assertTestFileContains(t, filepath.Join(workDir, "skills", "analyze", "SKILL.md"), "existing")
	matches, globErr := filepath.Glob(filepath.Join(workDir, ".agent47-stage-*"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(matches) != 0 {
		t.Fatalf("expected stage roots to be removed, got %v", matches)
	}
}

func TestRequireTemplatesFailsWhenSkillsTemplatesMissing(t *testing.T) {
	service, err := New(runtime.Config{
		TemplateMode: runtime.TemplateModeFilesystem,
		RepoRoot:     newBootstrapRepo(t, false),
	}, cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}

	err = service.requireTemplates(manifest.Manifest{RuleTemplates: []string{"rules-backend.yaml"}}, Options{})
	if err == nil {
		t.Fatal("expected missing skills templates error")
	}
	if !strings.Contains(err.Error(), "missing skills templates") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequireTemplatesFailsWhenAgentsTemplateMissing(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestFile(t, filepath.Join(repoRoot, "templates", "rules", "rules-backend.yaml"), "rule\n")
	writeTestFile(t, filepath.Join(repoRoot, "templates", "skills", "analyze", "SKILL.md"), "---\nname: analyze\ndescription: test\n---\n")
	service, err := New(runtime.Config{
		TemplateMode: runtime.TemplateModeFilesystem,
		RepoRoot:     repoRoot,
	}, cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.requireTemplates(manifest.Manifest{RuleTemplates: []string{"rules-backend.yaml"}}, Options{}); err == nil {
		t.Fatal("expected missing AGENTS template error")
	}
}

func TestRequireTemplatesFailsWhenRuleTemplateMissing(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestFile(t, filepath.Join(repoRoot, "templates", "AGENTS.md"), "agents\n")
	writeTestFile(t, filepath.Join(repoRoot, "templates", "skills", "analyze", "SKILL.md"), "---\nname: analyze\ndescription: test\n---\n")
	service, err := New(runtime.Config{
		TemplateMode: runtime.TemplateModeFilesystem,
		RepoRoot:     repoRoot,
	}, cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.requireTemplates(manifest.Manifest{RuleTemplates: []string{"rules-backend.yaml"}}, Options{}); err == nil {
		t.Fatal("expected missing rule template error")
	}
}

func TestCommitRulesWithoutForcePreservesExistingRule(t *testing.T) {
	workDir := t.TempDir()
	service := Service{FS: fsx.Service{}, Out: cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{})}
	st := state{
		stageRoot:  filepath.Join(t.TempDir(), "stage"),
		backupRoot: filepath.Join(t.TempDir(), "backup"),
	}

	writeTestFile(t, filepath.Join(workDir, "rules", "rules-backend.yaml"), "existing rule\n")
	writeTestFile(t, filepath.Join(st.stageRoot, "rules", "rules-backend.yaml"), "managed rule\n")
	if err := os.MkdirAll(filepath.Join(st.backupRoot, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := service.commitRules(workDir, manifest.Manifest{RuleTemplates: []string{"rules-backend.yaml"}}, Options{}, &st)
	if err != nil {
		t.Fatalf("commitRules failed: %v", err)
	}

	assertTestFileContains(t, filepath.Join(workDir, "rules", "rules-backend.yaml"), "existing rule")
	if _, err := os.Stat(filepath.Join(st.stageRoot, "rules", "rules-backend.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected staged rule to be removed, got err=%v", err)
	}
}

func TestCommitRulesForceRemovesStaleAndUpdatesManagedRules(t *testing.T) {
	workDir := t.TempDir()
	service := Service{FS: fsx.Service{}, Out: cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{})}
	st := state{
		stageRoot:  filepath.Join(t.TempDir(), "stage"),
		backupRoot: filepath.Join(t.TempDir(), "backup"),
	}

	writeTestFile(t, filepath.Join(workDir, "rules", "rules-backend.yaml"), "old managed rule\n")
	writeTestFile(t, filepath.Join(workDir, "rules", "stale.yaml"), "stale managed rule\n")
	writeTestFile(t, filepath.Join(st.stageRoot, "rules", "rules-backend.yaml"), "fresh managed rule\n")
	if err := os.MkdirAll(filepath.Join(st.backupRoot, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := service.commitRules(workDir, manifest.Manifest{RuleTemplates: []string{"rules-backend.yaml"}}, Options{Force: true}, &st)
	if err != nil {
		t.Fatalf("commitRules failed: %v", err)
	}

	assertTestFileContains(t, filepath.Join(workDir, "rules", "rules-backend.yaml"), "fresh managed rule")
	if _, err := os.Stat(filepath.Join(workDir, "rules", "stale.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected stale rule removal, got err=%v", err)
	}
	assertTestFileContains(t, filepath.Join(st.backupRoot, "rules", "stale.yaml"), "stale managed rule")
}

func TestCommitAgentsWithoutForceSkipsExistingFile(t *testing.T) {
	workDir := t.TempDir()
	service := Service{FS: fsx.Service{}, Out: cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{})}
	st := state{
		stageRoot:  filepath.Join(t.TempDir(), "stage"),
		backupRoot: filepath.Join(t.TempDir(), "backup"),
	}

	writeTestFile(t, filepath.Join(workDir, "AGENTS.md"), "existing agents\n")
	writeTestFile(t, filepath.Join(st.stageRoot, "AGENTS.md"), "managed agents\n")

	if err := service.commitAgents(workDir, Options{}, &st); err != nil {
		t.Fatal(err)
	}
	assertTestFileContains(t, filepath.Join(workDir, "AGENTS.md"), "existing agents")
}

func TestCommitSkillsReplacesExistingTree(t *testing.T) {
	workDir := t.TempDir()
	service := Service{FS: fsx.Service{}, Out: cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{})}
	st := state{
		stageRoot:  filepath.Join(t.TempDir(), "stage"),
		backupRoot: filepath.Join(t.TempDir(), "backup"),
	}
	if err := os.MkdirAll(st.backupRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(workDir, "skills", "old", "SKILL.md"), "old\n")
	writeTestFile(t, filepath.Join(st.stageRoot, "skills", "new", "SKILL.md"), "new\n")

	if err := service.commitSkills(workDir, &st); err != nil {
		t.Fatal(err)
	}
	assertTestFileContains(t, filepath.Join(workDir, "skills", "new", "SKILL.md"), "new")
	if !st.replacedSkills {
		t.Fatal("expected replacedSkills flag")
	}
}

func TestCommitAgentsForceReplacesExistingFile(t *testing.T) {
	workDir := t.TempDir()
	service := Service{FS: fsx.Service{}, Out: cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{})}
	st := state{
		stageRoot:  filepath.Join(t.TempDir(), "stage"),
		backupRoot: filepath.Join(t.TempDir(), "backup"),
	}
	writeTestFile(t, filepath.Join(workDir, "AGENTS.md"), "old agents\n")
	writeTestFile(t, filepath.Join(st.stageRoot, "AGENTS.md"), "new agents\n")
	if err := os.MkdirAll(st.backupRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := service.commitAgents(workDir, Options{Force: true}, &st); err != nil {
		t.Fatal(err)
	}
	assertTestFileContains(t, filepath.Join(workDir, "AGENTS.md"), "new agents")
	assertTestFileContains(t, filepath.Join(st.backupRoot, "AGENTS.md"), "old agents")
}

func TestCommitReadmeCreatesFileWhenMissing(t *testing.T) {
	workDir := t.TempDir()
	service := Service{FS: fsx.Service{}, Out: cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{})}
	st := state{}

	if err := service.commitReadme(workDir, &st); err != nil {
		t.Fatal(err)
	}
	assertTestFileContains(t, filepath.Join(workDir, "README.md"), "")
	if !st.createdReadme {
		t.Fatal("expected createdReadme flag")
	}
}

func TestCommitReadmePreservesExistingFile(t *testing.T) {
	workDir := t.TempDir()
	service := Service{FS: fsx.Service{}, Out: cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{})}
	st := state{}
	writeTestFile(t, filepath.Join(workDir, "README.md"), "existing\n")

	if err := service.commitReadme(workDir, &st); err != nil {
		t.Fatal(err)
	}
	assertTestFileContains(t, filepath.Join(workDir, "README.md"), "existing")
	if st.createdReadme {
		t.Fatal("did not expect createdReadme flag")
	}
}

func TestRollbackRestoresRulesReadmeAndAgentsState(t *testing.T) {
	workDir := t.TempDir()
	service := Service{FS: fsx.Service{}, Out: cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{})}
	writeTestFile(t, filepath.Join(workDir, "rules", "managed.yaml"), "current\n")
	writeTestFile(t, filepath.Join(workDir, "README.md"), "readme\n")
	writeTestFile(t, filepath.Join(workDir, "AGENTS.md"), "new agents\n")

	st := state{
		backupRoot:     filepath.Join(t.TempDir(), "backup"),
		createdReadme:  true,
		replacedAgents: true,
		writtenRules:   []string{"managed.yaml"},
		removedStale:   []string{"stale.yaml"},
	}
	writeTestFile(t, filepath.Join(st.backupRoot, "AGENTS.md"), "old agents\n")
	writeTestFile(t, filepath.Join(st.backupRoot, "rules", "managed.yaml"), "old managed\n")
	writeTestFile(t, filepath.Join(st.backupRoot, "rules", "stale.yaml"), "old stale\n")

	if err := service.rollback(workDir, &st); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(workDir, "README.md")); !os.IsNotExist(err) {
		t.Fatalf("expected README rollback removal, err=%v", err)
	}
	assertTestFileContains(t, filepath.Join(workDir, "AGENTS.md"), "old agents")
	assertTestFileContains(t, filepath.Join(workDir, "rules", "managed.yaml"), "old managed")
	assertTestFileContains(t, filepath.Join(workDir, "rules", "stale.yaml"), "old stale")
}

func TestCopyTemplateDirCopiesNestedFiles(t *testing.T) {
	repoRoot := newBootstrapRepo(t, true)
	service, err := New(runtime.Config{
		TemplateMode: runtime.TemplateModeFilesystem,
		RepoRoot:     repoRoot,
	}, cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(t.TempDir(), "copied")
	if err := service.copyTemplateDir("skills", dst); err != nil {
		t.Fatal(err)
	}
	assertTestFileContains(t, filepath.Join(dst, "analyze", "SKILL.md"), "name: analyze")
}

func TestCopyTemplateDirFailsWhenSourceMissing(t *testing.T) {
	repoRoot := newBootstrapRepo(t, true)
	service, err := New(runtime.Config{
		TemplateMode: runtime.TemplateModeFilesystem,
		RepoRoot:     repoRoot,
	}, cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.copyTemplateDir("missing", filepath.Join(t.TempDir(), "dst")); err == nil {
		t.Fatal("expected copy failure for missing source")
	}
}

func TestStageSkillsPreservesInvalidExistingSkillWhenForceDisabled(t *testing.T) {
	workDir := t.TempDir()
	repoRoot := newBootstrapRepoWithSkills(t, map[string]string{
		"analyze": validSkillBody("analyze"),
		"build":   validSkillBody("build"),
	})
	service, err := New(runtime.Config{
		TemplateMode: runtime.TemplateModeFilesystem,
		RepoRoot:     repoRoot,
	}, cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(workDir, "skills", "analyze", "SKILL.md"), "invalid\n")
	st := state{stageRoot: filepath.Join(t.TempDir(), "stage")}

	if err := service.stageSkills(Options{WorkDir: workDir}, &st); err != nil {
		t.Fatal(err)
	}
	assertTestFileContains(t, filepath.Join(st.stageRoot, "skills", "analyze", "SKILL.md"), "invalid")
	assertTestFileContains(t, filepath.Join(st.stageRoot, "skills", "build", "SKILL.md"), "name: build")
	assertTestFileContains(t, filepath.Join(st.stageRoot, "skills", "AVAILABLE_SKILLS.xml"), "build")
}

func TestStageSkillsRestoresTemplateWhenSkillFileMissing(t *testing.T) {
	workDir := t.TempDir()
	repoRoot := newBootstrapRepoWithSkills(t, map[string]string{
		"analyze": validSkillBody("analyze"),
	})
	service, err := New(runtime.Config{
		TemplateMode: runtime.TemplateModeFilesystem,
		RepoRoot:     repoRoot,
	}, cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(workDir, "skills", "analyze"), 0o755); err != nil {
		t.Fatal(err)
	}
	st := state{stageRoot: filepath.Join(t.TempDir(), "stage")}

	if err := service.stageSkills(Options{WorkDir: workDir}, &st); err != nil {
		t.Fatal(err)
	}
	assertTestFileContains(t, filepath.Join(st.stageRoot, "skills", "analyze", "SKILL.md"), "name: analyze")
}

func TestStageSkillsPreservesCustomSkillsWhenForceDisabled(t *testing.T) {
	workDir := t.TempDir()
	repoRoot := newBootstrapRepoWithSkills(t, map[string]string{
		"analyze": validSkillBody("analyze"),
	})
	service, err := New(runtime.Config{
		TemplateMode: runtime.TemplateModeFilesystem,
		RepoRoot:     repoRoot,
	}, cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(workDir, "skills", "custom-tool", "SKILL.md"), validSkillBody("custom-tool"))
	st := state{stageRoot: filepath.Join(t.TempDir(), "stage")}

	if err := service.stageSkills(Options{WorkDir: workDir}, &st); err != nil {
		t.Fatal(err)
	}
	assertTestFileContains(t, filepath.Join(st.stageRoot, "skills", "custom-tool", "SKILL.md"), "name: custom-tool")
	assertTestFileContains(t, filepath.Join(st.stageRoot, "skills", "AVAILABLE_SKILLS.xml"), "custom-tool")
}

func TestStageSkillsFailsWhenForceEnabledAndTemplateInvalid(t *testing.T) {
	repoRoot := newBootstrapRepoWithSkills(t, map[string]string{
		"broken": "invalid\n",
	})
	service, err := New(runtime.Config{
		TemplateMode: runtime.TemplateModeFilesystem,
		RepoRoot:     repoRoot,
	}, cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}
	st := state{stageRoot: filepath.Join(t.TempDir(), "stage")}

	if err := service.stageSkills(Options{WorkDir: t.TempDir(), Force: true}, &st); err == nil {
		t.Fatal("expected invalid template failure")
	}
}

func TestStageRulesAndAgentsCopiesManifestTargets(t *testing.T) {
	repoRoot := newBootstrapRepo(t, true)
	service, err := New(runtime.Config{
		TemplateMode: runtime.TemplateModeFilesystem,
		RepoRoot:     repoRoot,
	}, cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}
	st := state{stageRoot: filepath.Join(t.TempDir(), "stage")}

	if err := service.stageRulesAndAgents(manifest.Manifest{RuleTemplates: []string{"rules-backend.yaml"}}, &st); err != nil {
		t.Fatal(err)
	}
	assertTestFileContains(t, filepath.Join(st.stageRoot, "rules", "rules-backend.yaml"), "rule")
	assertTestFileContains(t, filepath.Join(st.stageRoot, "AGENTS.md"), "agents")
}

func TestStageRulesAndAgentsFailsWhenAgentsTemplateMissing(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), testManifestBody())
	writeTestFile(t, filepath.Join(repoRoot, "templates", "rules", "rules-backend.yaml"), "rule\n")
	service, err := New(runtime.Config{
		TemplateMode: runtime.TemplateModeFilesystem,
		RepoRoot:     repoRoot,
	}, cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}
	st := state{stageRoot: filepath.Join(t.TempDir(), "stage")}

	if err := service.stageRulesAndAgents(manifest.Manifest{RuleTemplates: []string{"rules-backend.yaml"}}, &st); err == nil {
		t.Fatal("expected missing AGENTS template failure")
	}
}

func TestStageRulesAndAgentsFailsWhenRuleTemplateMissing(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), testManifestBody())
	writeTestFile(t, filepath.Join(repoRoot, "templates", "AGENTS.md"), "agents\n")
	service, err := New(runtime.Config{
		TemplateMode: runtime.TemplateModeFilesystem,
		RepoRoot:     repoRoot,
	}, cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}
	st := state{stageRoot: filepath.Join(t.TempDir(), "stage")}

	if err := service.stageRulesAndAgents(manifest.Manifest{RuleTemplates: []string{"rules-backend.yaml"}}, &st); err == nil {
		t.Fatal("expected missing rule template failure")
	}
}

func TestBackupFileCopiesSourceOnce(t *testing.T) {
	root := t.TempDir()
	service := Service{FS: fsx.Service{}, Out: cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{})}
	src := filepath.Join(root, "src.txt")
	dst := filepath.Join(root, "backup", "src.txt")
	writeTestFile(t, src, "source\n")

	if err := service.backupFile(src, dst); err != nil {
		t.Fatal(err)
	}
	assertTestFileContains(t, dst, "source")

	writeTestFile(t, src, "updated\n")
	if err := service.backupFile(src, dst); err != nil {
		t.Fatal(err)
	}
	assertTestFileContains(t, dst, "source")
}

func TestCollectAvailableSkillsXMLSkillsRejectsInvalidSkillWithoutForce(t *testing.T) {
	stageSkillsDir := filepath.Join(t.TempDir(), "skills")
	writeTestFile(t, filepath.Join(stageSkillsDir, "broken", "SKILL.md"), "not-frontmatter\n")

	service := Service{FS: fsx.Service{}, Out: cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{})}
	if _, err := service.collectAvailableSkillsXMLSkills(stageSkillsDir, false); err == nil {
		t.Fatal("expected invalid skill error")
	}
}

func TestCollectAvailableSkillsXMLSkillsSkipsInvalidSkillWithoutForceWhenValidSkillExists(t *testing.T) {
	stageSkillsDir := filepath.Join(t.TempDir(), "skills")
	writeTestFile(t, filepath.Join(stageSkillsDir, "valid", "SKILL.md"), "---\nname: valid\ndescription: test\n---\n")
	writeTestFile(t, filepath.Join(stageSkillsDir, "broken", "SKILL.md"), "not-frontmatter\n")

	service := Service{FS: fsx.Service{}, Out: cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{})}
	got, err := service.collectAvailableSkillsXMLSkills(stageSkillsDir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "valid" {
		t.Fatalf("unexpected skills: %#v", got)
	}
}

func TestCollectAvailableSkillsXMLSkillsFailsWithForceOnInvalidSkill(t *testing.T) {
	stageSkillsDir := filepath.Join(t.TempDir(), "skills")
	writeTestFile(t, filepath.Join(stageSkillsDir, "broken", "SKILL.md"), "not-frontmatter\n")

	service := Service{FS: fsx.Service{}, Out: cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{})}
	if _, err := service.collectAvailableSkillsXMLSkills(stageSkillsDir, true); err == nil {
		t.Fatal("expected invalid skill error with force")
	}
}

func TestCollectAvailableSkillsXMLSkillsSkipsDirWithoutSkillFile(t *testing.T) {
	stageSkillsDir := filepath.Join(t.TempDir(), "skills")
	if err := os.MkdirAll(filepath.Join(stageSkillsDir, "empty-skill"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(stageSkillsDir, "valid", "SKILL.md"), validSkillBody("valid"))

	service := Service{FS: fsx.Service{}, Out: cli.NewOutput(&bytes.Buffer{}, &bytes.Buffer{})}
	got, err := service.collectAvailableSkillsXMLSkills(stageSkillsDir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "valid" {
		t.Fatalf("unexpected skills: %#v", got)
	}
}
func writeTestFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertTestFileContains(t *testing.T, path string, fragment string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(fragment)) {
		t.Fatalf("expected %s to contain %q, got %s", path, fragment, string(data))
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

func newBootstrapRepo(t *testing.T, withSkills bool) string {
	t.Helper()

	repoRoot := t.TempDir()
	writeTestFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), testManifestBody())
	writeTestFile(t, filepath.Join(repoRoot, "templates", "AGENTS.md"), "agents\n")
	writeTestFile(t, filepath.Join(repoRoot, "templates", "rules", "rules-backend.yaml"), "rule\n")
	if withSkills {
		writeTestFile(t, filepath.Join(repoRoot, "templates", "skills", "analyze", "SKILL.md"), validSkillBody("analyze"))
	}
	return repoRoot
}

func newBootstrapRepoWithSkills(t *testing.T, skills map[string]string) string {
	t.Helper()

	repoRoot := t.TempDir()
	writeTestFile(t, filepath.Join(repoRoot, "templates", "manifest.txt"), testManifestBody())
	writeTestFile(t, filepath.Join(repoRoot, "templates", "AGENTS.md"), "agents\n")
	writeTestFile(t, filepath.Join(repoRoot, "templates", "rules", "rules-backend.yaml"), "rule\n")
	for name, body := range skills {
		writeTestFile(t, filepath.Join(repoRoot, "templates", "skills", name, "SKILL.md"), body)
	}
	return repoRoot
}

func validSkillBody(name string) string {
	return "---\nname: " + name + "\ndescription: test\n---\n"
}

func testManifestBody() string {
	return strings.Join([]string{
		"[rule_templates]",
		"rules-backend.yaml",
		"[managed_targets]",
		"templates",
		"[preserved_targets]",
		"VERSION",
		"[required_template_files]",
		"AGENTS.md",
		"[required_template_dirs]",
		"rules",
	}, "\n") + "\n"
}

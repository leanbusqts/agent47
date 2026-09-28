package doctor

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/leanbusqts/agent47/internal/cli"
	runtimecfg "github.com/leanbusqts/agent47/internal/runtime"
	"github.com/leanbusqts/agent47/internal/update"
)

func TestCheckSecurityRuleIDsDetectsDuplicates(t *testing.T) {
	templateDir := t.TempDir()
	mustWriteDoctorFile(t, filepath.Join(templateDir, "base", "rules", "security-a.yaml"), "rules:\n  -\n    id: \"SEC-test-001\"\n")
	mustWriteDoctorFile(t, filepath.Join(templateDir, "base", "rules", "security-b.yaml"), "rules:\n  -\n    id: \"SEC-test-001\"\n")

	var stderr bytes.Buffer
	service := Service{Out: cli.NewOutput(ioDiscard{}, &stderr)}
	if !service.checkSecurityRuleIDs(templateDir) {
		t.Fatal("expected duplicate security IDs warning")
	}
	if !strings.Contains(stderr.String(), "Duplicate security rule IDs detected") {
		t.Fatalf("unexpected output: %s", stderr.String())
	}
}

func TestCheckAgentsSectionsWarnsOnMissingRequiredSection(t *testing.T) {
	agentsFile := filepath.Join(t.TempDir(), "AGENTS.md")
	mustWriteDoctorFile(t, agentsFile, "## Purpose\n## Authority Order\n")

	var stderr bytes.Buffer
	service := Service{Out: cli.NewOutput(ioDiscard{}, &stderr)}
	if !service.checkAgentsSections(agentsFile) {
		t.Fatal("expected missing sections warning")
	}
	if !strings.Contains(stderr.String(), "AGENTS missing section") {
		t.Fatalf("unexpected output: %s", stderr.String())
	}
}

func TestCheckAgentsSectionsSucceedsWithRequiredSections(t *testing.T) {
	agentsFile := filepath.Join(t.TempDir(), "AGENTS.md")
	mustWriteDoctorFile(t, agentsFile, strings.Join(requiredSections, "\n")+"\n")

	var stdout bytes.Buffer
	service := Service{Out: cli.NewOutput(&stdout, ioDiscard{})}
	if service.checkAgentsSections(agentsFile) {
		t.Fatal("did not expect warning")
	}
}

func TestCheckSecurityRuleIDsSucceedsWhenUnique(t *testing.T) {
	templateDir := t.TempDir()
	mustWriteDoctorFile(t, filepath.Join(templateDir, "base", "rules", "security-a.yaml"), "rules:\n  -\n    id: \"SEC-test-001\"\n")
	mustWriteDoctorFile(t, filepath.Join(templateDir, "base", "rules", "security-b.yaml"), "rules:\n  -\n    id: \"SEC-test-002\"\n")

	var stdout bytes.Buffer
	service := Service{Out: cli.NewOutput(&stdout, ioDiscard{})}
	if service.checkSecurityRuleIDs(templateDir) {
		t.Fatal("did not expect duplicate warning")
	}
}

func TestRunHealthyUnixConfiguration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-specific symlink expectations")
	}

	baseDir := t.TempDir()
	homeDir := filepath.Join(baseDir, "home")
	agentHome := filepath.Join(homeDir, ".agent47")
	userBin := filepath.Join(homeDir, "bin")
	templateDir := filepath.Join(agentHome, "templates")
	managedBin := filepath.Join(agentHome, "bin")
	repoRoot := filepath.Join(baseDir, "repo")

	mustSeedDoctorTemplates(t, templateDir)
	mustWriteDoctorExecutable(t, filepath.Join(repoRoot, "tests", "vendor", "bats", "bin", "bats"))

	managedAfs := filepath.Join(managedBin, "afs")
	mustWriteDoctorExecutable(t, managedAfs)
	if err := os.MkdirAll(userBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(managedAfs, filepath.Join(userBin, "afs")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", userBin)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	out := cli.NewOutput(&stdout, &stderr)
	service := Service{Out: out, Update: update.New(out)}
	cfg := runtimecfg.Config{
		OS:          "darwin",
		HomeDir:     homeDir,
		UserBinDir:  userBin,
		Agent47Home: agentHome,
		RepoRoot:    repoRoot,
		Version:     "1.2.3",
	}

	if err := service.Run(context.Background(), cfg, Options{}); err != nil {
		t.Fatalf("expected healthy doctor run, got %v", err)
	}
	output := stdout.String()
	if strings.Contains(output, "[WARN]") || stderr.Len() != 0 {
		t.Fatalf("did not expect warnings: stdout=%s stderr=%s", output, stderr.String())
	}
}

func TestNewRequiresRepoRootInFilesystemMode(t *testing.T) {
	_, err := New(runtimecfg.Config{TemplateMode: runtimecfg.TemplateModeFilesystem}, cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if err == nil {
		t.Fatal("expected loader initialization error")
	}
}

func TestSymlinkMatchesAndMismatches(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink expectations differ on windows runners")
	}

	root := t.TempDir()
	target := filepath.Join(root, "target")
	link := filepath.Join(root, "link")
	other := filepath.Join(root, "other")
	mustWriteDoctorExecutable(t, target)
	mustWriteDoctorExecutable(t, other)
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if !symlinkMatches(link, target) {
		t.Fatal("expected matching symlink")
	}
	if symlinkMatches(link, other) {
		t.Fatal("did not expect mismatched symlink to match")
	}
	if symlinkMatches(filepath.Join(root, "missing"), target) {
		t.Fatal("did not expect missing symlink to match")
	}
}

func TestTemplateChecksWarnWhenFilesAreMissing(t *testing.T) {
	templateDir := t.TempDir()
	var stdout bytes.Buffer
	service := Service{Out: cli.NewOutput(&stdout, ioDiscard{})}

	if !service.checkTemplateManifest(templateDir) {
		t.Fatal("expected missing manifest warning")
	}
	if !service.checkRequiredTemplateFiles(templateDir) {
		t.Fatal("expected missing template files warning")
	}
	if !service.checkRequiredTemplateDirs(templateDir) {
		t.Fatal("expected missing template dirs warning")
	}
	if !service.checkRuleTemplates(templateDir) {
		t.Fatal("expected missing rule templates warning")
	}
	if !service.checkSecurityTemplates(templateDir) {
		t.Fatal("expected missing security template warning")
	}
}

func TestCheckTemplateManifestWarnsWhenInvalid(t *testing.T) {
	templateDir := t.TempDir()
	mustWriteDoctorFile(t, filepath.Join(templateDir, "manifest.txt"), "[broken]\n")
	var stderr bytes.Buffer
	service := Service{Out: cli.NewOutput(ioDiscard{}, &stderr)}

	if !service.checkTemplateManifest(templateDir) {
		t.Fatal("expected invalid manifest warning")
	}
	if !strings.Contains(stderr.String(), "Template manifest invalid") {
		t.Fatalf("unexpected output: %s", stderr.String())
	}
}

func TestCheckTemplateManifestWarnsWhenContractDrifts(t *testing.T) {
	templateDir := t.TempDir()
	mustWriteDoctorFile(t, filepath.Join(templateDir, "manifest.txt"), strings.Join([]string{
		"[rule_templates]",
		"security-global.yaml",
		"[managed_targets]",
		"AGENTS.md",
		"[generated_targets]",
		".agent47/context.md",
		"[preserved_targets]",
		"README.md",
		"[force_cleanup_targets]",
		"rules/",
		"[required_template_files]",
		"AGENTS.md",
		"[required_template_dirs]",
		"rules",
	}, "\n")+"\n")
	var stderr bytes.Buffer
	service := Service{Out: cli.NewOutput(ioDiscard{}, &stderr)}

	if !service.checkTemplateManifest(templateDir) {
		t.Fatal("expected manifest contract warning")
	}
	if !strings.Contains(stderr.String(), "Template manifest contract invalid") {
		t.Fatalf("unexpected output: %s", stderr.String())
	}
}

func TestCheckTemplateManifestWarnsWhenContractExpands(t *testing.T) {
	templateDir := t.TempDir()
	mustWriteDoctorFile(t, filepath.Join(templateDir, "manifest.txt"), strings.Join([]string{
		"[rule_templates]",
		"security-global.yaml",
		"[managed_targets]",
		"AGENTS.md",
		"rules/security-global.yaml",
		"rules/security-shell.yaml",
		"rules/rules-cross.yaml",
		"skills/*",
		"skills/AVAILABLE_SKILLS.xml",
		"skills/AVAILABLE_SKILLS.json",
		"skills/SUMMARY.md",
		"docs/*",
		"[generated_targets]",
		".agent47/context.md",
		"[preserved_targets]",
		"README.md",
		".agents/specs/spec.yml",
		"SNAPSHOT.md",
		"SPEC.md",
		"[force_cleanup_targets]",
		"rules/",
		"skills/",
		"prompts/",
		"specs/spec.yml",
		".agents/specs/spec.yml",
		"[required_template_files]",
		"AGENTS.md",
		"[required_template_dirs]",
		"rules",
	}, "\n")+"\n")
	var stderr bytes.Buffer
	service := Service{Out: cli.NewOutput(ioDiscard{}, &stderr)}

	if !service.checkTemplateManifest(templateDir) {
		t.Fatal("expected manifest contract warning")
	}
	if !strings.Contains(stderr.String(), "Template manifest contract invalid") {
		t.Fatalf("unexpected output: %s", stderr.String())
	}
}

func TestCheckRuleTemplatesWarnsWhenStackRuleMissing(t *testing.T) {
	templateDir := t.TempDir()
	for _, file := range catalogRuleTemplates {
		if file == "rules-backend.yaml" {
			continue
		}
		mustWriteDoctorFile(t, ruleTemplatePath(templateDir, file), "rules:\n")
	}
	var stderr bytes.Buffer
	service := Service{Out: cli.NewOutput(ioDiscard{}, &stderr)}

	if !service.checkRuleTemplates(templateDir) {
		t.Fatal("expected missing rule template warning")
	}
	if !strings.Contains(stderr.String(), "Missing rule template: rules/rules-backend.yaml") {
		t.Fatalf("unexpected output: %s", stderr.String())
	}
}

func TestRunSkipsBatsCheckOutsideSourceRepo(t *testing.T) {
	baseDir := t.TempDir()
	homeDir := filepath.Join(baseDir, "home")
	agentHome := filepath.Join(homeDir, ".agent47")
	userBin := filepath.Join(homeDir, "bin")
	templateDir := filepath.Join(agentHome, "templates")
	managedBin := filepath.Join(agentHome, "bin")

	mustSeedDoctorTemplates(t, templateDir)

	managedAfs := filepath.Join(managedBin, executableName("afs"))
	mustWriteDoctorExecutable(t, managedAfs)
	mustWriteDoctorExecutable(t, filepath.Join(userBin, executableName("afs")))
	for _, helper := range []string{"add-agent", "add-agent-prompt", "add-ss-prompt"} {
		mustWriteDoctorExecutable(t, filepath.Join(userBin, executableName(helper)))
	}
	t.Setenv("PATH", userBin)

	var stdout bytes.Buffer
	out := cli.NewOutput(&stdout, ioDiscard{})
	service := Service{Out: out, Update: update.New(out)}
	cfg := runtimecfg.Config{
		OS:          runtime.GOOS,
		HomeDir:     homeDir,
		UserBinDir:  userBin,
		Agent47Home: agentHome,
		RepoRoot:    filepath.Join(baseDir, "repo"),
		Version:     "1.2.3",
	}

	if err := service.Run(context.Background(), cfg, Options{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "bats check skipped outside the source repository") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestRunAcceptsBatsFromPath(t *testing.T) {
	baseDir := t.TempDir()
	homeDir := filepath.Join(baseDir, "home")
	agentHome := filepath.Join(homeDir, ".agent47")
	userBin := filepath.Join(homeDir, "bin")
	templateDir := filepath.Join(agentHome, "templates")
	managedBin := filepath.Join(agentHome, "bin")
	repoRoot := filepath.Join(baseDir, "repo")
	batsDir := filepath.Join(baseDir, "bats-bin")

	mustSeedDoctorTemplates(t, templateDir)
	if err := os.MkdirAll(filepath.Join(repoRoot, "tests"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteDoctorExecutable(t, filepath.Join(batsDir, executableName("bats")))
	managedAfs := filepath.Join(managedBin, executableName("afs"))
	mustWriteDoctorExecutable(t, managedAfs)
	mustWriteDoctorExecutable(t, filepath.Join(userBin, executableName("afs")))
	for _, helper := range []string{"add-agent", "add-agent-prompt", "add-ss-prompt"} {
		mustWriteDoctorExecutable(t, filepath.Join(userBin, executableName(helper)))
	}
	t.Setenv("PATH", userBin+string(os.PathListSeparator)+batsDir)

	var stdout bytes.Buffer
	out := cli.NewOutput(&stdout, ioDiscard{})
	service := Service{Out: out, Update: update.New(out)}
	cfg := runtimecfg.Config{
		OS:          runtime.GOOS,
		HomeDir:     homeDir,
		UserBinDir:  userBin,
		Agent47Home: agentHome,
		RepoRoot:    repoRoot,
		Version:     "1.2.3",
	}

	if err := service.Run(context.Background(), cfg, Options{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "bats available") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestRunWarnsWhenTemplatesMissing(t *testing.T) {
	homeDir := t.TempDir()
	userBin := filepath.Join(homeDir, "bin")
	agentHome := filepath.Join(homeDir, ".agent47")
	managedAfs := filepath.Join(agentHome, "bin", executableName("afs"))
	mustWriteDoctorExecutable(t, managedAfs)
	mustWriteDoctorExecutable(t, filepath.Join(userBin, executableName("afs")))
	for _, helper := range []string{"add-agent", "add-agent-prompt", "add-ss-prompt"} {
		mustWriteDoctorExecutable(t, filepath.Join(userBin, executableName(helper)))
	}
	t.Setenv("PATH", userBin)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	out := cli.NewOutput(&stdout, &stderr)
	service := Service{Out: out, Update: update.New(out)}
	cfg := runtimecfg.Config{
		OS:          runtime.GOOS,
		HomeDir:     homeDir,
		UserBinDir:  userBin,
		Agent47Home: agentHome,
		Version:     "1.2.3",
	}

	if err := service.Run(context.Background(), cfg, Options{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "Templates missing") {
		t.Fatalf("unexpected streams: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunCheckUpdateFailOnWarnReturnsError(t *testing.T) {
	homeDir := t.TempDir()
	cfg := runtimecfg.Config{
		OS:          runtime.GOOS,
		HomeDir:     homeDir,
		UserBinDir:  filepath.Join(homeDir, "bin"),
		Agent47Home: filepath.Join(homeDir, ".agent47"),
		Version:     "1.2.3",
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	out := cli.NewOutput(&stdout, &stderr)
	service := Service{Out: out, Update: update.New(out)}
	if err := service.Run(context.Background(), cfg, Options{CheckUpdate: true, FailOnWarn: true}); err == nil {
		t.Fatal("expected doctor warnings error")
	}
	if !strings.Contains(stderr.String(), "no update source available") {
		t.Fatalf("unexpected streams: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunWarnsOnBrokenAfsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-specific symlink expectations")
	}

	baseDir := t.TempDir()
	homeDir := filepath.Join(baseDir, "home")
	agentHome := filepath.Join(homeDir, ".agent47")
	userBin := filepath.Join(homeDir, "bin")
	templateDir := filepath.Join(agentHome, "templates")
	repoRoot := filepath.Join(baseDir, "repo")
	mustSeedDoctorTemplates(t, templateDir)
	mustWriteDoctorExecutable(t, filepath.Join(repoRoot, "tests", "vendor", "bats", "bin", "bats"))
	missingManaged := filepath.Join(agentHome, "bin", "afs")
	if err := os.MkdirAll(userBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(missingManaged, filepath.Join(userBin, "afs")); err != nil {
		t.Fatal(err)
	}
	for _, helper := range []string{"add-agent", "add-agent-prompt", "add-ss-prompt"} {
		mustWriteDoctorExecutable(t, filepath.Join(userBin, helper))
	}
	t.Setenv("PATH", userBin)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	out := cli.NewOutput(&stdout, &stderr)
	service := Service{Out: out, Update: update.New(out)}
	cfg := runtimecfg.Config{
		OS:          "darwin",
		HomeDir:     homeDir,
		UserBinDir:  userBin,
		Agent47Home: agentHome,
		RepoRoot:    repoRoot,
		Version:     "1.2.3",
	}

	if err := service.Run(context.Background(), cfg, Options{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "broken or points to a non-executable target") {
		t.Fatalf("unexpected streams: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunWarnsWhenAfsSymlinkMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-specific symlink expectations")
	}

	baseDir := t.TempDir()
	homeDir := filepath.Join(baseDir, "home")
	agentHome := filepath.Join(homeDir, ".agent47")
	userBin := filepath.Join(homeDir, "bin")
	templateDir := filepath.Join(agentHome, "templates")
	managedBin := filepath.Join(agentHome, "bin")
	repoRoot := filepath.Join(baseDir, "repo")
	mustSeedDoctorTemplates(t, templateDir)
	mustWriteDoctorExecutable(t, filepath.Join(repoRoot, "tests", "vendor", "bats", "bin", "bats"))
	managedAfs := filepath.Join(managedBin, "afs")
	mustWriteDoctorExecutable(t, managedAfs)
	for _, helper := range []string{"add-agent", "add-agent-prompt", "add-ss-prompt"} {
		mustWriteDoctorExecutable(t, filepath.Join(userBin, helper))
	}
	t.Setenv("PATH", userBin)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	out := cli.NewOutput(&stdout, &stderr)
	service := Service{Out: out, Update: update.New(out)}
	cfg := runtimecfg.Config{
		OS:          "darwin",
		HomeDir:     homeDir,
		UserBinDir:  userBin,
		Agent47Home: agentHome,
		RepoRoot:    repoRoot,
		Version:     "1.2.3",
	}

	if err := service.Run(context.Background(), cfg, Options{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "afs symlink missing") {
		t.Fatalf("unexpected streams: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunWarnsWhenBatsMissingInSourceRepo(t *testing.T) {
	baseDir := t.TempDir()
	homeDir := filepath.Join(baseDir, "home")
	agentHome := filepath.Join(homeDir, ".agent47")
	userBin := filepath.Join(homeDir, "bin")
	templateDir := filepath.Join(agentHome, "templates")
	managedBin := filepath.Join(agentHome, "bin")
	repoRoot := filepath.Join(baseDir, "repo")

	mustSeedDoctorTemplates(t, templateDir)
	if err := os.MkdirAll(filepath.Join(repoRoot, "tests"), 0o755); err != nil {
		t.Fatal(err)
	}
	managedAfs := filepath.Join(managedBin, executableName("afs"))
	mustWriteDoctorExecutable(t, managedAfs)
	mustWriteDoctorExecutable(t, filepath.Join(userBin, executableName("afs")))
	for _, helper := range []string{"add-agent", "add-agent-prompt", "add-ss-prompt"} {
		mustWriteDoctorExecutable(t, filepath.Join(userBin, executableName(helper)))
	}
	t.Setenv("PATH", userBin)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	out := cli.NewOutput(&stdout, &stderr)
	service := Service{Out: out, Update: update.New(out)}
	cfg := runtimecfg.Config{
		OS:          runtime.GOOS,
		HomeDir:     homeDir,
		UserBinDir:  userBin,
		Agent47Home: agentHome,
		RepoRoot:    repoRoot,
		Version:     "1.2.3",
	}

	if err := service.Run(context.Background(), cfg, Options{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "bats missing") {
		t.Fatalf("unexpected streams: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestResolvePathFallsBackForRegularFileAndDirectory(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file.txt")
	if err := os.WriteFile(file, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	resolvedFile, err := resolvePath(file)
	if err != nil {
		t.Fatal(err)
	}
	if resolvedFile == "" {
		t.Fatal("expected resolved regular file path")
	}
	resolvedDir, err := resolvePath(root)
	if err != nil {
		t.Fatal(err)
	}
	if resolvedDir == "" {
		t.Fatal("expected resolved dir path")
	}
}

func TestCommandMatchesManagedExecutable(t *testing.T) {
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, executableName("afs"))
	mustWriteDoctorExecutable(t, target)
	t.Setenv("PATH", tempDir)

	if !commandMatches("afs", target) {
		t.Fatalf("expected afs in PATH to match %s", target)
	}
}

func TestCommandMatchesDetectsMismatch(t *testing.T) {
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, executableName("afs"))
	other := filepath.Join(t.TempDir(), executableName("afs"))
	mustWriteDoctorExecutable(t, target)
	mustWriteDoctorExecutable(t, other)
	t.Setenv("PATH", tempDir)

	if commandMatches("afs", other) {
		t.Fatalf("did not expect afs in PATH to match %s", other)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

func mustWriteDoctorFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustWriteDoctorExecutable(t *testing.T, path string) {
	t.Helper()
	body := "#!/bin/sh\nexit 0\n"
	mode := os.FileMode(0o755)
	if runtime.GOOS == "windows" {
		body = "@echo off\r\nexit /b 0\r\n"
	}
	mustWriteDoctorFile(t, path, body)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, mode); err != nil {
			t.Fatal(err)
		}
	}
}

func mustSeedDoctorTemplates(t *testing.T, templateDir string) {
	t.Helper()

	mustWriteDoctorFile(t, filepath.Join(templateDir, "manifest.txt"), validDoctorManifest())
	mustWriteDoctorFile(t, filepath.Join(templateDir, "base", "manifest.txt"), validDoctorManifest())
	mustWriteDoctorFile(t, filepath.Join(templateDir, "base", "AGENTS.md"), strings.Join(requiredSections, "\n")+"\n")
	for _, file := range catalogRuleTemplates {
		body := "rules:\n"
		if strings.HasPrefix(file, "security-") {
			body = "rules:\n  -\n    id: \"SEC-test-" + strings.TrimSuffix(file, ".yaml") + "\"\n"
		}
		mustWriteDoctorFile(t, ruleTemplatePath(templateDir, file), body)
	}
	mustWriteDoctorFile(t, filepath.Join(templateDir, "bundles", "project-cli", "manifest.txt"), strings.Join([]string{
		"[rule_templates]",
		"rules-cli.yaml",
		"",
		"[required_template_files]",
		"rules/rules-cli.yaml",
	}, "\n")+"\n")
	mustWriteDoctorFile(t, filepath.Join(templateDir, "bundles", "project-scripts", "manifest.txt"), "[rule_templates]\nrules-scripts.yaml\n\n[required_template_files]\nrules/rules-scripts.yaml\n")
	mustWriteDoctorFile(t, filepath.Join(templateDir, "bundles", "project-backend", "manifest.txt"), "[rule_templates]\nrules-backend.yaml\n\n[required_template_files]\nrules/rules-backend.yaml\n")
	mustWriteDoctorFile(t, filepath.Join(templateDir, "bundles", "project-frontend", "manifest.txt"), "[rule_templates]\nrules-frontend.yaml\n\n[required_template_files]\nrules/rules-frontend.yaml\n")
	mustWriteDoctorFile(t, filepath.Join(templateDir, "bundles", "project-mobile", "manifest.txt"), "[rule_templates]\nrules-mobile.yaml\n\n[required_template_files]\nrules/rules-mobile.yaml\n")
	mustWriteDoctorFile(t, filepath.Join(templateDir, "bundles", "project-infra", "manifest.txt"), "[rule_templates]\nrules-infra.yaml\n\n[required_template_files]\nrules/rules-infra.yaml\n")
	mustWriteDoctorFile(t, filepath.Join(templateDir, "bundles", "project-monorepo-tooling", "manifest.txt"), "[rule_templates]\nrules-monorepo-tooling.yaml\n\n[required_template_files]\nrules/rules-monorepo-tooling.yaml\n")
	mustWriteDoctorFile(t, filepath.Join(templateDir, "bundles", "project-desktop", "manifest.txt"), "[rule_templates]\nrules-desktop.yaml\n\n[required_template_files]\nrules/rules-desktop.yaml\n")
	mustWriteDoctorFile(t, filepath.Join(templateDir, "bundles", "project-plugin", "manifest.txt"), "[rule_templates]\nrules-plugin.yaml\n\n[required_template_files]\nrules/rules-plugin.yaml\n")
	mustWriteDoctorFile(t, filepath.Join(templateDir, "bundles", "shared-cli-behavior", "manifest.txt"), "[rule_templates]\nshared-cli-behavior.yaml\n\n[required_template_files]\nrules/shared-cli-behavior.yaml\n")
	mustWriteDoctorFile(t, filepath.Join(templateDir, "bundles", "shared-testing", "manifest.txt"), "[rule_templates]\nshared-testing.yaml\n\n[required_template_files]\nrules/shared-testing.yaml\n")
}

func executableName(base string) string {
	if runtime.GOOS == "windows" {
		switch base {
		case "afs":
			return "afs.exe"
		default:
			return base + ".cmd"
		}
	}
	return base
}

func validDoctorManifest() string {
	return strings.Join([]string{
		"[rule_templates]",
		"security-global.yaml",
		"security-shell.yaml",
		"rules-cross.yaml",
		"[managed_targets]",
		"AGENTS.md",
		"rules/security-global.yaml",
		"rules/security-shell.yaml",
		"rules/rules-cross.yaml",
		"[generated_targets]",
		".agent47/context.md",
		"[preserved_targets]",
		"README.md",
		"SNAPSHOT.md",
		"SPEC.md",
		".agents/",
		"skills/",
		"prompts/",
		"[force_cleanup_targets]",
		"rules/",
		"skills/",
		"prompts/",
		"specs/spec.yml",
		".agents/specs/spec.yml",
		"[required_template_files]",
		"AGENTS.md",
		"manifest.txt",
		"[required_template_dirs]",
		"rules",
	}, "\n") + "\n"
}

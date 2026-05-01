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

	var stdout bytes.Buffer
	service := Service{Out: cli.NewOutput(&stdout, ioDiscard{})}
	if !service.checkSecurityRuleIDs(templateDir) {
		t.Fatal("expected duplicate security IDs warning")
	}
	if !strings.Contains(stdout.String(), "Duplicate security rule IDs detected") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestCheckAgentsSectionsWarnsOnMissingRequiredSection(t *testing.T) {
	agentsFile := filepath.Join(t.TempDir(), "AGENTS.md")
	mustWriteDoctorFile(t, agentsFile, "## Purpose\n## Authority Order\n")

	var stdout bytes.Buffer
	service := Service{Out: cli.NewOutput(&stdout, ioDiscard{})}
	if !service.checkAgentsSections(agentsFile) {
		t.Fatal("expected missing sections warning")
	}
	if !strings.Contains(stdout.String(), "AGENTS missing section") {
		t.Fatalf("unexpected output: %s", stdout.String())
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

func TestRunWarnsWhenSkillsTemplatesMissing(t *testing.T) {
	homeDir := t.TempDir()
	agentHome := filepath.Join(homeDir, ".agent47")
	userBin := filepath.Join(homeDir, "bin")
	templateDir := filepath.Join(agentHome, "templates")
	managedBin := filepath.Join(agentHome, "bin")

	mustSeedDoctorTemplates(t, templateDir)
	if err := os.RemoveAll(filepath.Join(templateDir, "base", "skills")); err != nil {
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
	out := cli.NewOutput(&stdout, ioDiscard{})
	service := Service{
		Out:    out,
		Update: update.New(out),
	}
	cfg := runtimecfg.Config{
		OS:          runtimecfg.Config{}.OS,
		HomeDir:     homeDir,
		UserBinDir:  userBin,
		Agent47Home: agentHome,
		Version:     "1.2.3",
	}
	if runtime.GOOS == "windows" {
		cfg.OS = "windows"
	} else {
		cfg.OS = runtime.GOOS
	}

	err := service.Run(context.Background(), cfg, Options{FailOnWarn: true})
	if err == nil {
		t.Fatal("expected doctor to fail on missing skills templates")
	}
	if !strings.Contains(stdout.String(), "Skills templates missing") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestRunWarnsWhenBundleOwnedTemplatePayloadMissing(t *testing.T) {
	cfg := runtimecfg.Config{
		Agent47Home: t.TempDir(),
		UserBinDir:  t.TempDir(),
		OS:          runtimecfg.Config{}.OS,
	}
	templateDir := filepath.Join(cfg.Agent47Home, "templates")
	mustSeedDoctorTemplates(t, templateDir)
	if err := os.Remove(filepath.Join(templateDir, "bundles", "project-cli", "skills", "cli-design", "SKILL.md")); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	out := cli.NewOutput(&stdout, ioDiscard{})
	service := Service{Out: out, Update: update.New(out)}

	err := service.Run(context.Background(), cfg, Options{})
	if err != nil {
		t.Fatalf("expected warnings only, got %v", err)
	}
	if !strings.Contains(stdout.String(), "Bundle assembly invalid") {
		t.Fatalf("expected bundle assembly warning, got %s", stdout.String())
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
	for _, helper := range []string{"add-agent", "add-agent-prompt", "add-ss-prompt"} {
		mustWriteDoctorExecutable(t, filepath.Join(userBin, helper))
	}

	t.Setenv("PATH", userBin)

	var stdout bytes.Buffer
	out := cli.NewOutput(&stdout, ioDiscard{})
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
	if strings.Contains(output, "[WARN]") {
		t.Fatalf("did not expect warnings: %s", output)
	}
	if !strings.Contains(output, "Skills templates (.md) present") {
		t.Fatalf("unexpected output: %s", output)
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
	var stdout bytes.Buffer
	service := Service{Out: cli.NewOutput(&stdout, ioDiscard{})}

	if !service.checkTemplateManifest(templateDir) {
		t.Fatal("expected invalid manifest warning")
	}
	if !strings.Contains(stdout.String(), "Template manifest invalid") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestCheckTemplateManifestWarnsWhenContractDrifts(t *testing.T) {
	templateDir := t.TempDir()
	mustWriteDoctorFile(t, filepath.Join(templateDir, "manifest.txt"), strings.Join([]string{
		"[rule_templates]",
		"security-global.yaml",
		"[managed_targets]",
		"AGENTS.md",
		"[preserved_targets]",
		"README.md",
		"[required_template_files]",
		"AGENTS.md",
		"[required_template_dirs]",
		"rules",
	}, "\n")+"\n")
	var stdout bytes.Buffer
	service := Service{Out: cli.NewOutput(&stdout, ioDiscard{})}

	if !service.checkTemplateManifest(templateDir) {
		t.Fatal("expected manifest contract warning")
	}
	if !strings.Contains(stdout.String(), "Template manifest contract invalid") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestCheckTemplateManifestWarnsWhenContractExpands(t *testing.T) {
	templateDir := t.TempDir()
	mustWriteDoctorFile(t, filepath.Join(templateDir, "manifest.txt"), strings.Join([]string{
		"[rule_templates]",
		"security-global.yaml",
		"[managed_targets]",
		"AGENTS.md",
		"rules/*.yaml",
		"skills/*",
		"skills/AVAILABLE_SKILLS.xml",
		"skills/AVAILABLE_SKILLS.json",
		"skills/SUMMARY.md",
		"docs/*",
		"[preserved_targets]",
		"README.md",
		"specs/spec.yml",
		"SNAPSHOT.md",
		"SPEC.md",
		"[required_template_files]",
		"AGENTS.md",
		"[required_template_dirs]",
		"rules",
	}, "\n")+"\n")
	var stdout bytes.Buffer
	service := Service{Out: cli.NewOutput(&stdout, ioDiscard{})}

	if !service.checkTemplateManifest(templateDir) {
		t.Fatal("expected manifest contract warning")
	}
	if !strings.Contains(stdout.String(), "Template manifest contract invalid") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestCheckRequiredTemplateFilesWarnsWhenSpecMissing(t *testing.T) {
	templateDir := t.TempDir()
	mustWriteDoctorFile(t, filepath.Join(templateDir, "base", "AGENTS.md"), "agents\n")
	mustWriteDoctorFile(t, filepath.Join(templateDir, "manifest.txt"), "manifest\n")
	var stdout bytes.Buffer
	service := Service{Out: cli.NewOutput(&stdout, ioDiscard{})}

	if !service.checkRequiredTemplateFiles(templateDir) {
		t.Fatal("expected missing template files warning")
	}
	if !strings.Contains(stdout.String(), "Missing template file: specs/spec.yml") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestCheckRequiredTemplateDirsWarnsWhenSpecsDirMissing(t *testing.T) {
	templateDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(templateDir, "base", "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(templateDir, "base", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	service := Service{Out: cli.NewOutput(&stdout, ioDiscard{})}

	if !service.checkRequiredTemplateDirs(templateDir) {
		t.Fatal("expected missing template dirs warning")
	}
	if !strings.Contains(stdout.String(), "Missing template dir: specs") {
		t.Fatalf("unexpected output: %s", stdout.String())
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
	var stdout bytes.Buffer
	service := Service{Out: cli.NewOutput(&stdout, ioDiscard{})}

	if !service.checkRuleTemplates(templateDir) {
		t.Fatal("expected missing rule template warning")
	}
	if !strings.Contains(stdout.String(), "Missing rule template: rules/rules-backend.yaml") {
		t.Fatalf("unexpected output: %s", stdout.String())
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
	out := cli.NewOutput(&stdout, ioDiscard{})
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
	if !strings.Contains(stdout.String(), "Templates missing") {
		t.Fatalf("unexpected output: %s", stdout.String())
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
	out := cli.NewOutput(&stdout, ioDiscard{})
	service := Service{Out: out, Update: update.New(out)}
	if err := service.Run(context.Background(), cfg, Options{CheckUpdate: true, FailOnWarn: true}); err == nil {
		t.Fatal("expected doctor warnings error")
	}
	if !strings.Contains(stdout.String(), "no update source available") {
		t.Fatalf("unexpected output: %s", stdout.String())
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
	out := cli.NewOutput(&stdout, ioDiscard{})
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
	if !strings.Contains(stdout.String(), "broken or points to a non-executable target") {
		t.Fatalf("unexpected output: %s", stdout.String())
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
	out := cli.NewOutput(&stdout, ioDiscard{})
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
	if !strings.Contains(stdout.String(), "afs symlink missing") {
		t.Fatalf("unexpected output: %s", stdout.String())
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
	if !strings.Contains(stdout.String(), "bats missing") {
		t.Fatalf("unexpected output: %s", stdout.String())
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

func TestHelperMatchesPublishedHelper(t *testing.T) {
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, executableName("add-agent"))
	mustWriteDoctorExecutable(t, target)
	t.Setenv("PATH", tempDir)

	if !helperMatches("add-agent", filepath.Join(t.TempDir(), executableName("managed-add-agent")), target) {
		t.Fatalf("expected add-agent in PATH to match published helper %s", target)
	}
}

func TestHelperMatchesDetectsMismatch(t *testing.T) {
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, executableName("add-agent"))
	mustWriteDoctorExecutable(t, target)
	t.Setenv("PATH", tempDir)

	if helperMatches("add-agent", filepath.Join(t.TempDir(), executableName("managed-add-agent")), filepath.Join(t.TempDir(), executableName("published-add-agent"))) {
		t.Fatal("did not expect helper match")
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
	mustWriteDoctorFile(t, filepath.Join(templateDir, "base", "prompts", "agent-prompt.txt"), "agent prompt\n")
	mustWriteDoctorFile(t, filepath.Join(templateDir, "base", "prompts", "ss-prompt.txt"), "ss prompt\n")
	mustWriteDoctorFile(t, filepath.Join(templateDir, "base", "specs", "spec.yml"), "summary: test\n")
	mustWriteDoctorFile(t, filepath.Join(templateDir, "base", "skills", "analyze", "SKILL.md"), "---\nname: analyze\ndescription: test\n---\n")
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
		"skills/cli-design/SKILL.md",
	}, "\n")+"\n")
	mustWriteDoctorFile(t, filepath.Join(templateDir, "bundles", "project-cli", "skills", "cli-design", "SKILL.md"), "---\nname: cli-design\ndescription: test\n---\n")
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
		"[managed_targets]",
		"AGENTS.md",
		"rules/*.yaml",
		"skills/*",
		"skills/AVAILABLE_SKILLS.xml",
		"skills/AVAILABLE_SKILLS.json",
		"skills/SUMMARY.md",
		"[preserved_targets]",
		"README.md",
		"specs/spec.yml",
		"SNAPSHOT.md",
		"SPEC.md",
		"[required_template_files]",
		"AGENTS.md",
		"manifest.txt",
		"specs/spec.yml",
		"[required_template_dirs]",
		"rules",
		"skills",
		"specs",
	}, "\n") + "\n"
}

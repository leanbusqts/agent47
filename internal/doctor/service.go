package doctor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goRuntime "runtime"
	"strings"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/install"
	"github.com/leanbusqts/agent47/internal/manifest"
	"github.com/leanbusqts/agent47/internal/runtime"
	"github.com/leanbusqts/agent47/internal/templates"
	"github.com/leanbusqts/agent47/internal/update"
)

type WarningsError struct{}

func (WarningsError) Error() string {
	return "doctor reported warnings"
}

var (
	requiredSections = []string{
		"## Purpose",
		"## Authority Order",
		"## Executable Commands",
		"## Filesystem And Approval Boundaries",
		"### Always",
		"### Ask",
		"### Never",
		"## Security Expectations",
	}
	securityTemplateFiles = []string{
		"security-global.yaml",
		"security-shell.yaml",
		"security-js-ts.yaml",
		"security-py.yaml",
		"security-java-kotlin.yaml",
		"security-swift.yaml",
		"security-csharp.yaml",
	}
	requiredRuleTemplates = []string{
		"security-global.yaml",
		"security-shell.yaml",
	}
	catalogRuleTemplates = []string{
		"rules-cli.yaml",
		"rules-scripts.yaml",
		"rules-mobile.yaml",
		"rules-frontend.yaml",
		"rules-backend.yaml",
		"rules-infra.yaml",
		"rules-monorepo-tooling.yaml",
		"rules-desktop.yaml",
		"rules-plugin.yaml",
		"shared-cli-behavior.yaml",
		"shared-testing.yaml",
		"security-global.yaml",
		"security-shell.yaml",
		"security-js-ts.yaml",
		"security-py.yaml",
		"security-java-kotlin.yaml",
		"security-swift.yaml",
		"security-csharp.yaml",
	}
	requiredManagedTargets = []string{
		"AGENTS.md",
		"rules/*.yaml",
		"skills/*",
		"skills/AVAILABLE_SKILLS.xml",
		"skills/AVAILABLE_SKILLS.json",
		"skills/SUMMARY.md",
	}
	requiredPreservedTargets = []string{
		"README.md",
		"specs/spec.yml",
		"SNAPSHOT.md",
		"SPEC.md",
	}
	requiredTemplateFiles = []string{
		"AGENTS.md",
		"manifest.txt",
		"specs/spec.yml",
	}
	requiredTemplateDirs = []string{
		"rules",
		"skills",
		"specs",
	}
)

type Service struct {
	Loader *templates.Loader
	Out    cli.Output
	Update *update.Service
}

type Options struct {
	CheckUpdate bool
	ForceUpdate bool
	FailOnWarn  bool
}

func New(cfg runtime.Config, out cli.Output) (*Service, error) {
	loader, err := templates.NewLoader(cfg.TemplateMode, cfg.RepoRoot)
	if err != nil {
		return nil, err
	}

	return &Service{
		Loader: loader,
		Out:    out,
		Update: update.New(out),
	}, nil
}

func (s *Service) Run(ctx context.Context, cfg runtime.Config, opts Options) error {
	hadWarn := false

	s.Out.Printf("[*] afs doctor\n")
	s.Out.Info("Version: %s", cfg.Version)

	managedAfs := install.ManagedBinaryPathForDoctor(cfg)
	if commandMatches("afs", managedAfs) {
		s.Out.OK("afs in PATH")
	} else if _, err := exec.LookPath("afs"); err == nil {
		hadWarn = true
		s.Out.Warn("afs in PATH, but not the managed launcher")
		s.Out.Info(install.ReinstallHint(cfg))
	} else {
		hadWarn = true
		s.Out.Warn("afs not in PATH")
		s.Out.Info(install.ReinstallHint(cfg))
	}

	for _, script := range install.HelperCommands() {
		if helperMatches(script, managedAfs, install.PublishedHelperPathForDoctor(cfg, script)) {
			s.Out.OK("%s available", script)
		} else if _, err := exec.LookPath(script); err == nil {
			hadWarn = true
			s.Out.Warn("%s in PATH, but not the managed installed copy", script)
			s.Out.Info(install.ReinstallHint(cfg))
		} else {
			hadWarn = true
			s.Out.Warn("%s missing", script)
			s.Out.Info(install.ReinstallHint(cfg))
		}
	}

	templateDir := filepath.Join(cfg.Agent47Home, "templates")
	if info, err := os.Stat(templateDir); err == nil && info.IsDir() {
		s.Out.OK("Templates installed")
		hadWarn = s.checkTemplateManifest(templateDir) || hadWarn
		hadWarn = s.checkBundleAssembly(templateDir) || hadWarn
		hadWarn = s.checkRequiredTemplateFiles(templateDir) || hadWarn
		hadWarn = s.checkRequiredTemplateDirs(templateDir) || hadWarn
		hadWarn = s.checkRuleTemplates(templateDir) || hadWarn
		hadWarn = s.checkSecurityTemplates(templateDir) || hadWarn
		hadWarn = s.checkSecurityRuleIDs(templateDir) || hadWarn
		hadWarn = s.checkAgentsSections(layoutPath(templateDir, "AGENTS.md")) || hadWarn
	} else {
		hadWarn = true
		s.Out.Warn("Templates missing")
		s.Out.Info(install.ReinstallHint(cfg))
	}

	skillsDir := layoutPath(templateDir, "skills")
	if info, err := os.Stat(skillsDir); err == nil && info.IsDir() {
		s.Out.OK("Skills templates (.md) present")
	} else {
		hadWarn = true
		s.Out.Warn("Skills templates missing")
	}

	if cfg.RepoRoot != "" {
		if _, err := os.Stat(filepath.Join(cfg.RepoRoot, "tests")); err == nil {
			if _, err := os.Stat(filepath.Join(cfg.RepoRoot, "tests", "vendor", "bats", "bin", "bats")); err == nil {
				s.Out.OK("bats available")
			} else if _, err := exec.LookPath("bats"); err == nil {
				s.Out.OK("bats available")
			} else {
				hadWarn = true
				s.Out.Warn("bats missing")
			}
		} else {
			s.Out.Info("bats check skipped outside the source repository")
		}
	}

	userAfs := install.PublishedAfsPathForDoctor(cfg)
	if cfg.OS == "windows" {
		if install.PathContains(cfg.OS, cfg.UserBinDir) {
			s.Out.OK("afs launcher present in managed bin")
		} else {
			hadWarn = true
			s.Out.Warn("afs launcher missing from PATH")
			s.Out.Info(install.ReinstallHint(cfg))
		}
	} else if symlinkMatches(userAfs, managedAfs) {
		s.Out.OK("afs symlink present in ~/bin")
	} else if isSymlink(userAfs) && resolvesExecutable(userAfs) {
		hadWarn = true
		s.Out.Warn("afs symlink in ~/bin points to the wrong executable")
		s.Out.Info(install.ReinstallHint(cfg))
	} else if isSymlink(userAfs) {
		hadWarn = true
		s.Out.Warn("afs symlink in ~/bin is broken or points to a non-executable target")
		s.Out.Info(install.ReinstallHint(cfg))
	} else {
		hadWarn = true
		s.Out.Warn("afs symlink missing")
		s.Out.Info(install.ReinstallHint(cfg))
	}

	if install.PathContains(cfg.OS, cfg.UserBinDir) {
		if cfg.OS == "windows" {
			s.Out.OK("managed bin in PATH")
		} else {
			s.Out.OK("~/bin in PATH")
		}
	} else {
		hadWarn = true
		if cfg.OS == "windows" {
			s.Out.Warn("managed bin not in PATH")
			s.Out.Info("Add to your user PATH: %s", cfg.UserBinDir)
		} else {
			s.Out.Warn("~/bin not in PATH")
			s.Out.Info("Add to your shell config:")
			s.Out.Printf("       export PATH=\"$HOME/bin:$PATH\"\n")
		}
	}

	if opts.CheckUpdate {
		if err := s.Update.Check(ctx, cfg, update.CheckOptions{Force: opts.ForceUpdate}); err != nil {
			return err
		}
		if opts.FailOnWarn && hadWarn {
			return WarningsError{}
		}
		return nil
	}

	s.Out.Info("Skipping update check by default")
	s.Out.Info("Run: afs doctor --check-update")
	if opts.FailOnWarn && hadWarn {
		return WarningsError{}
	}
	return nil
}

func resolvePath(target string) (string, error) {
	resolved, err := filepath.EvalSymlinks(target)
	if err == nil {
		return resolved, nil
	}
	if info, statErr := os.Stat(target); statErr == nil && info.IsDir() {
		return filepath.Abs(target)
	}
	if _, statErr := os.Stat(target); statErr == nil {
		return filepath.Abs(target)
	}
	return "", err
}

func commandMatches(name, managedTarget string) bool {
	actualPath, err := exec.LookPath(name)
	if err != nil {
		return false
	}
	actualResolved, err := resolvePath(actualPath)
	if err != nil {
		return false
	}
	expectedResolved, err := resolvePath(managedTarget)
	if err != nil {
		return false
	}
	return sameResolvedPath(actualResolved, expectedResolved)
}

func helperMatches(name, managedTarget, userTarget string) bool {
	actualPath, err := exec.LookPath(name)
	if err != nil {
		return false
	}
	actualResolved, err := resolvePath(actualPath)
	if err == nil {
		if managedResolved, managedErr := resolvePath(managedTarget); managedErr == nil && sameResolvedPath(actualResolved, managedResolved) {
			return true
		}
		if userResolved, userErr := resolvePath(userTarget); userErr == nil && sameResolvedPath(actualResolved, userResolved) {
			return true
		}
	}
	return false
}

func resolvesExecutable(path string) bool {
	resolved, err := resolvePath(path)
	if err != nil {
		return false
	}
	info, err := os.Stat(resolved)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}

func symlinkMatches(linkPath, expectedTarget string) bool {
	if !isSymlink(linkPath) {
		return false
	}
	resolvedLink, err := resolvePath(linkPath)
	if err != nil {
		return false
	}
	resolvedTarget, err := resolvePath(expectedTarget)
	if err != nil {
		return false
	}
	return sameResolvedPath(resolvedLink, resolvedTarget)
}

func sameResolvedPath(left, right string) bool {
	leftClean := filepath.Clean(left)
	rightClean := filepath.Clean(right)
	if goRuntime.GOOS == "windows" {
		return strings.EqualFold(leftClean, rightClean)
	}
	return leftClean == rightClean
}

func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

func (s *Service) checkTemplateManifest(templateDir string) bool {
	data, err := os.ReadFile(filepath.Join(templateDir, "manifest.txt"))
	if err != nil {
		s.Out.Warn("Template manifest missing")
		return true
	}
	m, err := manifest.Parse(data)
	if err != nil {
		s.Out.Warn("Template manifest invalid")
		return true
	}
	if !matchesManifestTargets(m.ManagedTargets, requiredManagedTargets) {
		s.Out.Warn("Template manifest contract invalid")
		return true
	}
	if !matchesManifestTargets(m.PreservedTargets, requiredPreservedTargets) {
		s.Out.Warn("Template manifest contract invalid")
		return true
	}
	if !matchesManifestTargets(m.RuleTemplates, requiredRuleTemplates) {
		s.Out.Warn("Template manifest contract invalid")
		return true
	}
	if !matchesManifestTargets(m.RequiredTemplateFiles, requiredTemplateFiles) {
		s.Out.Warn("Template manifest contract invalid")
		return true
	}
	if !matchesManifestTargets(m.RequiredTemplateDirs, requiredTemplateDirs) {
		s.Out.Warn("Template manifest contract invalid")
		return true
	}
	s.Out.OK("Template manifest present")
	return false
}

func (s *Service) checkBundleAssembly(templateDir string) bool {
	rawSource := templates.NewFilesystemSource(templateDir)
	if _, err := rawSource.Stat("base/manifest.txt"); err != nil {
		s.Out.Warn("Base manifest missing")
		return true
	}
	bundleIDs, err := templates.DiscoverBundleIDs(rawSource)
	if err != nil {
		s.Out.Warn("Bundle manifest discovery failed")
		return true
	}
	if err := templates.ValidateAssembly(rawSource, bundleIDs); err != nil {
		s.Out.Warn("Bundle assembly invalid: %v", err)
		return true
	}
	assembledManifest, err := templates.AssembleManifest(rawSource, bundleIDs)
	if err != nil {
		s.Out.Warn("Bundle manifests invalid: %v", err)
		return true
	}
	if err := validateAssembledTemplateSource(templates.AssembleSource(rawSource, bundleIDs), assembledManifest); err != nil {
		s.Out.Warn("Bundle assembly invalid: %v", err)
		return true
	}
	s.Out.OK("Bundle manifests and assembly valid")
	return false
}

func validateAssembledTemplateSource(src templates.Source, m manifest.Manifest) error {
	for _, relPath := range m.RequiredTemplateFiles {
		if relPath == "manifest.txt" {
			continue
		}
		if _, err := src.Stat(relPath); err != nil {
			return err
		}
	}
	for _, dirPath := range m.RequiredTemplateDirs {
		info, err := src.Stat(dirPath)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", dirPath)
		}
	}
	for _, rule := range m.RuleTemplates {
		if _, err := src.Stat(filepath.ToSlash(filepath.Join("rules", rule))); err != nil {
			return err
		}
	}
	return nil
}

func matchesManifestTargets(actual, required []string) bool {
	if len(actual) != len(required) {
		return false
	}

	set := map[string]bool{}
	for _, item := range actual {
		set[item] = true
	}
	for _, item := range required {
		if !set[item] {
			return false
		}
	}
	return true
}

func (s *Service) checkRequiredTemplateFiles(templateDir string) bool {
	missing := false
	for _, relPath := range requiredTemplateFiles {
		if _, err := os.Stat(layoutPath(templateDir, relPath)); err != nil {
			s.Out.Warn("Missing template file: %s", relPath)
			missing = true
		}
	}
	if !missing {
		s.Out.OK("Required template files present")
	}
	return missing
}

func (s *Service) checkRequiredTemplateDirs(templateDir string) bool {
	missing := false
	for _, relPath := range requiredTemplateDirs {
		info, err := os.Stat(layoutPath(templateDir, relPath))
		if err != nil || !info.IsDir() {
			s.Out.Warn("Missing template dir: %s", relPath)
			missing = true
		}
	}
	if !missing {
		s.Out.OK("Required template dirs present")
	}
	return missing
}

func (s *Service) checkRuleTemplates(templateDir string) bool {
	missing := false
	for _, file := range catalogRuleTemplates {
		if _, err := os.Stat(ruleTemplatePath(templateDir, file)); err != nil {
			s.Out.Warn("Missing rule template: rules/%s", file)
			missing = true
		}
	}
	if !missing {
		s.Out.OK("Rule templates present")
	}
	return missing
}

func (s *Service) checkSecurityTemplates(templateDir string) bool {
	missing := false
	for _, file := range securityTemplateFiles {
		if _, err := os.Stat(filepath.Join(templateDir, "base", "rules", file)); err != nil {
			s.Out.Warn("Missing security template: rules/%s", file)
			missing = true
		}
	}
	if !missing {
		s.Out.OK("Security templates present")
	}
	return missing
}

func (s *Service) checkSecurityRuleIDs(templateDir string) bool {
	ruleFiles, _ := filepath.Glob(filepath.Join(templateDir, "base", "rules", "security-*.yaml"))
	seen := map[string]bool{}
	dupes := map[string]bool{}
	for _, file := range ruleFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "id:") || !strings.Contains(line, "\"SEC-") {
				continue
			}
			start := strings.Index(line, "\"")
			end := strings.LastIndex(line, "\"")
			if start >= 0 && end > start {
				id := line[start+1 : end]
				if seen[id] {
					dupes[id] = true
				}
				seen[id] = true
			}
		}
	}
	if len(dupes) > 0 {
		s.Out.Warn("Duplicate security rule IDs detected")
		return true
	}
	s.Out.OK("Security rule IDs unique")
	return false
}

func (s *Service) checkAgentsSections(agentsFile string) bool {
	data, err := os.ReadFile(agentsFile)
	if err != nil {
		s.Out.Warn("AGENTS.md missing")
		return true
	}
	content := string(data)
	for _, section := range requiredSections {
		if !strings.Contains(content, section) {
			s.Out.Warn("AGENTS missing section: %s", section)
			return true
		}
	}
	s.Out.OK("AGENTS required sections present")
	return false
}

func layoutPath(templateDir, relPath string) string {
	if filepath.ToSlash(relPath) == "manifest.txt" {
		return filepath.Join(templateDir, "manifest.txt")
	}
	return filepath.Join(templateDir, "base", filepath.FromSlash(relPath))
}

func ruleTemplatePath(templateDir, file string) string {
	switch file {
	case "rules-cli.yaml":
		return filepath.Join(templateDir, "bundles", "project-cli", "rules", file)
	case "rules-scripts.yaml":
		return filepath.Join(templateDir, "bundles", "project-scripts", "rules", file)
	case "rules-mobile.yaml":
		return filepath.Join(templateDir, "bundles", "project-mobile", "rules", file)
	case "rules-frontend.yaml":
		return filepath.Join(templateDir, "bundles", "project-frontend", "rules", file)
	case "rules-backend.yaml":
		return filepath.Join(templateDir, "bundles", "project-backend", "rules", file)
	case "rules-infra.yaml":
		return filepath.Join(templateDir, "bundles", "project-infra", "rules", file)
	case "rules-monorepo-tooling.yaml":
		return filepath.Join(templateDir, "bundles", "project-monorepo-tooling", "rules", file)
	case "rules-desktop.yaml":
		return filepath.Join(templateDir, "bundles", "project-desktop", "rules", file)
	case "rules-plugin.yaml":
		return filepath.Join(templateDir, "bundles", "project-plugin", "rules", file)
	case "shared-cli-behavior.yaml":
		return filepath.Join(templateDir, "bundles", "shared-cli-behavior", "rules", file)
	case "shared-testing.yaml":
		return filepath.Join(templateDir, "bundles", "shared-testing", "rules", file)
	}
	return filepath.Join(templateDir, "base", "rules", file)
}

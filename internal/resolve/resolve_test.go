package resolve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/leanbusqts/agent47/internal/analyze"
	"github.com/leanbusqts/agent47/internal/manifest"
)

func TestResolveLowSignalFallsBackToBaseBundle(t *testing.T) {
	set, err := Resolve(analyze.AnalysisResult{LowSignal: true}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Bundles) != 1 || set.Bundles[0] != "base" {
		t.Fatalf("expected base bundle only, got %v", set.Bundles)
	}
	if len(set.Prompts) != 0 {
		t.Fatalf("did not expect default scaffold prompts, got %v", set.Prompts)
	}
}

func TestResolveSupportsCLIScriptsComposition(t *testing.T) {
	set, err := Resolve(analyze.AnalysisResult{
		ProjectTypes: []analyze.DetectedProjectType{
			{ID: "cli", Confidence: analyze.ConfidenceHigh},
			{ID: "scripts", Confidence: analyze.ConfidenceHigh},
		},
		Technologies: []analyze.DetectedTechnology{
			{ID: "go", Confidence: analyze.ConfidenceHigh},
			{ID: "shell", Confidence: analyze.ConfidenceHigh},
		},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"base", "project-cli", "project-scripts", "shared-cli-behavior", "shared-testing"}
	if len(set.Bundles) != len(want) {
		t.Fatalf("expected bundles %v, got %v", want, set.Bundles)
	}
	for i := range want {
		if set.Bundles[i] != want[i] {
			t.Fatalf("expected bundles %v, got %v", want, set.Bundles)
		}
	}
}

func TestResolveSupportsCLIMonorepoToolingComposition(t *testing.T) {
	set, err := Resolve(analyze.AnalysisResult{
		ProjectTypes: []analyze.DetectedProjectType{
			{ID: "cli", Confidence: analyze.ConfidenceHigh},
			{ID: "monorepo-tooling", Confidence: analyze.ConfidenceHigh},
		},
		Technologies: []analyze.DetectedTechnology{
			{ID: "workspace-tooling", Confidence: analyze.ConfidenceHigh},
		},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"base", "project-cli", "project-monorepo-tooling", "shared-cli-behavior", "shared-testing"}
	if len(set.Bundles) != len(want) {
		t.Fatalf("expected bundles %v, got %v", want, set.Bundles)
	}
	for i := range want {
		if set.Bundles[i] != want[i] {
			t.Fatalf("expected bundles %v, got %v", want, set.Bundles)
		}
	}
	if !containsString(set.Rules, "shared-cli-behavior.yaml") {
		t.Fatalf("expected shared CLI behavior rule, got %v", set.Rules)
	}
}

func TestResolveSupportsPluginDesktopComposition(t *testing.T) {
	set, err := Resolve(analyze.AnalysisResult{
		ProjectTypes: []analyze.DetectedProjectType{
			{ID: "desktop", Confidence: analyze.ConfidenceHigh},
			{ID: "plugin", Confidence: analyze.ConfidenceHigh},
		},
		Technologies: []analyze.DetectedTechnology{
			{ID: "desktop-runtime", Confidence: analyze.ConfidenceHigh},
			{ID: "plugin-hosting", Confidence: analyze.ConfidenceHigh},
		},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"base", "project-desktop", "project-plugin", "shared-testing"}
	if len(set.Bundles) != len(want) {
		t.Fatalf("expected bundles %v, got %v", want, set.Bundles)
	}
	for i := range want {
		if set.Bundles[i] != want[i] {
			t.Fatalf("expected bundles %v, got %v", want, set.Bundles)
		}
	}
	if !containsString(set.Rules, "rules-plugin.yaml") || !containsString(set.Rules, "rules-desktop.yaml") {
		t.Fatalf("expected desktop and plugin rules, got %v", set.Rules)
	}
	if !containsString(set.Rules, "shared-testing.yaml") {
		t.Fatalf("expected shared testing rule, got %v", set.Rules)
	}
}

func TestResolveSupportsInfraBundle(t *testing.T) {
	set, err := Resolve(analyze.AnalysisResult{
		ProjectTypes: []analyze.DetectedProjectType{{ID: "infra", Confidence: analyze.ConfidenceHigh}},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"base", "project-infra"}
	if len(set.Bundles) != len(want) {
		t.Fatalf("expected bundles %v, got %v", want, set.Bundles)
	}
	for i := range want {
		if set.Bundles[i] != want[i] {
			t.Fatalf("expected bundles %v, got %v", want, set.Bundles)
		}
	}
}

func TestResolveAddsSharedTestingForFrontendBundles(t *testing.T) {
	set, err := Resolve(analyze.AnalysisResult{
		ProjectTypes: []analyze.DetectedProjectType{
			{ID: "frontend", Confidence: analyze.ConfidenceHigh},
		},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(set.Bundles, "shared-testing") {
		t.Fatalf("expected shared testing bundle, got %v", set.Bundles)
	}
	if !containsString(set.Rules, "shared-testing.yaml") {
		t.Fatalf("expected shared testing rule, got %v", set.Rules)
	}
}

func TestResolveMapsTestingTechnologiesToSkills(t *testing.T) {
	set, err := Resolve(analyze.AnalysisResult{
		Technologies: []analyze.DetectedTechnology{
			{ID: "vitest", Confidence: analyze.ConfidenceHigh},
			{ID: "playwright", Confidence: analyze.ConfidenceHigh},
			{ID: "go-test", Confidence: analyze.ConfidenceHigh},
		},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}

	if !containsString(set.Skills, "refactor") {
		t.Fatalf("expected refactor skill from testing tech mapping, got %v", set.Skills)
	}
	if !containsString(set.Skills, "optimize") {
		t.Fatalf("expected optimize skill from testing tech mapping, got %v", set.Skills)
	}
}

func TestResolveMapsInitialDetectedTechnologiesToSkills(t *testing.T) {
	set, err := Resolve(analyze.AnalysisResult{
		Technologies: []analyze.DetectedTechnology{
			{ID: "node", Confidence: analyze.ConfidenceHigh},
			{ID: "tailwind", Confidence: analyze.ConfidenceMedium},
			{ID: "workspace-tooling", Confidence: analyze.ConfidenceMedium},
		},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}

	if !containsString(set.Skills, "refactor") {
		t.Fatalf("expected refactor for node mapping, got %v", set.Skills)
	}
	if !containsString(set.Skills, "optimize") {
		t.Fatalf("expected optimize for tailwind mapping, got %v", set.Skills)
	}
	if !containsString(set.Skills, "troubleshoot") {
		t.Fatalf("expected troubleshoot for workspace-tooling mapping, got %v", set.Skills)
	}
}

func TestResolveUnresolvedConflictFallsBackToBaseBundle(t *testing.T) {
	set, err := Resolve(analyze.AnalysisResult{
		ProjectTypes: []analyze.DetectedProjectType{
			{ID: "backend", Confidence: analyze.ConfidenceHigh},
			{ID: "frontend", Confidence: analyze.ConfidenceHigh},
		},
		UnresolvedConflict:   true,
		ConflictProjectTypes: []string{"backend", "frontend"},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Bundles) != 1 || set.Bundles[0] != "base" {
		t.Fatalf("expected base bundle only, got %v", set.Bundles)
	}
	if !set.UnresolvedConflict {
		t.Fatal("expected unresolved conflict in install set")
	}
}

func TestResolveRejectsIncompatibleExplicitBundles(t *testing.T) {
	_, err := Resolve(analyze.AnalysisResult{}, Options{ExplicitBundles: []string{"frontend", "backend"}})
	if err == nil {
		t.Fatal("expected incompatible explicit bundle error")
	}
}

func TestResolveAcceptsCompatibleExplicitBundles(t *testing.T) {
	_, err := Resolve(analyze.AnalysisResult{}, Options{ExplicitBundles: []string{"plugin", "desktop"}})
	if err != nil {
		t.Fatalf("expected compatible explicit bundle selection, got %v", err)
	}
}

func TestResolveAppliesExclusionsToExplicitBundles(t *testing.T) {
	set, err := Resolve(analyze.AnalysisResult{}, Options{
		ExplicitBundles: []string{"cli", "scripts"},
		ExcludeBundles:  []string{"cli"},
	})
	if err != nil {
		t.Fatalf("expected explicit bundle exclusion to succeed, got %v", err)
	}
	if containsString(set.Bundles, "project-cli") {
		t.Fatalf("did not expect excluded explicit bundle in result: %v", set.Bundles)
	}
	if containsString(set.Bundles, "shared-cli-behavior") {
		t.Fatalf("did not expect CLI shared dependency after exclusion: %v", set.Bundles)
	}
	if !containsString(set.Bundles, "project-scripts") {
		t.Fatalf("expected remaining explicit bundle to stay selected: %v", set.Bundles)
	}
}

func TestAssembleManifestFiltersToResolvedContract(t *testing.T) {
	base := manifest.Manifest{
		RuleTemplates:         []string{"rules-backend.yaml", "security-global.yaml", "security-shell.yaml"},
		ManagedTargets:        []string{"AGENTS.md", "rules/*.yaml", "skills/*", "skills/AVAILABLE_SKILLS.xml", "skills/AVAILABLE_SKILLS.json", "skills/SUMMARY.md"},
		PreservedTargets:      []string{"README.md", "specs/spec.yml", "SNAPSHOT.md", "SPEC.md"},
		RequiredTemplateFiles: []string{"AGENTS.md", "manifest.txt", "specs/spec.yml"},
		RequiredTemplateDirs:  []string{"rules", "skills", "specs"},
	}

	got := AssembleManifest(base, InstallSet{
		Rules: []string{"security-global.yaml", "security-shell.yaml"},
	})

	if len(got.RuleTemplates) != 2 {
		t.Fatalf("expected filtered rule templates, got %v", got.RuleTemplates)
	}
	if got.RequiredTemplateFiles[0] != "AGENTS.md" {
		t.Fatalf("expected AGENTS.md in required template files, got %v", got.RequiredTemplateFiles)
	}
	foundPromptDir := false
	for _, file := range got.RequiredTemplateFiles {
		if file == "prompts/ss-prompt.txt" {
			t.Fatalf("did not expect prompt helper template in assembled manifest: %v", got.RequiredTemplateFiles)
		}
	}
	for _, dir := range got.RequiredTemplateDirs {
		if dir == "prompts" {
			foundPromptDir = true
			break
		}
	}
	if foundPromptDir {
		t.Fatalf("did not expect prompt helper directory in assembled manifest: %v", got.RequiredTemplateDirs)
	}
}

func TestBuildSkillsActionPlanForceShowsDirectoryReplacementAndRootRemovals(t *testing.T) {
	workDir := t.TempDir()
	mustWriteResolveFile(t, filepath.Join(workDir, "skills", "analyze", "SKILL.md"), "custom\n")
	mustWriteResolveFile(t, filepath.Join(workDir, "skills", "notes.txt"), "remove me\n")
	mustWriteResolveFile(t, filepath.Join(workDir, "skills", "custom-skill", "SKILL.md"), "stale\n")

	plan := BuildSkillsActionPlan(workDir, InstallSet{
		Skills: []string{"analyze"},
	}, true)

	if !containsString(plan.Update, "skills/") {
		t.Fatalf("expected skills directory replacement in update plan, got %v", plan.Update)
	}
	if !containsString(plan.Update, "skills/analyze/") {
		t.Fatalf("expected selected skill directory replacement in update plan, got %v", plan.Update)
	}
	if !containsString(plan.Remove, "skills/notes.txt") {
		t.Fatalf("expected unmanaged skills-root file removal in plan, got %v", plan.Remove)
	}
	if !containsString(plan.Remove, "skills/custom-skill") {
		t.Fatalf("expected stale skill directory removal in plan, got %v", plan.Remove)
	}
}

func mustWriteResolveFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

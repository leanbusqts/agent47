package resolve

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/leanbusqts/agent47/internal/analyze"
	"github.com/leanbusqts/agent47/internal/manifest"
)

func TestResolveLowSignalFallsBackToBasePolicy(t *testing.T) {
	set, err := Resolve(analyze.AnalysisResult{LowSignal: true}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !equalStrings(set.Bundles, []string{"base"}) {
		t.Fatalf("expected base bundle only, got %v", set.Bundles)
	}
	if !containsString(set.Rules, "rules-cross.yaml") {
		t.Fatalf("expected base rules, got %v", set.Rules)
	}
	if !equalStrings(set.CreateFiles, []string{"AGENTS.md"}) {
		t.Fatalf("expected AGENTS.md as the only non-rule target, got %v", set.CreateFiles)
	}
}

func TestResolveAddsGoRulesForCLI(t *testing.T) {
	set, err := Resolve(analyze.AnalysisResult{
		ProjectTypes: []analyze.DetectedProjectType{{ID: "cli", Confidence: analyze.ConfidenceHigh}},
		Technologies: []analyze.DetectedTechnology{{ID: "go", Confidence: analyze.ConfidenceHigh}},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"rules-cli.yaml", "rules-go.yaml", "security-go.yaml", "shared-cli-behavior.yaml", "shared-testing.yaml"} {
		if !containsString(set.Rules, want) {
			t.Fatalf("expected %s, got %v", want, set.Rules)
		}
	}
}

func TestResolveSupportsKnownComposition(t *testing.T) {
	set, err := Resolve(analyze.AnalysisResult{
		ProjectTypes: []analyze.DetectedProjectType{
			{ID: "cli", Confidence: analyze.ConfidenceHigh},
			{ID: "scripts", Confidence: analyze.ConfidenceHigh},
		},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"base", "project-cli", "project-scripts", "shared-cli-behavior", "shared-testing"} {
		if !containsString(set.Bundles, want) {
			t.Fatalf("expected %s, got %v", want, set.Bundles)
		}
	}
}

func TestResolveConflictFallsBackToBase(t *testing.T) {
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
	if !equalStrings(set.Bundles, []string{"base"}) || !set.UnresolvedConflict {
		t.Fatalf("expected conflict-safe base fallback, got %+v", set)
	}
}

func TestResolveAutomaticBundlesAndExclusions(t *testing.T) {
	set, err := Resolve(analyze.AnalysisResult{
		ProjectTypes: []analyze.DetectedProjectType{
			{ID: "cli", Confidence: analyze.ConfidenceHigh},
			{ID: "scripts", Confidence: analyze.ConfidenceHigh},
		},
	}, Options{
		ExcludeBundles: []string{"cli"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if containsString(set.Bundles, "project-cli") || containsString(set.Bundles, "shared-cli-behavior") {
		t.Fatalf("excluded CLI bundle leaked into result: %v", set.Bundles)
	}
	if !containsString(set.Bundles, "project-scripts") {
		t.Fatalf("expected scripts bundle, got %v", set.Bundles)
	}
}

func TestResolveRejectsInvalidExplicitSelection(t *testing.T) {
	if _, err := Resolve(analyze.AnalysisResult{}, Options{ExplicitBundles: []string{"frontend", "backend"}}); err == nil {
		t.Fatal("expected incompatible explicit bundle error")
	}
	if _, err := Resolve(analyze.AnalysisResult{}, Options{ExcludeBundles: []string{"base"}}); err == nil {
		t.Fatal("expected base exclusion error")
	}
	if _, err := Resolve(analyze.AnalysisResult{}, Options{ExplicitBundles: []string{"unknown"}}); err == nil {
		t.Fatal("expected unknown bundle error")
	}
	if _, err := Resolve(analyze.AnalysisResult{}, Options{ExplicitBundles: []string{"monorepo"}, ExcludeBundles: []string{"monorepo-tooling"}}); err == nil {
		t.Fatal("expected included/excluded alias conflict")
	}
}

func TestBundleRulesMatchTemplateManifests(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	for id, bundle := range bundles {
		t.Run(id, func(t *testing.T) {
			manifestPath := filepath.Join(repoRoot, "templates", "bundles", id, "manifest.txt")
			parse := manifest.ParsePartial
			if id == "base" {
				manifestPath = filepath.Join(repoRoot, "templates", "base", "manifest.txt")
				parse = manifest.Parse
			}
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				t.Fatal(err)
			}
			got, err := parse(data)
			if err != nil {
				t.Fatal(err)
			}
			want := append([]string{}, bundle.IncludesRules...)
			sort.Strings(want)
			sort.Strings(got.RuleTemplates)
			if !equalStrings(got.RuleTemplates, want) {
				t.Fatalf("resolver rules drift from %s: got %v want %v", manifestPath, want, got.RuleTemplates)
			}
		})
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

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

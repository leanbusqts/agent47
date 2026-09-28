package resolve

import (
	"fmt"
	"sort"

	"github.com/leanbusqts/agent47/internal/analyze"
)

type InstallSet struct {
	BaseBundle           bool     `json:"base_bundle"`
	Bundles              []string `json:"bundles"`
	Rules                []string `json:"rules"`
	CreateFiles          []string `json:"create_files"`
	KeepFiles            []string `json:"keep_files"`
	DecisionNotes        []string `json:"decision_notes"`
	UnresolvedConflict   bool     `json:"unresolved_conflict"`
	ConflictProjectTypes []string `json:"conflict_project_types"`
}

type Bundle struct {
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`
	Description   string   `json:"description"`
	Requires      []string `json:"requires"`
	IncludesRules []string `json:"includes_rules"`
}

type Options struct {
	ExplicitBundles []string
	ExcludeBundles  []string
}

var bundles = map[string]Bundle{
	"base": {
		ID:            "base",
		Kind:          "base",
		Description:   "Conservative default scaffold for low-signal repositories.",
		IncludesRules: []string{"rules-cross.yaml", "security-global.yaml", "security-shell.yaml"},
	},
	"project-frontend": {
		ID:            "project-frontend",
		Kind:          "project",
		Description:   "Frontend-specific guidance.",
		Requires:      []string{"shared-testing"},
		IncludesRules: []string{"rules-frontend.yaml", "security-js-ts.yaml"},
	},
	"project-backend": {
		ID:            "project-backend",
		Kind:          "project",
		Description:   "Backend-specific guidance.",
		Requires:      []string{"shared-testing"},
		IncludesRules: []string{"rules-backend.yaml"},
	},
	"project-mobile": {
		ID:            "project-mobile",
		Kind:          "project",
		Description:   "Mobile-specific guidance.",
		Requires:      []string{"shared-testing"},
		IncludesRules: []string{"rules-mobile.yaml"},
	},
	"project-cli": {
		ID:            "project-cli",
		Kind:          "project",
		Description:   "CLI-oriented guidance.",
		Requires:      []string{"shared-cli-behavior", "shared-testing"},
		IncludesRules: []string{"rules-cli.yaml"},
	},
	"project-scripts": {
		ID:            "project-scripts",
		Kind:          "project",
		Description:   "Shell and automation workflow guidance.",
		Requires:      []string{"shared-testing"},
		IncludesRules: []string{"rules-scripts.yaml"},
	},
	"project-infra": {
		ID:            "project-infra",
		Kind:          "project",
		Description:   "Infrastructure and deployment guidance.",
		IncludesRules: []string{"rules-infra.yaml"},
	},
	"project-monorepo-tooling": {
		ID:            "project-monorepo-tooling",
		Kind:          "project",
		Description:   "Workspace and task orchestration guidance.",
		Requires:      []string{"shared-cli-behavior", "shared-testing"},
		IncludesRules: []string{"rules-monorepo-tooling.yaml"},
	},
	"project-desktop": {
		ID:            "project-desktop",
		Kind:          "project",
		Description:   "Desktop application guidance.",
		Requires:      []string{"shared-testing"},
		IncludesRules: []string{"rules-desktop.yaml"},
	},
	"project-plugin": {
		ID:            "project-plugin",
		Kind:          "project",
		Description:   "Plugin and extension guidance.",
		Requires:      []string{"shared-testing"},
		IncludesRules: []string{"rules-plugin.yaml"},
	},
	"shared-cli-behavior": {
		ID:            "shared-cli-behavior",
		Kind:          "shared",
		Description:   "Shared CLI behavior guidance.",
		IncludesRules: []string{"shared-cli-behavior.yaml"},
	},
	"shared-testing": {
		ID:            "shared-testing",
		Kind:          "shared",
		Description:   "Shared testing guidance.",
		IncludesRules: []string{"shared-testing.yaml"},
	},
}

var supportedProjectCompositions = map[string]bool{
	"cli+monorepo-tooling": true,
	"cli+scripts":          true,
	"desktop+scripts":      true,
	"desktop+plugin":       true,
}

var bundleAliases = map[string]string{
	"base":             "base",
	"frontend":         "project-frontend",
	"backend":          "project-backend",
	"mobile":           "project-mobile",
	"cli":              "project-cli",
	"scripts":          "project-scripts",
	"infra":            "project-infra",
	"monorepo":         "project-monorepo-tooling",
	"monorepo-tooling": "project-monorepo-tooling",
	"desktop":          "project-desktop",
	"plugin":           "project-plugin",
}

func Resolve(result analyze.AnalysisResult, opts Options) (InstallSet, error) {
	resolved, err := resolvedBundles(result, opts)
	if err != nil {
		return InstallSet{}, err
	}

	set := InstallSet{
		BaseBundle:           true,
		Bundles:              resolved,
		CreateFiles:          []string{"AGENTS.md"},
		KeepFiles:            []string{"README.md", "SNAPSHOT.md", "SPEC.md", ".agents", "skills", "prompts"},
		UnresolvedConflict:   result.UnresolvedConflict,
		ConflictProjectTypes: append([]string{}, result.ConflictProjectTypes...),
	}

	for _, bundleID := range resolved {
		bundle := bundles[bundleID]
		set.Rules = append(set.Rules, bundle.IncludesRules...)
	}

	addLanguageSecurityRules(&set, result)
	set.Rules = uniqSorted(set.Rules)

	if len(opts.ExplicitBundles) > 0 {
		set.DecisionNotes = append(set.DecisionNotes, "Explicit bundle selection overrides automatic resolution.")
	} else if result.UnresolvedConflict {
		set.DecisionNotes = append(set.DecisionNotes, "Multiple project types detected with no supported automatic composition; using the base bundle only.")
	} else if result.LowSignal {
		set.DecisionNotes = append(set.DecisionNotes, "No strong project signals found; using the base bundle only.")
	} else {
		set.DecisionNotes = append(set.DecisionNotes, "Resolved bundles from detected project types and technologies.")
	}

	return set, nil
}

func resolvedBundles(result analyze.AnalysisResult, opts Options) ([]string, error) {
	for _, excluded := range opts.ExcludeBundles {
		bundleID, ok := bundleAliases[excluded]
		if !ok {
			return nil, fmt.Errorf("unknown bundle: %s", excluded)
		}
		if bundleID == "base" {
			return nil, fmt.Errorf("cannot exclude the base bundle")
		}
	}

	if len(opts.ExplicitBundles) > 0 {
		selected := []string{"base"}
		for _, item := range opts.ExplicitBundles {
			bundleID, ok := bundleAliases[item]
			if !ok {
				return nil, fmt.Errorf("unknown bundle: %s", item)
			}
			selected = append(selected, bundleID)
		}
		for _, excluded := range opts.ExcludeBundles {
			bundleID, ok := bundleAliases[excluded]
			if !ok {
				return nil, fmt.Errorf("unknown bundle: %s", excluded)
			}
			if bundleID == "base" {
				return nil, fmt.Errorf("cannot exclude the base bundle")
			}
			if containsBundle(selected, bundleID) {
				return nil, fmt.Errorf("bundle cannot be both included and excluded: %s", excluded)
			}
			selected = remove(selected, bundleID)
		}
		selected = uniqSorted(selected)
		if err := validateBundleSelection(selected); err != nil {
			return nil, err
		}
		return expandBundleDependencies(selected)
	}

	if result.LowSignal || len(result.ProjectTypes) == 0 {
		return []string{"base"}, nil
	}
	if result.UnresolvedConflict {
		return []string{"base"}, nil
	}

	projectIDs := make([]string, 0, len(result.ProjectTypes))
	for _, projectType := range result.ProjectTypes {
		switch projectType.ID {
		case "frontend", "backend", "mobile", "cli", "scripts", "infra", "monorepo-tooling", "desktop", "plugin":
			projectIDs = append(projectIDs, projectType.ID)
		}
	}

	sort.Strings(projectIDs)
	if len(projectIDs) > 1 {
		key := compositionKey(projectIDs)
		if len(projectIDs) != 2 || !supportedProjectCompositions[key] {
			return []string{"base"}, nil
		}
	}

	selected := []string{"base"}
	for _, projectID := range projectIDs {
		selected = append(selected, bundleAliases[projectID])
	}

	for _, excluded := range opts.ExcludeBundles {
		bundleID, ok := bundleAliases[excluded]
		if !ok {
			return nil, fmt.Errorf("unknown bundle: %s", excluded)
		}
		if bundleID == "base" {
			return nil, fmt.Errorf("cannot exclude the base bundle")
		}
		selected = remove(selected, bundleID)
	}

	return expandBundleDependencies(selected)
}

func expandBundleDependencies(selected []string) ([]string, error) {
	expanded := make([]string, 0, len(selected))
	queue := append([]string{}, selected...)
	seen := map[string]bool{}

	for i := 0; i < len(queue); i++ {
		bundleID := queue[i]
		if bundleID == "" || bundleID == "base" || seen[bundleID] {
			continue
		}
		bundle, ok := bundles[bundleID]
		if !ok {
			return nil, fmt.Errorf("unknown bundle: %s", bundleID)
		}
		seen[bundleID] = true
		expanded = append(expanded, bundleID)
		for _, required := range bundle.Requires {
			if required == "" || required == "base" || seen[required] || containsBundle(queue, required) {
				continue
			}
			if _, ok := bundles[required]; !ok {
				return nil, fmt.Errorf("unknown bundle requirement: %s requires %s", bundleID, required)
			}
			queue = append(queue, required)
		}
	}

	return uniqSorted(append([]string{"base"}, expanded...)), nil
}

func containsBundle(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func validateBundleSelection(selected []string) error {
	projectBundles := make([]string, 0, len(selected))
	for _, bundleID := range selected {
		if bundleID == "base" {
			continue
		}
		projectBundles = append(projectBundles, bundleID)
	}
	if len(projectBundles) <= 1 {
		return nil
	}

	ids := make([]string, 0, len(projectBundles))
	for _, bundleID := range projectBundles {
		ids = append(ids, projectBundleID(bundleID))
	}
	sort.Strings(ids)
	key := compositionKey(ids)
	if len(ids) == 2 && supportedProjectCompositions[key] {
		return nil
	}

	return fmt.Errorf("explicit bundle selection is incompatible: %v", ids)
}

func projectBundleID(bundleID string) string {
	for key, value := range bundleAliases {
		if value == bundleID {
			return key
		}
	}
	return bundleID
}

func addLanguageSecurityRules(set *InstallSet, result analyze.AnalysisResult) {
	for _, technology := range result.Technologies {
		switch technology.ID {
		case "go":
			set.Rules = append(set.Rules, "rules-go.yaml", "security-go.yaml")
		case "typescript", "react", "node":
			set.Rules = append(set.Rules, "security-js-ts.yaml")
		case "python":
			set.Rules = append(set.Rules, "security-py.yaml")
		case "java-kotlin":
			set.Rules = append(set.Rules, "security-java-kotlin.yaml")
		case "swift":
			set.Rules = append(set.Rules, "security-swift.yaml")
		case "csharp":
			set.Rules = append(set.Rules, "security-csharp.yaml")
		}
	}
}

func compositionKey(ids []string) string {
	if len(ids) != 2 {
		return ""
	}
	return ids[0] + "+" + ids[1]
}

func uniqSorted(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func remove(values []string, target string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == target {
			continue
		}
		result = append(result, value)
	}
	return result
}

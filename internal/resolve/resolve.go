package resolve

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/leanbusqts/agent47/internal/analyze"
	"github.com/leanbusqts/agent47/internal/manifest"
)

type InstallSet struct {
	BaseBundle           bool     `json:"base_bundle"`
	Bundles              []string `json:"bundles"`
	Rules                []string `json:"rules"`
	Skills               []string `json:"skills"`
	Prompts              []string `json:"prompts"`
	CreateFiles          []string `json:"create_files"`
	KeepFiles            []string `json:"keep_files"`
	RemoveOnForce        []string `json:"remove_on_force"`
	DecisionNotes        []string `json:"decision_notes"`
	UnresolvedConflict   bool     `json:"unresolved_conflict"`
	ConflictProjectTypes []string `json:"conflict_project_types"`
}

type Bundle struct {
	ID             string   `json:"id"`
	Kind           string   `json:"kind"`
	Description    string   `json:"description"`
	Requires       []string `json:"requires"`
	IncludesRules  []string `json:"includes_rules"`
	IncludesSkills []string `json:"includes_skills"`
	IncludesFiles  []string `json:"includes_files"`
	ManagedTargets []string `json:"managed_targets"`
}

type Options struct {
	ExplicitBundles []string
	ExcludeBundles  []string
}

type ActionPlan struct {
	Create []string
	Update []string
	Keep   []string
	Remove []string
}

var universalSkills = []string{
	"analyze",
	"implement",
	"review",
	"troubleshoot",
	"plan",
	"spec-clarify",
}

var bundles = map[string]Bundle{
	"base": {
		ID:             "base",
		Kind:           "base",
		Description:    "Conservative default scaffold for low-signal repositories.",
		IncludesRules:  []string{"security-global.yaml", "security-shell.yaml"},
		IncludesSkills: universalSkills,
		IncludesFiles:  []string{"AGENTS.md", "specs/spec.yml", "README.md"},
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
		ID:             "project-cli",
		Kind:           "project",
		Description:    "CLI-oriented guidance.",
		Requires:       []string{"shared-cli-behavior", "shared-testing"},
		IncludesRules:  []string{"rules-cli.yaml"},
		IncludesSkills: []string{"cli-design"},
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
		BaseBundle: true,
		Bundles:    resolved,
		CreateFiles: []string{
			"AGENTS.md",
			"skills/AVAILABLE_SKILLS.xml",
			"skills/AVAILABLE_SKILLS.json",
			"skills/SUMMARY.md",
			"specs/spec.yml",
		},
		KeepFiles:            []string{"README.md", "SNAPSHOT.md", "SPEC.md"},
		UnresolvedConflict:   result.UnresolvedConflict,
		ConflictProjectTypes: append([]string{}, result.ConflictProjectTypes...),
	}

	for _, bundleID := range resolved {
		bundle := bundles[bundleID]
		set.Rules = append(set.Rules, bundle.IncludesRules...)
		set.Skills = append(set.Skills, bundle.IncludesSkills...)
		set.Prompts = append(set.Prompts, promptNames(bundle.IncludesFiles)...)
	}

	addLanguageSecurityRules(&set, result)
	addTechnologySkills(&set, result.Technologies)
	set.Rules = uniqSorted(set.Rules)
	set.Skills = uniqSorted(set.Skills)
	set.Prompts = uniqSorted(set.Prompts)
	set.RemoveOnForce = append(set.RemoveOnForce, removableTargets(set)...)

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

func addTechnologySkills(set *InstallSet, technologies []analyze.DetectedTechnology) {
	for _, technology := range technologies {
		switch technology.ID {
		case "go", "node", "typescript", "react", "java-kotlin", "swift", "csharp", "vitest", "jest", "go-test":
			set.Skills = append(set.Skills, "refactor")
		case "tailwind", "playwright", "cypress":
			set.Skills = append(set.Skills, "optimize")
		case "python", "shell", "bats", "infra", "workspace-tooling", "desktop-runtime", "plugin-hosting":
			set.Skills = append(set.Skills, "troubleshoot")
		}
	}
}

func compositionKey(ids []string) string {
	if len(ids) != 2 {
		return ""
	}
	return ids[0] + "+" + ids[1]
}

func promptNames(files []string) []string {
	var prompts []string
	for _, file := range files {
		if filepath.Dir(file) == "prompts" {
			prompts = append(prompts, filepath.Base(file))
		}
	}
	return prompts
}

func removableTargets(set InstallSet) []string {
	var targets []string
	for _, rule := range set.Rules {
		targets = append(targets, filepath.ToSlash(filepath.Join("rules", rule)))
	}
	for _, skill := range set.Skills {
		targets = append(targets, filepath.ToSlash(filepath.Join("skills", skill)))
	}
	return uniqSorted(targets)
}

func BuildActionPlan(workDir string, set InstallSet, force bool) ActionPlan {
	plan := ActionPlan{}

	targets := make([]string, 0, len(set.CreateFiles)+len(set.Rules)+len(set.Skills)+2)
	targets = append(targets, set.CreateFiles...)
	for _, prompt := range set.Prompts {
		targets = append(targets, filepath.ToSlash(filepath.Join("prompts", prompt)))
	}
	for _, rule := range set.Rules {
		targets = append(targets, filepath.ToSlash(filepath.Join("rules", rule)))
	}
	for _, skill := range set.Skills {
		targets = append(targets, filepath.ToSlash(filepath.Join("skills", skill, "SKILL.md")))
	}

	for _, rel := range uniqSorted(targets) {
		path := filepath.Join(workDir, filepath.FromSlash(rel))
		if exists(path) {
			if force {
				plan.Update = append(plan.Update, rel)
			} else {
				plan.Keep = append(plan.Keep, rel)
			}
			continue
		}
		plan.Create = append(plan.Create, rel)
	}

	readmePath := filepath.Join(workDir, "README.md")
	if exists(readmePath) {
		plan.Keep = append(plan.Keep, "README.md")
	} else {
		plan.Create = append(plan.Create, "README.md")
	}

	if force {
		plan.Remove = append(plan.Remove, staleForceTargets(workDir, set)...)
	}

	plan.Create = uniqSorted(plan.Create)
	plan.Update = uniqSorted(plan.Update)
	plan.Keep = uniqSorted(plan.Keep)
	plan.Remove = uniqSorted(plan.Remove)
	return plan
}

func BuildSkillsActionPlan(workDir string, set InstallSet, force bool) ActionPlan {
	plan := ActionPlan{}
	skillsDirPath := filepath.Join(workDir, "skills")

	targets := []string{
		"skills/AVAILABLE_SKILLS.xml",
		"skills/AVAILABLE_SKILLS.json",
		"skills/SUMMARY.md",
	}
	for _, skill := range set.Skills {
		if force {
			skillDirRel := filepath.ToSlash(filepath.Join("skills", skill)) + "/"
			skillDirPath := filepath.Join(workDir, "skills", skill)
			if exists(skillDirPath) {
				plan.Update = append(plan.Update, skillDirRel)
			} else {
				targets = append(targets, filepath.ToSlash(filepath.Join("skills", skill, "SKILL.md")))
			}
			continue
		}
		targets = append(targets, filepath.ToSlash(filepath.Join("skills", skill, "SKILL.md")))
	}

	for _, rel := range uniqSorted(targets) {
		path := filepath.Join(workDir, filepath.FromSlash(rel))
		if exists(path) {
			if force {
				plan.Update = append(plan.Update, rel)
			} else {
				plan.Keep = append(plan.Keep, rel)
			}
			continue
		}
		plan.Create = append(plan.Create, rel)
	}

	if force {
		if exists(skillsDirPath) {
			plan.Update = append(plan.Update, "skills/")
		}
		plan.Remove = append(plan.Remove, staleSkillsForceTargets(workDir, set)...)
	}

	plan.Create = uniqSorted(plan.Create)
	plan.Update = uniqSorted(plan.Update)
	plan.Keep = uniqSorted(plan.Keep)
	plan.Remove = uniqSorted(plan.Remove)
	return plan
}

func AssembleManifest(base manifest.Manifest, set InstallSet) manifest.Manifest {
	assembled := manifest.Manifest{
		ManagedTargets:   append([]string{}, base.ManagedTargets...),
		PreservedTargets: append([]string{}, base.PreservedTargets...),
	}

	if len(set.Rules) > 0 {
		assembled.RuleTemplates = append([]string{}, set.Rules...)
	} else {
		assembled.RuleTemplates = append([]string{}, base.RuleTemplates...)
	}

	requiredFiles := []string{"AGENTS.md", "specs/spec.yml", "manifest.txt"}
	for _, prompt := range set.Prompts {
		requiredFiles = append(requiredFiles, filepath.ToSlash(filepath.Join("prompts", prompt)))
	}
	assembled.RequiredTemplateFiles = uniqSorted(requiredFiles)

	requiredDirs := []string{"skills", "specs"}
	if len(assembled.RuleTemplates) > 0 {
		requiredDirs = append(requiredDirs, "rules")
	}
	if len(set.Prompts) > 0 {
		requiredDirs = append(requiredDirs, "prompts")
	}
	assembled.RequiredTemplateDirs = uniqSorted(requiredDirs)

	return assembled
}

func staleForceTargets(workDir string, set InstallSet) []string {
	var stale []string

	selectedRules := make(map[string]bool, len(set.Rules))
	for _, rule := range set.Rules {
		selectedRules[rule] = true
	}
	ruleFiles, _ := filepath.Glob(filepath.Join(workDir, "rules", "*.yaml"))
	for _, path := range ruleFiles {
		name := filepath.Base(path)
		if !selectedRules[name] {
			stale = append(stale, filepath.ToSlash(filepath.Join("rules", name)))
		}
	}

	selectedSkills := make(map[string]bool, len(set.Skills))
	for _, skill := range set.Skills {
		selectedSkills[skill] = true
	}
	entries, err := osReadDir(filepath.Join(workDir, "skills"))
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			if !selectedSkills[entry.Name()] {
				stale = append(stale, filepath.ToSlash(filepath.Join("skills", entry.Name())))
			}
		}
	}

	return stale
}

func staleSkillsForceTargets(workDir string, set InstallSet) []string {
	var stale []string

	selectedSkills := make(map[string]bool, len(set.Skills))
	for _, skill := range set.Skills {
		selectedSkills[skill] = true
	}
	managedIndexFiles := map[string]bool{
		"AVAILABLE_SKILLS.xml":  true,
		"AVAILABLE_SKILLS.json": true,
		"SUMMARY.md":            true,
	}
	entries, err := osReadDir(filepath.Join(workDir, "skills"))
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				if !selectedSkills[entry.Name()] {
					stale = append(stale, filepath.ToSlash(filepath.Join("skills", entry.Name())))
				}
				continue
			}
			if !managedIndexFiles[entry.Name()] {
				stale = append(stale, filepath.ToSlash(filepath.Join("skills", entry.Name())))
			}
		}
	}

	return uniqSorted(stale)
}

var (
	exists    = defaultExists
	osReadDir = defaultReadDir
)

func defaultExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func defaultReadDir(path string) ([]osDirEntry, error) {
	return readDir(path)
}

type osDirEntry interface {
	IsDir() bool
	Name() string
}

func readDir(path string) ([]osDirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	result := make([]osDirEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry)
	}
	return result, nil
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

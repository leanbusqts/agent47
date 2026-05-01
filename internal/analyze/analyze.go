package analyze

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Confidence string

const (
	ConfidenceLow    Confidence = "low"
	ConfidenceMedium Confidence = "medium"
	ConfidenceHigh   Confidence = "high"
)

type DetectedProjectType struct {
	ID         string     `json:"id"`
	Confidence Confidence `json:"confidence"`
	Evidence   []string   `json:"evidence"`
}

type DetectedTechnology struct {
	ID         string     `json:"id"`
	Confidence Confidence `json:"confidence"`
	Evidence   []string   `json:"evidence"`
}

type EvidenceItem struct {
	ID          string   `json:"id"`
	Kind        string   `json:"kind"`
	Detail      string   `json:"detail"`
	SourcePaths []string `json:"source_paths"`
}

type ManagedState struct {
	LegacyScaffold bool     `json:"legacy_scaffold"`
	Notes          []string `json:"notes"`
}

type AnalysisResult struct {
	ProjectTypes         []DetectedProjectType `json:"project_types"`
	Technologies         []DetectedTechnology  `json:"technologies"`
	RepoShape            string                `json:"repo_shape"`
	Confidence           Confidence            `json:"confidence"`
	LowSignal            bool                  `json:"low_signal"`
	UnresolvedConflict   bool                  `json:"unresolved_conflict"`
	ConflictProjectTypes []string              `json:"conflict_project_types"`
	Evidence             []EvidenceItem        `json:"evidence"`
	ManagedState         ManagedState          `json:"managed_state"`
	Warnings             []string              `json:"warnings"`
}

type Service struct{}

func (Service) Analyze(root string) (AnalysisResult, error) {
	signals, err := scan(root)
	if err != nil {
		return AnalysisResult{}, err
	}

	result := AnalysisResult{
		RepoShape:    detectRepoShape(signals),
		ProjectTypes: detectProjectTypes(signals),
		Technologies: detectTechnologies(signals),
		ManagedState: detectManagedState(signals),
	}
	result.Evidence = sortEvidence(append(append([]EvidenceItem{}, signals.evidence...), classificationEvidence(result.ProjectTypes, result.Technologies)...))

	result.Confidence = overallConfidence(result.ProjectTypes, result.Technologies, signals)
	result.LowSignal = len(result.ProjectTypes) == 0
	result.UnresolvedConflict, result.ConflictProjectTypes = detectUnresolvedConflict(result.ProjectTypes)
	if result.LowSignal {
		result.Warnings = append(result.Warnings, "No strong project signals found.")
	}
	if result.UnresolvedConflict {
		result.Warnings = append(result.Warnings, "Multiple project types detected with no supported automatic composition.")
	}

	return result, nil
}

type repoSignals struct {
	entries         []string
	files           map[string]bool
	dirs            map[string]bool
	extCounts       map[string]int
	coreFiles       map[string]bool
	coreDirs        map[string]bool
	coreExtCounts   map[string]int
	packageJSON     string
	goMod           string
	evidence        []EvidenceItem
	hasAgents       bool
	hasRules        bool
	hasSkills       bool
	hasPrompts      bool
	hasPackageJSON  bool
	hasGoMod        bool
	hasGradle       bool
	hasSwiftPackage bool
}

func scan(root string) (repoSignals, error) {
	signals := repoSignals{
		files:         make(map[string]bool),
		dirs:          make(map[string]bool),
		extCounts:     make(map[string]int),
		coreFiles:     make(map[string]bool),
		coreDirs:      make(map[string]bool),
		coreExtCounts: make(map[string]int),
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return signals, err
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") && name != ".github" && name != ".codex-plugin" {
			continue
		}
		signals.entries = append(signals.entries, name)
		if entry.IsDir() {
			signals.dirs[name] = true
		} else {
			signals.files[name] = true
		}
	}

	sort.Strings(signals.entries)
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		base := filepath.Base(path)
		if d.IsDir() {
			if strings.HasPrefix(base, ".") && base != ".github" && base != ".codex-plugin" {
				return filepath.SkipDir
			}
			signals.dirs[rel] = true
			if isCorePath(rel) {
				signals.coreDirs[rel] = true
			}
			return nil
		}

		signals.files[rel] = true
		ext := strings.ToLower(filepath.Ext(rel))
		if ext != "" {
			signals.extCounts[ext]++
			if isCorePath(rel) {
				signals.coreExtCounts[ext]++
			}
		}
		if isCorePath(rel) {
			signals.coreFiles[rel] = true
		}

		switch rel {
		case "package.json":
			signals.hasPackageJSON = true
			body, err := os.ReadFile(path)
			if err == nil {
				signals.packageJSON = string(body)
				signals.evidence = append(signals.evidence, evidence("technology", "package-json", "Detected package.json", rel))
			}
		case "go.mod":
			signals.hasGoMod = true
			body, err := os.ReadFile(path)
			if err == nil {
				signals.goMod = string(body)
				signals.evidence = append(signals.evidence, evidence("technology", "go-mod", "Detected go.mod", rel))
			}
		case "Package.swift":
			signals.hasSwiftPackage = true
			signals.evidence = append(signals.evidence, evidence("technology", "swift-package", "Detected Package.swift", rel))
		case "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts":
			signals.hasGradle = true
			signals.evidence = append(signals.evidence, evidence("technology", "gradle", "Detected Gradle build file", rel))
		case "AGENTS.md":
			signals.hasAgents = true
		}

		return nil
	}); err != nil {
		return signals, err
	}

	signals.hasRules = signals.dirs["rules"]
	signals.hasSkills = signals.dirs["skills"]
	signals.hasPrompts = signals.dirs["prompts"]
	return signals, nil
}

func detectRepoShape(signals repoSignals) string {
	switch {
	case len(signals.entries) == 0:
		return "empty"
	case signals.files["pnpm-workspace.yaml"] || signals.files["turbo.json"] || signals.dirs["apps"] || signals.dirs["packages"]:
		return "monorepo"
	case signals.files["templates/manifest.txt"] || signals.dirs["templates"]:
		return "template"
	case mostlyDocs(signals):
		return "docs-heavy"
	default:
		return "single-package"
	}
}

func mostlyDocs(signals repoSignals) bool {
	docCount := signals.coreExtCounts[".md"] + signals.coreExtCounts[".mdx"]
	codeCount := 0
	for ext, count := range signals.coreExtCounts {
		switch ext {
		case ".go", ".js", ".ts", ".tsx", ".jsx", ".py", ".sh", ".bash", ".zsh", ".swift", ".kt", ".java", ".cs":
			codeCount += count
		}
	}
	return docCount > 0 && codeCount == 0
}

func detectTechnologies(signals repoSignals) []DetectedTechnology {
	var detected []DetectedTechnology

	if signals.hasGoMod {
		detected = append(detected, DetectedTechnology{ID: "go", Confidence: ConfidenceHigh, Evidence: []string{"go.mod"}})
	}
	if signals.hasPackageJSON {
		detected = append(detected, DetectedTechnology{ID: "node", Confidence: ConfidenceHigh, Evidence: []string{"package.json"}})
	}
	if strings.Contains(signals.packageJSON, `"typescript"`) || signals.coreExtCounts[".ts"] > 0 || signals.coreExtCounts[".tsx"] > 0 {
		detected = append(detected, DetectedTechnology{ID: "typescript", Confidence: confidenceFromCount(signals.coreExtCounts[".ts"]+signals.coreExtCounts[".tsx"], "package.json"), Evidence: collectEvidence(signals, "package.json", ".ts/.tsx files")})
	}
	if strings.Contains(signals.packageJSON, `"react"`) || signals.coreExtCounts[".tsx"] > 0 {
		detected = append(detected, DetectedTechnology{ID: "react", Confidence: confidenceFromCount(signals.coreExtCounts[".tsx"], "package.json"), Evidence: collectEvidence(signals, "package.json", ".tsx files")})
	}
	if strings.Contains(signals.packageJSON, `"tailwindcss"`) || hasPrefixFile(signals.files, "tailwind.config.") {
		detected = append(detected, DetectedTechnology{ID: "tailwind", Confidence: ConfidenceMedium, Evidence: collectEvidence(signals, "tailwind.config.*", "package.json")})
	}
	if signals.hasGradle || signals.coreExtCounts[".kt"] > 0 || signals.coreExtCounts[".java"] > 0 {
		detected = append(detected, DetectedTechnology{ID: "java-kotlin", Confidence: confidenceFromCount(signals.coreExtCounts[".kt"]+signals.coreExtCounts[".java"], "Gradle"), Evidence: collectEvidence(signals, "build.gradle", ".kt/.java files")})
	}
	if signals.hasSwiftPackage || signals.coreExtCounts[".swift"] > 0 {
		detected = append(detected, DetectedTechnology{ID: "swift", Confidence: confidenceFromCount(signals.coreExtCounts[".swift"], "Package.swift"), Evidence: collectEvidence(signals, "Package.swift", ".swift files")})
	}
	if signals.coreExtCounts[".py"] > 0 || signals.files["pyproject.toml"] || signals.files["requirements.txt"] {
		detected = append(detected, DetectedTechnology{ID: "python", Confidence: confidenceFromCount(signals.coreExtCounts[".py"], "pyproject.toml"), Evidence: collectEvidence(signals, "pyproject.toml", ".py files")})
	}
	if signals.coreExtCounts[".cs"] > 0 || hasSuffixFile(signals.files, ".csproj") || hasSuffixFile(signals.files, ".sln") {
		detected = append(detected, DetectedTechnology{ID: "csharp", Confidence: confidenceFromCount(signals.coreExtCounts[".cs"], ".csproj"), Evidence: collectEvidence(signals, ".csproj", ".cs files")})
	}
	if signals.coreExtCounts[".sh"] > 0 || signals.coreExtCounts[".bash"] > 0 || signals.coreExtCounts[".bats"] > 0 || signals.files["install.sh"] || signals.coreDirs["scripts"] {
		detected = append(detected, DetectedTechnology{ID: "shell", Confidence: confidenceFromCount(signals.coreExtCounts[".sh"]+signals.coreExtCounts[".bash"]+signals.coreExtCounts[".bats"], "scripts/"), Evidence: collectEvidence(signals, "install.sh", "shell files")})
	}
	if countInfraSignals(signals) > 0 {
		detected = append(detected, DetectedTechnology{ID: "infra", Confidence: confidenceFromSignals(countInfraSignals(signals)), Evidence: collectEvidence(signals, ".tf files", "helmfile.yaml", "charts/", "terraform/")})
	}
	if countMonorepoSignals(signals) > 0 {
		detected = append(detected, DetectedTechnology{ID: "workspace-tooling", Confidence: confidenceFromSignals(countMonorepoSignals(signals)), Evidence: collectEvidence(signals, "pnpm-workspace.yaml", "turbo.json", "nx.json", "lerna.json")})
	}
	if countDesktopSignals(signals) > 0 {
		detected = append(detected, DetectedTechnology{ID: "desktop-runtime", Confidence: confidenceFromSignals(countDesktopSignals(signals)), Evidence: collectEvidence(signals, "package.json", "src-tauri/", "wails.json")})
	}
	if countPluginSignals(signals) > 0 {
		detected = append(detected, DetectedTechnology{ID: "plugin-hosting", Confidence: confidenceFromSignals(countPluginSignals(signals)), Evidence: collectEvidence(signals, "plugin.json", ".codex-plugin/plugin.json", "plugins/", "plugin/")})
	}
	detected = append(detected, detectTestingTechnologies(signals)...)

	sort.Slice(detected, func(i, j int) bool {
		return detected[i].ID < detected[j].ID
	})
	return detected
}

func detectTestingTechnologies(signals repoSignals) []DetectedTechnology {
	var detected []DetectedTechnology

	if containsAny(signals.packageJSON, `"vitest"`, `"@vitest/ui"`) || hasPrefixFile(signals.files, "vitest.config.") || hasPrefixFile(signals.files, "vitest.workspace.") {
		signalsCount := 0
		if containsAny(signals.packageJSON, `"vitest"`, `"@vitest/ui"`) {
			signalsCount++
		}
		if hasPrefixFile(signals.files, "vitest.config.") || hasPrefixFile(signals.files, "vitest.workspace.") {
			signalsCount++
		}
		detected = append(detected, DetectedTechnology{ID: "vitest", Confidence: confidenceFromSignals(signalsCount), Evidence: collectEvidence(signals, "package.json", "vitest.config.*", "vitest.workspace.*")})
	}
	if containsAny(signals.packageJSON, `"jest"`, `"@jest/"`) || hasPrefixFile(signals.files, "jest.config.") || hasPrefixFile(signals.files, "jest.setup.") {
		signalsCount := 0
		if containsAny(signals.packageJSON, `"jest"`, `"@jest/"`) {
			signalsCount++
		}
		if hasPrefixFile(signals.files, "jest.config.") || hasPrefixFile(signals.files, "jest.setup.") {
			signalsCount++
		}
		detected = append(detected, DetectedTechnology{ID: "jest", Confidence: confidenceFromSignals(signalsCount), Evidence: collectEvidence(signals, "package.json", "jest.config.*", "jest.setup.*")})
	}
	if containsAny(signals.packageJSON, `"playwright"`, `"@playwright/test"`) || hasPrefixFile(signals.files, "playwright.config.") || signals.dirs["playwright"] {
		signalsCount := 0
		if containsAny(signals.packageJSON, `"playwright"`, `"@playwright/test"`) {
			signalsCount++
		}
		if hasPrefixFile(signals.files, "playwright.config.") || signals.dirs["playwright"] {
			signalsCount++
		}
		detected = append(detected, DetectedTechnology{ID: "playwright", Confidence: confidenceFromSignals(signalsCount), Evidence: collectEvidence(signals, "package.json", "playwright.config.*", "playwright/")})
	}
	if containsAny(signals.packageJSON, `"cypress"`) || hasPrefixFile(signals.files, "cypress.config.") || signals.dirs["cypress"] {
		signalsCount := 0
		if containsAny(signals.packageJSON, `"cypress"`) {
			signalsCount++
		}
		if hasPrefixFile(signals.files, "cypress.config.") || signals.dirs["cypress"] {
			signalsCount++
		}
		detected = append(detected, DetectedTechnology{ID: "cypress", Confidence: confidenceFromSignals(signalsCount), Evidence: collectEvidence(signals, "package.json", "cypress.config.*", "cypress/")})
	}
	if signals.hasGoMod && countFilesWithSuffix(signals.files, "_test.go") > 0 {
		signalsCount := 1
		if countFilesWithSuffix(signals.files, "_test.go") > 0 {
			signalsCount++
		}
		detected = append(detected, DetectedTechnology{ID: "go-test", Confidence: confidenceFromSignals(signalsCount), Evidence: collectEvidence(signals, "go.mod", "_test.go files")})
	}
	if countFilesWithSuffix(signals.files, ".bats") > 0 || (signals.dirs["tests"] && countFilesWithSuffix(signals.files, ".bats") > 0) {
		signalsCount := 0
		if countFilesWithSuffix(signals.files, ".bats") > 0 {
			signalsCount++
		}
		if signals.dirs["tests"] {
			signalsCount++
		}
		detected = append(detected, DetectedTechnology{ID: "bats", Confidence: confidenceFromSignals(signalsCount), Evidence: collectEvidence(signals, "tests/", ".bats files")})
	}

	return detected
}

func detectProjectTypes(signals repoSignals) []DetectedProjectType {
	var detected []DetectedProjectType

	if signals.coreDirs["cmd"] || strings.Contains(signals.goMod, "cobra") || strings.Contains(signals.packageJSON, `"bin"`) || signals.files["install.sh"] || signals.files["install.ps1"] {
		detected = append(detected, DetectedProjectType{ID: "cli", Confidence: ConfidenceHigh, Evidence: collectEvidence(signals, "cmd/", "install.sh", "install.ps1")})
	}
	if countMonorepoSignals(signals) > 0 {
		detected = append(detected, DetectedProjectType{ID: "monorepo-tooling", Confidence: confidenceFromSignals(countMonorepoSignals(signals)), Evidence: collectEvidence(signals, "pnpm-workspace.yaml", "turbo.json", "nx.json", "lerna.json", "apps/", "packages/")})
	}
	if signals.dirs["android"] || signals.dirs["ios"] || signals.hasGradle || signals.hasSwiftPackage {
		detected = append(detected, DetectedProjectType{ID: "mobile", Confidence: ConfidenceHigh, Evidence: collectEvidence(signals, "android/", "ios/", "Gradle", "Package.swift")})
	}
	if hasPrefixDir(signals.coreDirs, "src") || hasPrefixDir(signals.coreDirs, "app") || hasPrefixDir(signals.coreDirs, "pages") || strings.Contains(signals.packageJSON, `"next"`) || strings.Contains(signals.packageJSON, `"astro"`) || strings.Contains(signals.packageJSON, `"react"`) {
		detected = append(detected, DetectedProjectType{ID: "frontend", Confidence: confidenceFromCount(signals.coreExtCounts[".tsx"]+signals.coreExtCounts[".jsx"], "package.json"), Evidence: collectEvidence(signals, "src/", "app/", "pages/", "package.json")})
	}
	if signals.coreDirs["api"] || signals.coreDirs["server"] || signals.coreDirs["handlers"] || strings.Contains(signals.goMod, "gin-gonic") || strings.Contains(signals.goMod, "labstack/echo") || strings.Contains(signals.packageJSON, `"express"`) || strings.Contains(signals.packageJSON, `"fastify"`) {
		detected = append(detected, DetectedProjectType{ID: "backend", Confidence: ConfidenceHigh, Evidence: collectEvidence(signals, "api/", "server/", "handlers/", "framework dependency")})
	}
	if signals.coreDirs["scripts"] || signals.files["install.sh"] || signals.coreExtCounts[".sh"]+signals.coreExtCounts[".bash"]+signals.coreExtCounts[".bats"] >= 2 {
		detected = append(detected, DetectedProjectType{ID: "scripts", Confidence: confidenceFromCount(signals.coreExtCounts[".sh"]+signals.coreExtCounts[".bash"]+signals.coreExtCounts[".bats"], "scripts/"), Evidence: collectEvidence(signals, "scripts/", "shell files")})
	}
	if countInfraSignals(signals) > 0 {
		detected = append(detected, DetectedProjectType{ID: "infra", Confidence: confidenceFromSignals(countInfraSignals(signals)), Evidence: collectEvidence(signals, ".tf files", "helmfile.yaml", "charts/", "terraform/", "infra/")})
	}
	if countDesktopSignals(signals) > 0 {
		detected = append(detected, DetectedProjectType{ID: "desktop", Confidence: confidenceFromSignals(countDesktopSignals(signals)), Evidence: collectEvidence(signals, "package.json", "src-tauri/", "wails.json")})
	}
	if countPluginSignals(signals) > 0 {
		detected = append(detected, DetectedProjectType{ID: "plugin", Confidence: confidenceFromSignals(countPluginSignals(signals)), Evidence: collectEvidence(signals, "plugin.json", ".codex-plugin/plugin.json", "plugins/", "plugin/")})
	}

	sort.Slice(detected, func(i, j int) bool {
		return detected[i].ID < detected[j].ID
	})
	return detected
}

func detectManagedState(signals repoSignals) ManagedState {
	state := ManagedState{
		LegacyScaffold: signals.hasAgents || signals.hasRules || signals.hasSkills,
	}
	if state.LegacyScaffold {
		state.Notes = append(state.Notes, "Existing managed scaffold signals detected.")
	}
	if signals.hasPrompts {
		state.Notes = append(state.Notes, "Prompts directory already exists.")
	}
	return state
}

func detectUnresolvedConflict(projectTypes []DetectedProjectType) (bool, []string) {
	if len(projectTypes) <= 1 {
		return false, nil
	}

	ids := make([]string, 0, len(projectTypes))
	for _, projectType := range projectTypes {
		ids = append(ids, projectType.ID)
	}
	sort.Strings(ids)
	if len(ids) == 2 && supportedAutomaticComposition(ids[0], ids[1]) {
		return false, nil
	}

	return true, ids
}

func supportedAutomaticComposition(left string, right string) bool {
	switch left + "+" + right {
	case "cli+scripts", "cli+monorepo-tooling", "desktop+plugin":
		return true
	default:
		return false
	}
}

func countInfraSignals(signals repoSignals) int {
	count := 0
	if countFilesWithSuffix(signals.coreFiles, ".tf") > 0 {
		count++
	}
	if signals.coreFiles["helmfile.yaml"] || signals.coreFiles["helmfile.yml"] {
		count++
	}
	if signals.coreDirs["terraform"] || signals.coreDirs["infra"] || signals.coreDirs["charts"] || signals.coreDirs["helm"] || signals.coreDirs["k8s"] || signals.coreDirs["kubernetes"] {
		count++
	}
	return count
}

func countMonorepoSignals(signals repoSignals) int {
	count := 0
	if signals.coreFiles["pnpm-workspace.yaml"] || signals.coreFiles["turbo.json"] || signals.coreFiles["nx.json"] || signals.coreFiles["lerna.json"] || signals.coreFiles["lage.config.js"] || signals.coreFiles["lage.config.ts"] {
		count++
	}
	if signals.coreDirs["apps"] || signals.coreDirs["packages"] {
		count++
	}
	return count
}

func countDesktopSignals(signals repoSignals) int {
	count := 0
	if containsAny(signals.packageJSON, `"electron"`, `"tauri"`, `"wails"`) {
		count++
	}
	if signals.coreDirs["src-tauri"] || signals.coreFiles["wails.json"] {
		count++
	}
	if hasPrefixFile(signals.coreFiles, "electron-builder.") || hasPrefixFile(signals.coreFiles, "tauri.conf.") {
		count++
	}
	return count
}

func countPluginSignals(signals repoSignals) int {
	count := 0
	if signals.coreFiles["plugin.json"] || signals.files[".codex-plugin/plugin.json"] {
		count++
	}
	if signals.coreDirs["plugins"] || signals.coreDirs["plugin"] {
		count++
	}
	return count
}

func isCorePath(rel string) bool {
	return !hasAuxiliarySegment(rel)
}

func hasAuxiliarySegment(rel string) bool {
	for _, segment := range strings.Split(rel, "/") {
		switch segment {
		case "templates", "vendor", "node_modules", "test", "tests", "__tests__", "testdata", "fixtures":
			return true
		}
	}
	return false
}

func overallConfidence(projectTypes []DetectedProjectType, technologies []DetectedTechnology, signals repoSignals) Confidence {
	if len(projectTypes) == 0 && len(technologies) == 0 {
		if len(signals.entries) == 0 {
			return ConfidenceLow
		}
		return ConfidenceMedium
	}
	for _, projectType := range projectTypes {
		if projectType.Confidence == ConfidenceHigh {
			return ConfidenceHigh
		}
	}
	return ConfidenceMedium
}

func confidenceFromCount(count int, extraEvidence string) Confidence {
	switch {
	case count >= 3:
		return ConfidenceHigh
	case count >= 1 || extraEvidence != "":
		return ConfidenceMedium
	default:
		return ConfidenceLow
	}
}

func confidenceFromSignals(count int) Confidence {
	switch {
	case count >= 2:
		return ConfidenceHigh
	case count == 1:
		return ConfidenceMedium
	default:
		return ConfidenceLow
	}
}

func collectEvidence(signals repoSignals, values ...string) []string {
	var evidence []string
	for _, value := range values {
		if value == "" {
			continue
		}
		evidence = append(evidence, value)
	}
	return evidence
}

func evidence(kind string, id string, detail string, paths ...string) EvidenceItem {
	return EvidenceItem{ID: id, Kind: kind, Detail: detail, SourcePaths: paths}
}

func classificationEvidence(projectTypes []DetectedProjectType, technologies []DetectedTechnology) []EvidenceItem {
	items := make([]EvidenceItem, 0, len(projectTypes)+len(technologies))

	for _, projectType := range projectTypes {
		items = append(items, EvidenceItem{
			ID:          "project-type-" + projectType.ID,
			Kind:        "project-type",
			Detail:      fmt.Sprintf("Resolved project type %s (%s)", projectType.ID, projectType.Confidence),
			SourcePaths: append([]string{}, projectType.Evidence...),
		})
	}
	for _, technology := range technologies {
		items = append(items, EvidenceItem{
			ID:          "technology-" + technology.ID,
			Kind:        "technology",
			Detail:      fmt.Sprintf("Resolved technology %s (%s)", technology.ID, technology.Confidence),
			SourcePaths: append([]string{}, technology.Evidence...),
		})
	}

	return items
}

func sortEvidence(items []EvidenceItem) []EvidenceItem {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Kind == items[j].Kind {
			return items[i].ID < items[j].ID
		}
		return items[i].Kind < items[j].Kind
	})
	return items
}

func hasPrefixFile(files map[string]bool, prefix string) bool {
	for file := range files {
		if strings.HasPrefix(filepath.Base(file), prefix) {
			return true
		}
	}
	return false
}

func countFilesWithSuffix(files map[string]bool, suffix string) int {
	count := 0
	for file := range files {
		if strings.HasSuffix(file, suffix) {
			count++
		}
	}
	return count
}

func containsAny(body string, needles ...string) bool {
	for _, needle := range needles {
		if needle != "" && strings.Contains(body, needle) {
			return true
		}
	}
	return false
}

func hasSuffixFile(files map[string]bool, suffix string) bool {
	for file := range files {
		if strings.HasSuffix(file, suffix) {
			return true
		}
	}
	return false
}

func hasPrefixDir(dirs map[string]bool, prefix string) bool {
	for dir := range dirs {
		if dir == prefix || strings.HasPrefix(dir, prefix+"/") {
			return true
		}
	}
	return false
}

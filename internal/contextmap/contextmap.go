package contextmap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/leanbusqts/agent47/internal/analyze"
)

const (
	TargetPath       = ".agent47/context.md"
	SchemaVersion    = 1
	rendererVersion  = 1
	MaxDocumentBytes = 12 * 1024
	maxEntries       = 5000
	maxDepth         = 8
	maxFilesRead     = 1200
	maxFileBytes     = 256 * 1024
	maxTotalBytes    = 8 * 1024 * 1024
)

type Metadata struct {
	SchemaVersion     int    `json:"schema_version"`
	SourceFingerprint string `json:"source_fingerprint"`
	BodySHA256        string `json:"body_sha256"`
}

type Document struct {
	Metadata Metadata
	Body     []byte
	Content  []byte
}

type BuildOptions struct {
	PlannedPolicies []string
	ExcludedPaths   []string
}

type Action string

const (
	ActionCreate  Action = "create"
	ActionUpdate  Action = "update"
	ActionCurrent Action = "current"
)

var (
	ErrModifiedContext = errors.New("generated context was modified; rerun with --force to replace .agent47/context.md")
	ErrUnknownContext  = errors.New("existing .agent47/context.md is not a recognized generated context; rerun with --force to replace it")
)

type component struct {
	Path      string   `json:"path"`
	Languages []string `json:"languages"`
	Sources   int      `json:"sources"`
	Tests     int      `json:"tests"`
}

type evidenceItem struct {
	Path       string `json:"path"`
	Kind       string `json:"kind"`
	Confidence string `json:"confidence,omitempty"`
	Evidence   string `json:"evidence,omitempty"`
}

type workspaceRoot struct {
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	Evidence string `json:"evidence"`
}

type relationship struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Kind       string `json:"kind"`
	Confidence string `json:"confidence"`
	Evidence   string `json:"evidence"`
}

type testLink struct {
	Component string   `json:"component"`
	Count     int      `json:"count"`
	Examples  []string `json:"examples"`
}

type command struct {
	Invocation string `json:"invocation"`
	Evidence   string `json:"evidence"`
}

type graphModel struct {
	SchemaVersion   int             `json:"schema_version"`
	RendererVersion int             `json:"renderer_version"`
	RepoShape       string          `json:"repo_shape"`
	ProjectTypes    []string        `json:"project_types"`
	Technologies    []string        `json:"technologies"`
	Components      []component     `json:"components"`
	Workspaces      []workspaceRoot `json:"workspaces"`
	Entrypoints     []evidenceItem  `json:"entrypoints"`
	Relationships   []relationship  `json:"relationships"`
	Tests           []testLink      `json:"tests"`
	Manifests       []string        `json:"manifests"`
	Policies        []string        `json:"policies"`
	Commands        []command       `json:"commands"`
	Bounds          []string        `json:"bounds"`
}

type sourceFile struct {
	Path       string
	Dir        string
	Language   string
	Test       bool
	Body       []byte
	GoImports  []string
	RelImports []string
}

type goModule struct {
	ImportPath string
	Directory  string
}

type componentState struct {
	languages map[string]bool
	sources   int
	tests     []string
}

var safePathPattern = regexp.MustCompile(`^[A-Za-z0-9._@+:/-]+$`)
var verifyNamePattern = regexp.MustCompile(`(?i)(^|[-_.:])(test|tests|check|verify|lint|build|typecheck|smoke)([-_.:]|$)`)
var makeTargetPattern = regexp.MustCompile(`(?m)^([A-Za-z0-9][A-Za-z0-9_.-]*):(?:[^=]|$)`)
var relativeImportPattern = regexp.MustCompile(`(?m)(?:import\s+(?:[^'"\n]+\s+from\s+)?|export\s+[^'"\n]+\s+from\s+|require\s*\()\s*['"](\.{1,2}/[A-Za-z0-9._@+/-]+)['"]`)
var pythonRelativePattern = regexp.MustCompile(`(?m)^\s*from\s+(\.+[A-Za-z0-9_.]*)\s+import\s+`)
var goMainPattern = regexp.MustCompile(`(?m)^\s*func\s+main\s*\(`)

func Build(ctx context.Context, root string, analysis analyze.AnalysisResult, opts BuildOptions) (Document, error) {
	canonical, err := canonicalRoot(root)
	if err != nil {
		return Document{}, err
	}
	model, err := inspect(ctx, canonical, analysis, opts)
	if err != nil {
		return Document{}, err
	}
	encoded, err := json.Marshal(model)
	if err != nil {
		return Document{}, err
	}
	metadata := Metadata{
		SchemaVersion:     SchemaVersion,
		SourceFingerprint: digest(encoded),
	}
	body := render(model)
	metadata.BodySHA256 = digest(body)
	markerJSON, err := json.Marshal(metadata)
	if err != nil {
		return Document{}, err
	}
	content := append([]byte("<!-- afs-context "), markerJSON...)
	content = append(content, []byte(" -->\n")...)
	content = append(content, body...)
	if len(content) > MaxDocumentBytes {
		return Document{}, fmt.Errorf("generated context exceeds %d bytes", MaxDocumentBytes)
	}
	return Document{Metadata: metadata, Body: body, Content: content}, nil
}

func Decide(existing []byte, candidate Document, force bool) (Action, error) {
	if existing == nil {
		return ActionCreate, nil
	}
	metadata, body, err := Parse(existing)
	if err != nil {
		if force {
			return ActionUpdate, nil
		}
		return "", ErrUnknownContext
	}
	if digest(body) != metadata.BodySHA256 {
		if force {
			return ActionUpdate, nil
		}
		return "", ErrModifiedContext
	}
	if bytes.Equal(existing, candidate.Content) {
		return ActionCurrent, nil
	}
	return ActionUpdate, nil
}

func Parse(content []byte) (Metadata, []byte, error) {
	lineEnd := bytes.IndexByte(content, '\n')
	if lineEnd < 0 {
		return Metadata{}, nil, errors.New("missing metadata line")
	}
	const prefix = "<!-- afs-context "
	const suffix = " -->"
	line := string(content[:lineEnd])
	if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, suffix) {
		return Metadata{}, nil, errors.New("invalid metadata marker")
	}
	payload := strings.TrimSuffix(strings.TrimPrefix(line, prefix), suffix)
	decoder := json.NewDecoder(strings.NewReader(payload))
	decoder.DisallowUnknownFields()
	var metadata Metadata
	if err := decoder.Decode(&metadata); err != nil {
		return Metadata{}, nil, err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return Metadata{}, nil, errors.New("invalid metadata payload")
	}
	if metadata.SchemaVersion != SchemaVersion || !validDigest(metadata.SourceFingerprint) || !validDigest(metadata.BodySHA256) {
		return Metadata{}, nil, errors.New("unsupported context metadata")
	}
	return metadata, append([]byte(nil), content[lineEnd+1:]...), nil
}

func inspect(ctx context.Context, root string, analysis analyze.AnalysisResult, opts BuildOptions) (graphModel, error) {
	model := graphModel{
		SchemaVersion:   SchemaVersion,
		RendererVersion: rendererVersion,
		RepoShape:       safeToken(analysis.RepoShape),
		ProjectTypes:    detectedProjectTypes(analysis.ProjectTypes),
	}
	states := map[string]*componentState{}
	fileSet := map[string]bool{}
	var sources []sourceFile
	var manifests []string
	var policies []string
	var commands []command
	entries := 0
	filesRead := 0
	bytesRead := int64(0)
	unsafeOmitted := false
	stop := errors.New("lite inventory limit reached")

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			if path != root {
				model.Bounds = appendUnique(model.Bounds, "unreadable paths omitted")
				if entry != nil && entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
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
		if excludedPath(rel, opts.ExcludedPaths) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() && ignoredDir(entry.Name()) {
			return filepath.SkipDir
		}
		if strings.Count(rel, "/")+1 > maxDepth {
			model.Bounds = appendUnique(model.Bounds, fmt.Sprintf("paths deeper than %d levels omitted", maxDepth))
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entries >= maxEntries {
			model.Bounds = appendUnique(model.Bounds, fmt.Sprintf("inventory capped at %d entries", maxEntries))
			return stop
		}
		entries++
		if entry.Type()&os.ModeSymlink != 0 {
			model.Bounds = appendUnique(model.Bounds, "symlinks not followed")
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if !safePath(rel) {
			unsafeOmitted = true
			return nil
		}
		fileSet[rel] = true
		if isManifest(rel) {
			manifests = append(manifests, rel)
		}
		if isPolicy(rel) {
			policies = append(policies, rel)
		}
		language := sourceLanguage(rel)
		needBody := language != "" || rel == "Makefile" || rel == "package.json"
		if !needBody || filesRead >= maxFilesRead || bytesRead >= maxTotalBytes {
			if needBody {
				model.Bounds = appendUnique(model.Bounds, "source inspection byte/file limit reached")
			}
			return nil
		}
		body, readErr := readBoundedFile(path)
		if readErr != nil {
			if errors.Is(readErr, errFileTooLarge) {
				model.Bounds = appendUnique(model.Bounds, "oversized files omitted")
			} else {
				model.Bounds = appendUnique(model.Bounds, "unreadable or concurrently changed files omitted")
			}
			return nil
		}
		filesRead++
		bytesRead += int64(len(body))
		if rel == "Makefile" {
			commands = append(commands, makeCommands(body, rel)...)
		}
		if rel == "package.json" {
			commands = append(commands, packageCommands(body, rel)...)
		}
		if language == "" {
			return nil
		}
		dir := filepath.ToSlash(filepath.Dir(rel))
		if dir == "" {
			dir = "."
		}
		isTest := isTestPath(rel)
		state := states[dir]
		if state == nil {
			state = &componentState{languages: map[string]bool{}}
			states[dir] = state
		}
		state.languages[language] = true
		if isTest {
			state.tests = append(state.tests, rel)
		} else {
			state.sources++
		}
		source := sourceFile{Path: rel, Dir: dir, Language: language, Test: isTest, Body: body}
		if language == "go" {
			source.GoImports = parseGoImports(body)
		} else if language == "javascript/typescript" {
			source.RelImports = regexpMatches(relativeImportPattern, body)
		} else if language == "python" {
			source.RelImports = regexpMatches(pythonRelativePattern, body)
		}
		sources = append(sources, source)
		return nil
	})
	if err == stop {
		err = nil
	}
	if err != nil {
		return graphModel{}, err
	}
	if unsafeOmitted {
		model.Bounds = appendUnique(model.Bounds, "unsafe or Markdown-active paths omitted")
	}

	for _, policy := range opts.PlannedPolicies {
		policy = filepath.ToSlash(filepath.Clean(policy))
		if safePath(policy) && isPolicy(policy) {
			policies = append(policies, policy)
		}
	}
	model.Technologies = detectedTechnologies(analysis.Technologies)
	if len(opts.ExcludedPaths) > 0 {
		model.Technologies = filterTechnologiesForInventory(model.Technologies, fileSet)
	}
	model.Components = buildComponents(states, &model.Bounds)
	model.Workspaces = buildWorkspaceRoots(manifests, &model.Bounds)
	model.Entrypoints = buildEntrypoints(sources, fileSet, &model.Bounds)
	model.Relationships = buildRelationships(sources, states, root, manifests, &model.Bounds)
	model.Tests = buildTestLinks(states, &model.Bounds)
	model.Manifests = capStrings(uniqueSorted(manifests), 60, "manifest/config list capped", &model.Bounds)
	model.Policies = capStrings(filterApplicablePolicies(policies, opts.PlannedPolicies, analysis, states), 60, "policy list capped", &model.Bounds)
	model.Commands = capCommands(commands, 40, &model.Bounds)
	model.Bounds = uniqueSorted(model.Bounds)
	return model, nil
}

func buildComponents(states map[string]*componentState, bounds *[]string) []component {
	paths := make([]string, 0, len(states))
	for path := range states {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	if len(paths) > 80 {
		paths = paths[:80]
		*bounds = appendUnique(*bounds, "component list capped at 80")
	}
	result := make([]component, 0, len(paths))
	for _, path := range paths {
		state := states[path]
		languages := make([]string, 0, len(state.languages))
		for language := range state.languages {
			languages = append(languages, language)
		}
		sort.Strings(languages)
		result = append(result, component{Path: path, Languages: languages, Sources: state.sources, Tests: len(state.tests)})
	}
	return result
}

func buildWorkspaceRoots(manifests []string, bounds *[]string) []workspaceRoot {
	seen := map[string]bool{}
	var result []workspaceRoot
	for _, manifest := range uniqueSorted(manifests) {
		kind := packageManifestKind(manifest)
		if kind == "" {
			continue
		}
		path := filepath.ToSlash(filepath.Dir(manifest))
		if path == "" {
			path = "."
		}
		key := path + "\x00" + kind
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, workspaceRoot{Path: path, Kind: kind, Evidence: manifest})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Path == result[j].Path {
			return result[i].Kind < result[j].Kind
		}
		return result[i].Path < result[j].Path
	})
	if len(result) > 50 {
		result = result[:50]
		*bounds = appendUnique(*bounds, "package/workspace root list capped at 50")
	}
	return result
}

func buildEntrypoints(sources []sourceFile, files map[string]bool, bounds *[]string) []evidenceItem {
	var result []evidenceItem
	for _, source := range sources {
		base := strings.ToLower(filepath.Base(source.Path))
		kind := ""
		confidence := "medium"
		switch {
		case source.Language == "go" && goMainPattern.Match(source.Body) && bytes.Contains(source.Body, []byte("package main")):
			kind, confidence = "executable", "high"
		case base == "main.py" || base == "main.js" || base == "main.ts" || base == "main.swift" || base == "main.kt" || base == "program.cs":
			kind = "conventional entrypoint"
		}
		if kind != "" {
			result = append(result, evidenceItem{Path: source.Path, Kind: kind, Confidence: confidence, Evidence: source.Path})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	if len(result) > 50 {
		result = result[:50]
		*bounds = appendUnique(*bounds, "entrypoint list capped at 50")
	}
	_ = files
	return result
}

func buildRelationships(sources []sourceFile, states map[string]*componentState, root string, manifests []string, bounds *[]string) []relationship {
	modules := readGoModules(root, manifests, bounds)
	seen := map[string]bool{}
	var result []relationship
	for _, source := range sources {
		if source.Test {
			continue
		}
		for _, imported := range source.GoImports {
			to := ""
			for _, module := range modules {
				if imported != module.ImportPath && !strings.HasPrefix(imported, module.ImportPath+"/") {
					continue
				}
				suffix := strings.TrimPrefix(strings.TrimPrefix(imported, module.ImportPath), "/")
				to = filepath.ToSlash(filepath.Clean(filepath.Join(module.Directory, suffix)))
				if to == "" {
					to = "."
				}
				break
			}
			if to == "" {
				continue
			}
			result = appendRelationship(result, seen, source.Dir, to, "local import", "high", source.Path, states)
		}
		for _, imported := range source.RelImports {
			to := resolveRelativeComponent(source, imported, states)
			if to == "" {
				continue
			}
			confidence := "high"
			if source.Language == "python" {
				confidence = "medium"
			}
			result = appendRelationship(result, seen, source.Dir, to, "local import", confidence, source.Path, states)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		left := result[i].From + "\x00" + result[i].To + "\x00" + result[i].Evidence
		right := result[j].From + "\x00" + result[j].To + "\x00" + result[j].Evidence
		return left < right
	})
	if len(result) > 100 {
		result = result[:100]
		*bounds = appendUnique(*bounds, "relationship list capped at 100")
	}
	return result
}

func appendRelationship(items []relationship, seen map[string]bool, from, to, kind, confidence, evidence string, states map[string]*componentState) []relationship {
	if from == to || states[to] == nil || !safePath(from) || !safePath(to) || !safePath(evidence) {
		return items
	}
	key := from + "\x00" + to + "\x00" + kind
	if seen[key] {
		return items
	}
	seen[key] = true
	return append(items, relationship{From: from, To: to, Kind: kind, Confidence: confidence, Evidence: evidence})
}

func buildTestLinks(states map[string]*componentState, bounds *[]string) []testLink {
	var result []testLink
	for path, state := range states {
		if len(state.tests) == 0 {
			continue
		}
		examples := append([]string(nil), state.tests...)
		sort.Strings(examples)
		if len(examples) > 3 {
			examples = examples[:3]
		}
		result = append(result, testLink{Component: path, Count: len(state.tests), Examples: examples})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Component < result[j].Component })
	if len(result) > 60 {
		result = result[:60]
		*bounds = appendUnique(*bounds, "test relationship list capped at 60")
	}
	return result
}

func render(model graphModel) []byte {
	const bodyLimit = MaxDocumentBytes - 320
	var builder strings.Builder
	outputFull := false
	add := func(line string) bool {
		if outputFull {
			return false
		}
		if builder.Len()+len(line)+1 > bodyLimit-120 {
			outputFull = true
			return false
		}
		builder.WriteString(line)
		builder.WriteByte('\n')
		return true
	}
	add("# Repository Context")
	add("")
	add("> Generated by `afs map`. Derived evidence only: source, `AGENTS.md`, and applicable rules remain authoritative. Do not edit manually.")
	add("")
	add("## Overview")
	add("")
	add("- Repository shape: `" + valueOrUnknown(model.RepoShape) + "`")
	add("- Project types: " + codeList(model.ProjectTypes))
	add("- Technologies: " + codeList(model.Technologies))
	renderItems := func(title string, count, budget int, item func(int) string) {
		add("")
		add("## " + title)
		add("")
		if count == 0 {
			add("- None detected within Lite bounds.")
			return
		}
		start := builder.Len()
		for index := 0; index < count; index++ {
			line := "- " + item(index)
			if builder.Len()-start+len(line)+1 > budget {
				add("- Additional items omitted by the section budget.")
				return
			}
			if !add(line) {
				return
			}
		}
	}
	renderItems("Components", len(model.Components), 1700, func(index int) string {
		item := model.Components[index]
		return "`" + item.Path + "` — " + strings.Join(item.Languages, ", ") + fmt.Sprintf("; %d source, %d test files", item.Sources, item.Tests)
	})
	renderItems("Package and Workspace Roots", len(model.Workspaces), 900, func(index int) string {
		item := model.Workspaces[index]
		return "`" + item.Path + "` — " + item.Kind + "; evidence: `" + item.Evidence + "`"
	})
	renderItems("Entrypoints", len(model.Entrypoints), 900, func(index int) string {
		item := model.Entrypoints[index]
		return "`" + item.Path + "` — " + item.Kind + "; confidence: " + item.Confidence + "; evidence: `" + item.Evidence + "`"
	})
	renderItems("Direct Local Relationships", len(model.Relationships), 2600, func(index int) string {
		item := model.Relationships[index]
		return "`" + item.From + "` -> `" + item.To + "` — " + item.Kind + "; confidence: " + item.Confidence + "; evidence: `" + item.Evidence + "`"
	})
	renderItems("Tests", len(model.Tests), 1300, func(index int) string {
		item := model.Tests[index]
		return "`" + item.Component + "` — " + fmt.Sprintf("%d test files; examples: %s", item.Count, codeList(item.Examples))
	})
	renderItems("Manifests and Configuration", len(model.Manifests), 700, func(index int) string { return "`" + model.Manifests[index] + "`" })
	renderItems("Applicable Policy", len(model.Policies), 900, func(index int) string { return "`" + model.Policies[index] + "`" })
	renderItems("Verification Commands", len(model.Commands), 1000, func(index int) string {
		item := model.Commands[index]
		return "`" + item.Invocation + "` — evidence: `" + item.Evidence + "` (not executed)"
	})
	bounds := append([]string(nil), model.Bounds...)
	if outputFull {
		bounds = appendUnique(bounds, "rendered output capped at 12 KiB")
	}
	if len(bounds) > 0 {
		if builder.Len()+len("\n## Lite Bounds\n\n- Additional evidence was omitted by bounded scanning or rendering.\n") < bodyLimit {
			builder.WriteString("\n## Lite Bounds\n\n")
			for _, bound := range bounds {
				line := "- " + bound + "\n"
				if builder.Len()+len(line) > bodyLimit {
					break
				}
				builder.WriteString(line)
			}
		}
	}
	return []byte(builder.String())
}

var errFileTooLarge = errors.New("file exceeds Lite read limit")

func readBoundedFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, errors.New("not a regular file")
	}
	if info.Size() > maxFileBytes {
		return nil, errFileTooLarge
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	opened, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	if !os.SameFile(info, opened) || opened.Size() != info.Size() || !opened.ModTime().Equal(info.ModTime()) {
		file.Close()
		return nil, errors.New("file changed during context inspection")
	}
	body, err := io.ReadAll(io.LimitReader(file, maxFileBytes+1))
	if err != nil {
		file.Close()
		return nil, err
	}
	afterOpen, statErr := file.Stat()
	closeErr := file.Close()
	if statErr != nil {
		return nil, statErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	afterPath, err := os.Lstat(path)
	if err != nil || afterPath.Mode()&os.ModeSymlink != 0 || !os.SameFile(info, afterOpen) || !os.SameFile(afterOpen, afterPath) || afterPath.Size() != info.Size() || !afterPath.ModTime().Equal(info.ModTime()) {
		return nil, errors.New("file changed during context inspection")
	}
	if len(body) > maxFileBytes {
		return nil, errFileTooLarge
	}
	return body, nil
}

func canonicalRoot(root string) (string, error) {
	if root == "" {
		return "", errors.New("work directory is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("work directory must be a directory")
	}
	return filepath.Clean(canonical), nil
}

func ignoredDir(name string) bool {
	switch strings.ToLower(name) {
	case ".git", ".agent47", "node_modules", "vendor", "templates", "dist", "build", "out", "target", ".next", "coverage", ".cache", ".venv", "venv", ".tox", ".gradle", ".terraform", ".turbo", "generated":
		return true
	default:
		return false
	}
}

func excludedPath(path string, exclusions []string) bool {
	for _, excluded := range exclusions {
		excluded = filepath.ToSlash(filepath.Clean(excluded))
		if excluded == "." || excluded == "" || strings.HasPrefix(excluded, "../") || filepath.IsAbs(excluded) {
			continue
		}
		if path == excluded || strings.HasPrefix(path, excluded+"/") {
			return true
		}
	}
	return false
}

func safePath(path string) bool {
	if path == "" || len(path) > 240 || !safePathPattern.MatchString(path) {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	return clean == path && path != ".." && !strings.HasPrefix(path, "../")
}

func safeToken(value string) string {
	if safePath(value) && !strings.Contains(value, "/") {
		return value
	}
	return "unknown"
}

func sourceLanguage(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs":
		return "javascript/typescript"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".swift":
		return "swift"
	case ".java", ".kt", ".kts":
		return "java/kotlin"
	case ".cs":
		return "csharp"
	case ".c", ".cc", ".cpp", ".cxx", ".h", ".hpp":
		return "c/cpp"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".ex", ".exs":
		return "elixir"
	case ".dart":
		return "dart"
	case ".sh", ".bash", ".zsh", ".ps1":
		return "shell"
	default:
		return ""
	}
}

func isTestPath(path string) bool {
	lower := strings.ToLower(path)
	base := strings.ToLower(filepath.Base(path))
	return strings.Contains(lower, "/test/") || strings.Contains(lower, "/tests/") || strings.Contains(lower, "/__tests__/") ||
		strings.HasPrefix(lower, "test/") || strings.HasPrefix(lower, "tests/") || strings.HasSuffix(base, "_test.go") ||
		strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") || strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py")
}

func isManifest(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	if strings.HasSuffix(base, ".csproj") || strings.HasSuffix(base, ".sln") || strings.HasSuffix(base, ".xcworkspace") || strings.HasSuffix(base, ".xcodeproj") || strings.HasSuffix(path, ".tf") {
		return true
	}
	switch base {
	case "go.mod", "go.work", "package.json", "pnpm-workspace.yaml", "pyproject.toml", "cargo.toml", "package.swift", "pom.xml", "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts", "composer.json", "gemfile", "mix.exs", "pubspec.yaml", "dockerfile", "docker-compose.yml", "docker-compose.yaml", "makefile", "taskfile.yml", "taskfile.yaml", "justfile":
		return true
	default:
		return false
	}
}

func packageManifestKind(path string) string {
	base := strings.ToLower(filepath.Base(path))
	if strings.HasSuffix(base, ".csproj") {
		return "C# project"
	}
	if strings.HasSuffix(base, ".sln") {
		return "C# solution"
	}
	switch base {
	case "go.mod":
		return "Go module"
	case "go.work":
		return "Go workspace"
	case "package.json":
		return "Node package/workspace"
	case "pnpm-workspace.yaml":
		return "pnpm workspace"
	case "pyproject.toml":
		return "Python project"
	case "cargo.toml":
		return "Rust package/workspace"
	case "package.swift":
		return "Swift package"
	case "pom.xml":
		return "Maven project"
	case "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts":
		return "Gradle project"
	case "composer.json":
		return "PHP package"
	case "gemfile":
		return "Ruby project"
	case "mix.exs":
		return "Elixir project"
	case "pubspec.yaml":
		return "Dart package"
	default:
		return ""
	}
}

func isPolicy(path string) bool {
	base := filepath.Base(path)
	return base == "AGENTS.md" || (strings.HasPrefix(path, "rules/") && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")))
}

func filterApplicablePolicies(values, planned []string, analysis analyze.AnalysisResult, states map[string]*componentState) []string {
	plannedSet := map[string]bool{}
	for _, path := range planned {
		plannedSet[filepath.ToSlash(filepath.Clean(path))] = true
	}
	types := map[string]bool{}
	for _, value := range analysis.ProjectTypes {
		types[value.ID] = true
	}
	technologies := map[string]bool{}
	for _, value := range analysis.Technologies {
		technologies[value.ID] = true
	}
	hasTests := false
	for _, state := range states {
		if len(state.tests) > 0 {
			hasTests = true
			break
		}
	}
	known := map[string]bool{
		"rules-backend.yaml": true, "rules-cli.yaml": true, "rules-cross.yaml": true,
		"rules-desktop.yaml": true, "rules-frontend.yaml": true, "rules-go.yaml": true,
		"rules-infra.yaml": true, "rules-mobile.yaml": true, "rules-monorepo-tooling.yaml": true,
		"rules-plugin.yaml": true, "rules-scripts.yaml": true, "security-csharp.yaml": true,
		"security-global.yaml": true, "security-go.yaml": true, "security-java-kotlin.yaml": true,
		"security-js-ts.yaml": true, "security-py.yaml": true, "security-shell.yaml": true,
		"security-swift.yaml": true, "shared-cli-behavior.yaml": true, "shared-testing.yaml": true,
	}
	var result []string
	for _, path := range uniqueSorted(values) {
		base := filepath.Base(path)
		include := plannedSet[path] || base == "AGENTS.md"
		switch base {
		case "rules-cross.yaml", "security-global.yaml", "security-shell.yaml":
			include = true
		case "rules-backend.yaml", "rules-cli.yaml", "rules-desktop.yaml", "rules-frontend.yaml", "rules-infra.yaml", "rules-mobile.yaml", "rules-monorepo-tooling.yaml", "rules-plugin.yaml", "rules-scripts.yaml":
			projectType := strings.TrimSuffix(strings.TrimPrefix(base, "rules-"), ".yaml")
			include = include || types[projectType]
		case "rules-go.yaml", "security-go.yaml":
			include = include || technologies["go"]
		case "security-js-ts.yaml":
			include = include || technologies["node"] || technologies["typescript"]
		case "security-py.yaml":
			include = include || technologies["python"]
		case "security-java-kotlin.yaml":
			include = include || technologies["java-kotlin"]
		case "security-swift.yaml":
			include = include || technologies["swift"]
		case "security-csharp.yaml":
			include = include || technologies["csharp"]
		case "shared-cli-behavior.yaml":
			include = include || types["cli"]
		case "shared-testing.yaml":
			include = include || hasTests
		}
		if !known[base] {
			include = true
		}
		if include {
			result = append(result, path)
		}
	}
	return uniqueSorted(result)
}

func parseGoImports(body []byte) []string {
	file, err := parser.ParseFile(token.NewFileSet(), "source.go", body, parser.ImportsOnly)
	if err != nil {
		return nil
	}
	var result []string
	for _, imported := range file.Imports {
		value := strings.Trim(imported.Path.Value, `"`)
		if safePath(value) {
			result = append(result, value)
		}
	}
	return uniqueSorted(result)
}

func readGoModules(root string, manifests []string, bounds *[]string) []goModule {
	var result []goModule
	inspected := 0
	totalBytes := 0
	for _, manifest := range manifests {
		if filepath.Base(manifest) != "go.mod" {
			continue
		}
		if inspected >= 100 || totalBytes >= 1024*1024 {
			*bounds = appendUnique(*bounds, "Go module relationship scan capped")
			break
		}
		body, err := readBoundedFile(filepath.Join(root, filepath.FromSlash(manifest)))
		if err != nil {
			*bounds = appendUnique(*bounds, "unreadable Go module manifests omitted")
			continue
		}
		inspected++
		totalBytes += len(body)
		for _, line := range strings.Split(string(body), "\n") {
			fields := strings.Fields(line)
			if len(fields) != 2 || fields[0] != "module" || !safePath(fields[1]) {
				continue
			}
			dir := filepath.ToSlash(filepath.Dir(manifest))
			if dir == "" {
				dir = "."
			}
			result = append(result, goModule{ImportPath: fields[1], Directory: dir})
			break
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if len(result[i].ImportPath) == len(result[j].ImportPath) {
			return result[i].ImportPath < result[j].ImportPath
		}
		return len(result[i].ImportPath) > len(result[j].ImportPath)
	})
	return result
}

func regexpMatches(pattern *regexp.Regexp, body []byte) []string {
	var result []string
	for _, match := range pattern.FindAllSubmatch(body, -1) {
		if len(match) > 1 {
			value := string(match[1])
			if safeImportSpec(value) {
				result = append(result, value)
			}
		}
	}
	return uniqueSorted(result)
}

func safeImportSpec(value string) bool {
	return value != "" && len(value) <= 240 && safePathPattern.MatchString(value) &&
		!strings.Contains(value, "//") && value != "." && value != ".."
}

func resolveRelativeComponent(source sourceFile, imported string, states map[string]*componentState) string {
	base := source.Dir
	value := imported
	if source.Language == "python" {
		dots := len(value) - len(strings.TrimLeft(value, "."))
		for count := 1; count < dots; count++ {
			base = filepath.ToSlash(filepath.Dir(base))
		}
		value = strings.TrimLeft(value, ".")
		value = strings.ReplaceAll(value, ".", "/")
	}
	joined := filepath.ToSlash(filepath.Clean(filepath.Join(base, value)))
	joined = strings.TrimSuffix(joined, filepath.Ext(joined))
	for candidate := joined; candidate != "." && candidate != "/"; candidate = filepath.ToSlash(filepath.Dir(candidate)) {
		if states[candidate] != nil {
			return candidate
		}
	}
	if states["."] != nil {
		return "."
	}
	return ""
}

func makeCommands(body []byte, evidence string) []command {
	var result []command
	for _, match := range makeTargetPattern.FindAllSubmatch(body, -1) {
		name := string(match[1])
		if verificationName(name) {
			result = append(result, command{Invocation: "make " + name, Evidence: evidence})
		}
	}
	return result
}

func packageCommands(body []byte, evidence string) []command {
	var value struct {
		Scripts        map[string]json.RawMessage `json:"scripts"`
		PackageManager string                     `json:"packageManager"`
	}
	if json.Unmarshal(body, &value) != nil {
		return nil
	}
	manager := "npm"
	declaredManager := strings.SplitN(value.PackageManager, "@", 2)[0]
	if declaredManager == "pnpm" || declaredManager == "yarn" || declaredManager == "bun" {
		manager = declaredManager
	}
	var result []command
	for name := range value.Scripts {
		if safePath(name) && !strings.Contains(name, "/") && verificationName(name) {
			result = append(result, command{Invocation: manager + " run " + name, Evidence: evidence})
		}
	}
	return result
}

func verificationName(name string) bool {
	lower := strings.ToLower(name)
	return !strings.HasPrefix(lower, "clean") && !strings.HasPrefix(lower, "remove") && verifyNamePattern.MatchString(name)
}

func capCommands(values []command, limit int, bounds *[]string) []command {
	seen := map[string]bool{}
	var result []command
	for _, value := range values {
		key := value.Invocation + "\x00" + value.Evidence
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Invocation == result[j].Invocation {
			return result[i].Evidence < result[j].Evidence
		}
		return result[i].Invocation < result[j].Invocation
	})
	if len(result) > limit {
		result = result[:limit]
		*bounds = appendUnique(*bounds, fmt.Sprintf("verification command list capped at %d", limit))
	}
	return result
}

func capStrings(values []string, limit int, note string, bounds *[]string) []string {
	if len(values) <= limit {
		return values
	}
	*bounds = appendUnique(*bounds, note)
	return values[:limit]
}

func detectedProjectTypes(values []analyze.DetectedProjectType) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if token := safeToken(value.ID); token != "unknown" {
			result = append(result, token)
		}
	}
	return uniqueSorted(result)
}

func detectedTechnologies(values []analyze.DetectedTechnology) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if token := safeToken(value.ID); token != "unknown" {
			result = append(result, token)
		}
	}
	return uniqueSorted(result)
}

func filterTechnologiesForInventory(values []string, files map[string]bool) []string {
	var result []string
	for _, value := range values {
		if technologyHasInventoryEvidence(value, files) {
			result = append(result, value)
		}
	}
	return result
}

func technologyHasInventoryEvidence(value string, files map[string]bool) bool {
	hasBase := func(names ...string) bool {
		for path := range files {
			base := strings.ToLower(filepath.Base(path))
			for _, name := range names {
				if base == name {
					return true
				}
			}
		}
		return false
	}
	hasSuffix := func(suffixes ...string) bool {
		for path := range files {
			lower := strings.ToLower(path)
			for _, suffix := range suffixes {
				if strings.HasSuffix(lower, suffix) {
					return true
				}
			}
		}
		return false
	}
	hasPrefix := func(prefixes ...string) bool {
		for path := range files {
			lower := strings.ToLower(path)
			for _, prefix := range prefixes {
				if strings.HasPrefix(lower, prefix) {
					return true
				}
			}
		}
		return false
	}

	switch value {
	case "go":
		return hasBase("go.mod", "go.work") || hasSuffix(".go")
	case "node":
		return hasBase("package.json") || hasSuffix(".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx")
	case "typescript":
		return hasBase("package.json") || hasSuffix(".ts", ".tsx")
	case "react":
		return hasBase("package.json") || hasSuffix(".jsx", ".tsx")
	case "tailwind":
		return hasBase("package.json") || hasPrefix("tailwind.config.")
	case "java-kotlin":
		return hasBase("pom.xml", "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts") || hasSuffix(".java", ".kt", ".kts")
	case "swift":
		return hasBase("package.swift") || hasSuffix(".swift")
	case "python":
		return hasBase("pyproject.toml", "requirements.txt") || hasSuffix(".py")
	case "csharp":
		return hasSuffix(".cs", ".csproj", ".sln")
	case "shell":
		return hasBase("install.sh") || hasSuffix(".sh", ".bash", ".zsh", ".bats")
	case "infra":
		return hasBase("helmfile.yaml", "helmfile.yml") || hasSuffix(".tf") || hasPrefix("infra/", "terraform/", "charts/", "helm/", "k8s/", "kubernetes/")
	case "workspace-tooling":
		return hasBase("pnpm-workspace.yaml", "turbo.json", "nx.json", "lerna.json") || hasPrefix("apps/", "packages/")
	case "desktop-runtime":
		return hasBase("package.json", "wails.json", "package.swift") || hasPrefix("src-tauri/")
	case "plugin-hosting":
		return hasBase("plugin.json") || hasPrefix(".codex-plugin/", "plugin/", "plugins/")
	case "vitest":
		return hasBase("package.json") || hasPrefix("vitest.config.", "vitest.workspace.")
	case "jest":
		return hasBase("package.json") || hasPrefix("jest.config.", "jest.setup.")
	case "playwright":
		return hasBase("package.json") || hasPrefix("playwright.config.", "playwright/")
	case "cypress":
		return hasBase("package.json") || hasPrefix("cypress.config.", "cypress/")
	case "go-test":
		return hasSuffix("_test.go")
	case "bats":
		return hasSuffix(".bats")
	default:
		return true
	}
}

func codeList(values []string) string {
	if len(values) == 0 {
		return "none detected"
	}
	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = "`" + value + "`"
	}
	return strings.Join(quoted, ", ")
}

func valueOrUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
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

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func digest(value []byte) string {
	hash := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(hash[:])
}

func validDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil && len(decoded) == sha256.Size && value == strings.ToLower(value)
}

package analyze

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const maxInspectedFileBytes int64 = 262144

const (
	maxRepositoryEntries = 10000
	maxRepositoryDepth   = 32
)

type PolicyState string

const (
	PolicyAbsent  PolicyState = "absent"
	PolicyPresent PolicyState = "present"
	PolicyValid   PolicyState = "valid"
	PolicyInvalid PolicyState = "invalid"
)

type AgentPolicyFile struct {
	Path  string `json:"path"`
	Scope string `json:"scope"`
}

type ToolPolicy struct {
	State    PolicyState `json:"state"`
	Evidence []string    `json:"evidence"`
	Warnings []string    `json:"warnings"`
}

type AgentPolicyTools struct {
	Claude   ToolPolicy `json:"claude"`
	Codex    ToolPolicy `json:"codex"`
	OpenCode ToolPolicy `json:"opencode"`
}

type AgentPolicy struct {
	SchemaVersion int               `json:"schema_version"`
	AgentsFiles   []AgentPolicyFile `json:"agents_files"`
	Tools         AgentPolicyTools  `json:"tools"`
	Legacy        []string          `json:"legacy"`
	Warnings      []string          `json:"warnings"`
}

type safeReadResult struct {
	Body      []byte
	Truncated bool
}

type repositoryEntry struct {
	Path  string
	IsDir bool
}

type repositoryInventory struct {
	Root        string
	Entries     []repositoryEntry
	Paths       []string
	Skipped     []AuditNote
	Truncations []AuditNote
}

func detectAgentPolicy(root string) (AgentPolicy, error) {
	inventory, err := inspectRepository(context.Background(), root)
	if err != nil {
		return AgentPolicy{}, err
	}
	return detectAgentPolicyFromInventory(root, inventory)
}

func detectAgentPolicyFromInventory(root string, inventory repositoryInventory) (AgentPolicy, error) {
	policy := AgentPolicy{
		SchemaVersion: 1,
		AgentsFiles:   []AgentPolicyFile{},
		Tools: AgentPolicyTools{
			Claude:   emptyToolPolicy(),
			Codex:    emptyToolPolicy(),
			OpenCode: emptyToolPolicy(),
		},
		Legacy:   []string{},
		Warnings: []string{},
	}

	paths := inventory.Paths
	pathSet := make(map[string]bool, len(paths))
	for _, rel := range paths {
		pathSet[rel] = true
		base := filepath.Base(rel)
		if base == "AGENTS.md" {
			scope := filepath.ToSlash(filepath.Dir(rel))
			policy.AgentsFiles = append(policy.AgentsFiles, AgentPolicyFile{Path: rel, Scope: scope})
		} else if strings.EqualFold(base, "AGENTS.md") {
			policy.Warnings = append(policy.Warnings, fmt.Sprintf("case-mismatched agent policy path: %s", rel))
		}
	}
	sort.Slice(policy.AgentsFiles, func(i, j int) bool { return policy.AgentsFiles[i].Path < policy.AgentsFiles[j].Path })

	policy.Tools.Claude = detectClaudePolicy(root, paths, pathSet)
	policy.Tools.Codex = detectCodexPolicy(root, paths, pathSet)
	policy.Tools.OpenCode = detectOpenCodePolicy(root, paths, pathSet)
	for _, note := range append(append([]AuditNote{}, inventory.Skipped...), inventory.Truncations...) {
		warningPath := strings.ToLower(note.Path)
		warning := note.Reason
		if note.Path != "" {
			warning += ": " + note.Path
		}
		switch {
		case warningPath == "claude.md" || strings.HasPrefix(warningPath, ".claude/"):
			policy.Tools.Claude.Warnings = append(policy.Tools.Claude.Warnings, warning)
		case strings.HasPrefix(warningPath, ".codex/"):
			policy.Tools.Codex.Warnings = append(policy.Tools.Codex.Warnings, warning)
		case strings.HasPrefix(warningPath, ".opencode/"):
			policy.Tools.OpenCode.Warnings = append(policy.Tools.OpenCode.Warnings, warning)
		default:
			policy.Warnings = append(policy.Warnings, warning)
		}
	}
	for _, legacy := range []string{"skills", "prompts"} {
		if pathSet[legacy] || hasPathPrefix(paths, legacy+"/") {
			policy.Legacy = append(policy.Legacy, legacy+"/")
		}
	}
	policy.Warnings = uniqueSorted(policy.Warnings)
	return policy, nil
}

func emptyToolPolicy() ToolPolicy {
	return ToolPolicy{State: PolicyAbsent, Evidence: []string{}, Warnings: []string{}}
}

func detectClaudePolicy(root string, paths []string, pathSet map[string]bool) ToolPolicy {
	result := emptyToolPolicy()
	for _, rel := range paths {
		lower := strings.ToLower(rel)
		if rel == "CLAUDE.md" || rel == "CLAUDE.local.md" || rel == ".claude" || strings.HasPrefix(rel, ".claude/") {
			result.Evidence = append(result.Evidence, rel)
		} else if lower == "claude.md" || lower == ".claude" || strings.HasPrefix(lower, ".claude/") {
			result.Warnings = append(result.Warnings, "case-mismatched Claude path: "+rel)
		}
	}
	result.Evidence = uniqueSorted(result.Evidence)
	if len(result.Evidence) > 0 {
		result.State = PolicyPresent
	}
	valid := false
	invalid := false
	for _, rel := range result.Evidence {
		if rel != "CLAUDE.md" && rel != "CLAUDE.local.md" && !isFileUnder(rel, ".claude/rules", ".md") && !isFileUnder(rel, ".claude/agents", ".md") {
			continue
		}
		read, err := safeReadRepositoryFile(root, rel, maxInspectedFileBytes)
		if err != nil {
			invalid = true
			result.Warnings = append(result.Warnings, fmt.Sprintf("unreadable Claude instruction surface: %s", rel))
			continue
		}
		if read.Truncated {
			invalid = true
			result.Warnings = append(result.Warnings, fmt.Sprintf("oversized Claude instruction surface: %s", rel))
			continue
		}
		if len(bytes.TrimSpace(read.Body)) > 0 {
			valid = true
		}
	}
	if invalid {
		result.State = PolicyInvalid
	} else if valid {
		result.State = PolicyValid
	} else if len(result.Evidence) > 0 {
		result.Warnings = append(result.Warnings, "empty or incomplete Claude policy directory")
	}
	_ = pathSet
	result.Warnings = uniqueSorted(result.Warnings)
	return result
}

func detectCodexPolicy(root string, paths []string, pathSet map[string]bool) ToolPolicy {
	result := emptyToolPolicy()
	for _, rel := range paths {
		lower := strings.ToLower(rel)
		if rel == ".codex" || strings.HasPrefix(rel, ".codex/") {
			result.Evidence = append(result.Evidence, rel)
		} else if lower == ".codex" || strings.HasPrefix(lower, ".codex/") {
			result.Warnings = append(result.Warnings, "case-mismatched Codex path: "+rel)
		}
	}
	result.Evidence = uniqueSorted(result.Evidence)
	if len(result.Evidence) > 0 {
		result.State = PolicyPresent
	}
	validRule := false
	for _, rel := range result.Evidence {
		if isFileUnder(rel, ".codex/rules", "") {
			read, err := safeReadRepositoryFile(root, rel, maxInspectedFileBytes)
			if err == nil && !read.Truncated && len(bytes.TrimSpace(read.Body)) > 0 {
				validRule = true
			} else if err != nil || read.Truncated {
				result.State = PolicyInvalid
				result.Warnings = append(result.Warnings, "unreadable or oversized Codex rule: "+rel)
			}
		}
	}
	if pathSet[".codex/config.toml"] {
		read, err := safeReadRepositoryFile(root, ".codex/config.toml", maxInspectedFileBytes)
		if err != nil || read.Truncated || !validMinimalTOML(read.Body) {
			result.State = PolicyInvalid
			result.Warnings = append(result.Warnings, "malformed or unreadable Codex config: .codex/config.toml")
		} else if result.State != PolicyInvalid {
			result.State = PolicyValid
		}
	} else if result.State != PolicyInvalid && validRule {
		result.State = PolicyValid
	} else if result.State != PolicyInvalid && len(result.Evidence) > 0 {
		result.Warnings = append(result.Warnings, "empty or incomplete Codex policy directory")
	}
	result.Warnings = uniqueSorted(result.Warnings)
	return result
}

func detectOpenCodePolicy(root string, paths []string, pathSet map[string]bool) ToolPolicy {
	result := emptyToolPolicy()
	for _, rel := range paths {
		lower := strings.ToLower(rel)
		if rel == ".opencode" || strings.HasPrefix(rel, ".opencode/") {
			result.Evidence = append(result.Evidence, rel)
		} else if lower == ".opencode" || strings.HasPrefix(lower, ".opencode/") {
			result.Warnings = append(result.Warnings, "case-mismatched OpenCode path: "+rel)
		}
	}
	result.Evidence = uniqueSorted(result.Evidence)
	if len(result.Evidence) > 0 {
		result.State = PolicyPresent
	}
	validInstruction := false
	for _, rel := range result.Evidence {
		if isFileUnder(rel, ".opencode/agents", ".md") || isFileUnder(rel, ".opencode/commands", ".md") {
			read, err := safeReadRepositoryFile(root, rel, maxInspectedFileBytes)
			if err == nil && !read.Truncated && len(bytes.TrimSpace(read.Body)) > 0 {
				validInstruction = true
			} else if err != nil || read.Truncated {
				result.State = PolicyInvalid
				result.Warnings = append(result.Warnings, "unreadable or oversized OpenCode instruction: "+rel)
			}
		}
	}
	config := ""
	if pathSet[".opencode/opencode.json"] {
		config = ".opencode/opencode.json"
	} else if pathSet[".opencode/opencode.jsonc"] {
		config = ".opencode/opencode.jsonc"
	}
	if config != "" {
		read, err := safeReadRepositoryFile(root, config, maxInspectedFileBytes)
		if err != nil || read.Truncated || !validJSONConfig(read.Body, strings.HasSuffix(config, ".jsonc")) {
			result.State = PolicyInvalid
			result.Warnings = append(result.Warnings, "malformed or unreadable OpenCode config: "+config)
		} else if result.State != PolicyInvalid {
			result.State = PolicyValid
		}
	} else if result.State != PolicyInvalid && validInstruction {
		result.State = PolicyValid
	} else if result.State != PolicyInvalid && len(result.Evidence) > 0 {
		result.Warnings = append(result.Warnings, "empty or incomplete OpenCode policy directory")
	}
	result.Warnings = uniqueSorted(result.Warnings)
	return result
}

func repositoryPaths(root string) ([]string, []AuditNote, []AuditNote, error) {
	inventory, err := inspectRepository(context.Background(), root)
	return inventory.Paths, inventory.Skipped, inventory.Truncations, err
}

func inspectRepository(ctx context.Context, root string) (repositoryInventory, error) {
	canonical, err := canonicalRoot(root)
	if err != nil {
		return repositoryInventory{}, err
	}
	inventory := repositoryInventory{Root: canonical}
	var paths []string
	var skipped []AuditNote
	var truncations []AuditNote
	entries := 0
	stop := fmt.Errorf("repository entry limit reached")
	err = filepath.WalkDir(canonical, func(path string, d os.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			if path == canonical {
				return walkErr
			}
			rel, relErr := filepath.Rel(canonical, path)
			if relErr == nil {
				skipped = append(skipped, AuditNote{Path: filepath.ToSlash(rel), Reason: "unreadable path"})
			}
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path == canonical {
			return nil
		}
		rel, err := filepath.Rel(canonical, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		depth := strings.Count(rel, "/") + 1
		if depth > maxRepositoryDepth {
			truncations = append(truncations, AuditNote{Path: rel, Reason: "repository traversal depth limit reached"})
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entries >= maxRepositoryEntries {
			truncations = append(truncations, AuditNote{Reason: fmt.Sprintf("repository entry limit reached: %d", maxRepositoryEntries)})
			return stop
		}
		entries++
		if d.Type()&os.ModeSymlink != 0 {
			skipped = append(skipped, AuditNote{Path: rel, Reason: "symlink not followed"})
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() && ignoredAnalysisDir(d.Name()) {
			return filepath.SkipDir
		}
		paths = append(paths, rel)
		inventory.Entries = append(inventory.Entries, repositoryEntry{Path: rel, IsDir: d.IsDir()})
		return nil
	})
	if err == stop {
		err = nil
	}
	sort.Strings(paths)
	sort.Slice(inventory.Entries, func(i, j int) bool { return inventory.Entries[i].Path < inventory.Entries[j].Path })
	sortAuditNotes(skipped)
	sortAuditNotes(truncations)
	inventory.Paths = paths
	inventory.Skipped = skipped
	inventory.Truncations = truncations
	return inventory, err
}

func ignoredAnalysisDir(name string) bool {
	switch strings.ToLower(name) {
	case ".git", "node_modules", "vendor", "templates", "dist", "build", "out", "target", ".next", "coverage", ".cache", ".venv", "venv", ".tox", ".gradle", ".terraform", ".turbo", "generated":
		return true
	default:
		return false
	}
}

func canonicalRoot(root string) (string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	return filepath.Clean(canonical), nil
}

func safeReadRepositoryFile(root string, rel string, limit int64) (safeReadResult, error) {
	canonical, err := canonicalRoot(root)
	if err != nil {
		return safeReadResult{}, err
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return safeReadResult{}, fmt.Errorf("path escapes repository root")
	}
	target := filepath.Join(canonical, clean)
	if !withinRoot(canonical, target) {
		return safeReadResult{}, fmt.Errorf("path escapes repository root")
	}
	file, err := os.Open(target)
	if err != nil {
		return safeReadResult{}, err
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil || !openedInfo.Mode().IsRegular() {
		return safeReadResult{}, fmt.Errorf("not a regular repository file")
	}
	currentInfo, err := os.Lstat(target)
	if err != nil || currentInfo.Mode()&os.ModeSymlink != 0 || !currentInfo.Mode().IsRegular() || !os.SameFile(openedInfo, currentInfo) {
		return safeReadResult{}, fmt.Errorf("repository path changed during inspection")
	}
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil || !withinRoot(canonical, resolved) {
		return safeReadResult{}, fmt.Errorf("path escapes repository root")
	}
	resolvedInfo, err := os.Stat(resolved)
	if err != nil || !os.SameFile(openedInfo, resolvedInfo) {
		return safeReadResult{}, fmt.Errorf("repository path changed during inspection")
	}
	body, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return safeReadResult{}, err
	}
	truncated := int64(len(body)) > limit
	if truncated {
		body = body[:limit]
	}
	if bytes.IndexByte(body, 0) >= 0 || !utf8.Valid(body) {
		return safeReadResult{}, fmt.Errorf("binary or invalid UTF-8")
	}
	return safeReadResult{Body: body, Truncated: truncated}, nil
}

func withinRoot(root string, target string) bool {
	rel, err := filepath.Rel(root, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func validMinimalTOML(body []byte) bool {
	if len(bytes.TrimSpace(body)) == 0 {
		return false
	}
	arrayDepth := 0
	objectDepth := 0
	for _, raw := range strings.Split(string(body), "\n") {
		line := strings.TrimSpace(stripHashComment(raw))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") && arrayDepth == 0 && objectDepth == 0 {
			if !validTOMLHeader(line) {
				return false
			}
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			return false
		}
		quote := byte(0)
		escaped := false
		for i := 0; i < len(value); i++ {
			b := value[i]
			if quote != 0 {
				if escaped {
					escaped = false
				} else if b == '\\' && quote == '"' {
					escaped = true
				} else if b == quote {
					quote = 0
				}
				continue
			}
			if b == '"' || b == '\'' {
				quote = b
				continue
			}
			switch b {
			case '[':
				arrayDepth++
			case ']':
				arrayDepth--
			case '{':
				objectDepth++
			case '}':
				objectDepth--
			}
		}
		if quote != 0 || arrayDepth < 0 || objectDepth < 0 {
			return false
		}
	}
	return arrayDepth == 0 && objectDepth == 0
}

func validTOMLHeader(line string) bool {
	if strings.HasPrefix(line, "[[") {
		if !strings.HasSuffix(line, "]]") {
			return false
		}
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "[["), "]]"))
		return name != "" && !strings.ContainsAny(name, "[]")
	}
	if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
		return name != "" && !strings.ContainsAny(name, "[]")
	}
	return false
}

func stripHashComment(line string) string {
	inString := false
	for i, r := range line {
		if r == '"' {
			inString = !inString
		}
		if r == '#' && !inString {
			return line[:i]
		}
	}
	return line
}

func validJSONConfig(body []byte, jsonc bool) bool {
	if jsonc {
		var ok bool
		body, ok = stripJSONComments(body)
		if !ok {
			return false
		}
		body = stripTrailingJSONCommas(body)
	}
	var value map[string]json.RawMessage
	return json.Unmarshal(body, &value) == nil && value != nil
}

func stripJSONComments(body []byte) ([]byte, bool) {
	var out bytes.Buffer
	inString := false
	escaped := false
	for i := 0; i < len(body); i++ {
		b := body[i]
		if inString {
			out.WriteByte(b)
			if escaped {
				escaped = false
			} else if b == '\\' {
				escaped = true
			} else if b == '"' {
				inString = false
			}
			continue
		}
		if b == '"' {
			inString = true
			out.WriteByte(b)
			continue
		}
		if b == '/' && i+1 < len(body) && body[i+1] == '/' {
			for i < len(body) && body[i] != '\n' {
				i++
			}
			out.WriteByte('\n')
			continue
		}
		if b == '/' && i+1 < len(body) && body[i+1] == '*' {
			i += 2
			for i+1 < len(body) && !(body[i] == '*' && body[i+1] == '/') {
				i++
			}
			if i+1 >= len(body) {
				return nil, false
			}
			i++
			continue
		}
		out.WriteByte(b)
	}
	return out.Bytes(), !inString
}

func stripTrailingJSONCommas(body []byte) []byte {
	var out bytes.Buffer
	for i := 0; i < len(body); i++ {
		if body[i] == ',' {
			j := i + 1
			for j < len(body) && (body[j] == ' ' || body[j] == '\t' || body[j] == '\r' || body[j] == '\n') {
				j++
			}
			if j < len(body) && (body[j] == '}' || body[j] == ']') {
				continue
			}
		}
		out.WriteByte(body[i])
	}
	return out.Bytes()
}

func isFileUnder(rel string, dir string, ext string) bool {
	if !strings.HasPrefix(rel, dir+"/") || rel == dir {
		return false
	}
	if ext == "" {
		return filepath.Ext(rel) != ""
	}
	return strings.EqualFold(filepath.Ext(rel), ext)
}

func hasPathPrefix(paths []string, prefix string) bool {
	for _, path := range paths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

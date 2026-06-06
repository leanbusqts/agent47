package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Rule struct {
	File       string
	ID         string
	Topic      string
	Severity   string
	AppliesTo  string
	Rule       string
	Rationale  bool
	Refs       []string
	LineNumber int
}

var (
	idPattern       = regexp.MustCompile(`^((BE|FE|MO|GO|CLI|SCR|INF|DT|PL|MR|SHC|SHT)-[0-9]{3}|X-[a-z][a-z0-9-]*-[0-9]{3}|SEC-(global|py|js-ts|java-kotlin|csharp|swift|go|shell)-[0-9]{3})$`)
	topicPattern    = regexp.MustCompile(`^[a-z][a-z0-9-]*:[a-z][a-z0-9-]*$`)
	metadataPattern = regexp.MustCompile(`^metadata:$`)
)

func main() {
	if err := validateSchemaJSON(); err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}

	files, err := filepath.Glob("rules/*.yaml")
	if err != nil || len(files) == 0 {
		fmt.Fprintln(os.Stderr, "ERROR: no rules/*.yaml found; run from repo root")
		os.Exit(2)
	}
	sort.Strings(files)

	var errors []string
	all := map[string]Rule{}
	totalRules := 0

	for _, file := range files {
		rules, fileErrors := parseFile(file)
		errors = append(errors, fileErrors...)
		if len(rules) == 0 {
			errors = append(errors, fmt.Sprintf("%s: no rules found", file))
		}
		topics := map[string]string{}
		for _, rule := range rules {
			totalRules++
			if existing, ok := all[rule.ID]; ok {
				errors = append(errors, fmt.Sprintf("duplicate ID %s in %s and %s", rule.ID, existing.File, rule.File))
			}
			all[rule.ID] = rule
			if prev, ok := topics[rule.Topic]; ok {
				errors = append(errors, fmt.Sprintf("%s: duplicate topic %s also used by %s", rule.ID, rule.Topic, prev))
			}
			topics[rule.Topic] = rule.ID
			errors = append(errors, validateRule(rule)...)
		}
	}

	for _, rule := range all {
		for _, ref := range rule.Refs {
			if strings.Contains(ref, "#") || strings.HasSuffix(ref, ".md") {
				continue
			}
			if _, ok := all[ref]; !ok {
				errors = append(errors, fmt.Sprintf("%s: refs missing ID %s", rule.ID, ref))
			}
		}
	}

	errors = append(errors, driftErrors()...)

	if len(errors) > 0 {
		sort.Strings(errors)
		for _, err := range errors {
			fmt.Println("ERROR:", err)
		}
		os.Exit(1)
	}

	fmt.Printf("OK: %d rules across %d files validated.\n", totalRules, len(files))
}

func parseFile(path string) ([]Rule, []string) {
	f, err := os.Open(path)
	if err != nil {
		return nil, []string{fmt.Sprintf("%s: %v", path, err)}
	}
	defer f.Close()

	var errors []string
	var rules []Rule
	var current *Rule
	var inRationale bool
	var inRefs bool
	seenMetadata := false
	seenRules := false
	requiredMetadata := map[string]bool{
		"version": false, "last_updated": false, "owner": false, "review_cadence": false,
	}
	allowedRuleFields := map[string]bool{
		"id": true, "topic": true, "severity": true, "applies_to": true, "applies_when": true,
		"rule": true, "rationale": true, "how_to_apply": true, "verification": true, "examples": true,
		"refs": true, "tags": true, "superseded_by": true,
	}

	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		trim := strings.TrimSpace(line)
		if metadataPattern.MatchString(trim) {
			seenMetadata = true
			continue
		}
		if trim == "rules:" {
			seenRules = true
			continue
		}
		if seenMetadata && !seenRules && strings.HasPrefix(trim, "version:") {
			requiredMetadata["version"] = true
		}
		if seenMetadata && !seenRules && strings.HasPrefix(trim, "last_updated:") {
			requiredMetadata["last_updated"] = true
		}
		if seenMetadata && !seenRules && strings.HasPrefix(trim, "owner:") {
			requiredMetadata["owner"] = true
		}
		if seenMetadata && !seenRules && strings.HasPrefix(trim, "review_cadence:") {
			requiredMetadata["review_cadence"] = true
		}
		if strings.HasPrefix(line, "  - id: ") {
			if current != nil {
				rules = append(rules, *current)
			}
			current = &Rule{File: path, ID: quoted(line), LineNumber: lineNo}
			inRationale = false
			inRefs = false
			continue
		}
		if current == nil {
			continue
		}
		if strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "      ") {
			field := strings.TrimSpace(line)
			if strings.HasPrefix(field, "#") {
				continue
			}
			if idx := strings.Index(field, ":"); idx > 0 {
				field = field[:idx]
				if !allowedRuleFields[field] {
					errors = append(errors, fmt.Sprintf("%s:%d %s: unknown rule field %s", path, lineNo, current.ID, field))
				}
			}
		}
		switch {
		case strings.HasPrefix(line, "    topic: "):
			current.Topic = quoted(line)
			inRationale = false
			inRefs = false
		case strings.HasPrefix(line, "    severity: "):
			current.Severity = quotedOrBare(line)
			inRationale = false
			inRefs = false
		case strings.HasPrefix(line, "    applies_to: "):
			current.AppliesTo = strings.TrimSpace(strings.TrimPrefix(line, "    applies_to: "))
			inRationale = false
			inRefs = false
		case strings.HasPrefix(line, "    rule: "):
			current.Rule = quoted(line)
			inRationale = false
			inRefs = false
		case strings.HasPrefix(line, "    rationale:"):
			current.Rationale = true
			inRationale = true
			inRefs = false
		case strings.HasPrefix(line, "    refs:"):
			inRationale = false
			inRefs = true
			current.Refs = append(current.Refs, inlineList(line)...)
		case inRefs && strings.HasPrefix(line, "      - "):
			current.Refs = append(current.Refs, quotedOrBare(line))
		case strings.HasPrefix(line, "    "):
			if inRationale && strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "      ") {
				inRationale = false
			}
			if inRefs && !strings.HasPrefix(line, "      ") {
				inRefs = false
			}
		}
	}
	if current != nil {
		rules = append(rules, *current)
	}
	if err := scanner.Err(); err != nil {
		errors = append(errors, fmt.Sprintf("%s: %v", path, err))
	}
	if !seenMetadata {
		errors = append(errors, fmt.Sprintf("%s: missing metadata", path))
	}
	for key, ok := range requiredMetadata {
		if !ok {
			errors = append(errors, fmt.Sprintf("%s: missing metadata.%s", path, key))
		}
	}
	if !seenRules {
		errors = append(errors, fmt.Sprintf("%s: missing rules", path))
	}
	return rules, errors
}

func validateSchemaJSON() error {
	raw, err := os.ReadFile("rules/schema.json")
	if err != nil {
		return fmt.Errorf("rules/schema.json: %v", err)
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return fmt.Errorf("rules/schema.json: invalid JSON: %v", err)
	}
	return nil
}

func validateRule(rule Rule) []string {
	var errors []string
	prefix := fmt.Sprintf("%s:%d %s", rule.File, rule.LineNumber, rule.ID)
	if !idPattern.MatchString(rule.ID) {
		errors = append(errors, fmt.Sprintf("%s: invalid id", prefix))
	}
	if !topicPattern.MatchString(rule.Topic) {
		errors = append(errors, fmt.Sprintf("%s: invalid topic %q", prefix, rule.Topic))
	}
	switch rule.Severity {
	case "critical", "high":
		if !rule.Rationale {
			errors = append(errors, fmt.Sprintf("%s: missing rationale for %s severity", prefix, rule.Severity))
		}
	case "medium", "low":
	default:
		errors = append(errors, fmt.Sprintf("%s: invalid severity %q", prefix, rule.Severity))
	}
	if !strings.HasPrefix(rule.AppliesTo, "[") || !strings.HasSuffix(rule.AppliesTo, "]") {
		errors = append(errors, fmt.Sprintf("%s: applies_to must be a list", prefix))
	}
	if strings.TrimSpace(rule.Rule) == "" {
		errors = append(errors, fmt.Sprintf("%s: missing rule text", prefix))
	}
	return errors
}

func driftErrors() []string {
	var errors []string
	base := []string{"security-global", "security-shell", "security-csharp", "security-java-kotlin", "security-js-ts", "security-py", "security-swift", "security-go", "rules-backend", "rules-frontend", "rules-mobile", "rules-cross", "rules-go"}
	for _, name := range base {
		a := filepath.Join("rules", name+".yaml")
		b := filepath.Join("templates", "base", "rules", name+".yaml")
		if !sameFile(a, b) {
			errors = append(errors, fmt.Sprintf("base drift: %s != %s", a, b))
		}
	}
	bundles := map[string]string{
		"project-backend":          "rules-backend",
		"project-cli":              "rules-cli",
		"project-desktop":          "rules-desktop",
		"project-frontend":         "rules-frontend",
		"project-infra":            "rules-infra",
		"project-mobile":           "rules-mobile",
		"project-monorepo-tooling": "rules-monorepo-tooling",
		"project-plugin":           "rules-plugin",
		"project-scripts":          "rules-scripts",
		"shared-cli-behavior":      "shared-cli-behavior",
		"shared-testing":           "shared-testing",
	}
	for bundle, name := range bundles {
		a := filepath.Join("rules", name+".yaml")
		b := filepath.Join("templates", "bundles", bundle, "rules", name+".yaml")
		if !sameFile(a, b) {
			errors = append(errors, fmt.Sprintf("bundle drift: %s != %s", a, b))
		}
	}
	return errors
}

func sameFile(a, b string) bool {
	aa, errA := os.ReadFile(a)
	bb, errB := os.ReadFile(b)
	return errA == nil && errB == nil && string(aa) == string(bb)
}

func quoted(line string) string {
	start := strings.Index(line, `"`)
	end := strings.LastIndex(line, `"`)
	if start >= 0 && end > start {
		return line[start+1 : end]
	}
	return strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
}

func quotedOrBare(line string) string {
	if strings.Contains(line, `"`) {
		return quoted(line)
	}
	parts := strings.SplitN(line, ":", 2)
	value := parts[len(parts)-1]
	value = strings.TrimSpace(strings.TrimPrefix(value, "-"))
	return strings.Trim(value, ` "'`)
}

func inlineList(line string) []string {
	start := strings.Index(line, "[")
	end := strings.LastIndex(line, "]")
	if start < 0 || end <= start {
		return nil
	}
	var out []string
	for _, part := range strings.Split(line[start+1:end], ",") {
		ref := strings.Trim(part, ` "'`)
		if ref != "" {
			out = append(out, ref)
		}
	}
	return out
}

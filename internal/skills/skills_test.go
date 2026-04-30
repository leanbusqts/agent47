package skills

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leanbusqts/agent47/internal/templates"
)

func TestDiscoverFromFilesystemTemplates(t *testing.T) {
	t.Parallel()

	source := templates.NewFilesystemSource(filepath.Join(repoRoot(t), "templates"))
	service := Service{}

	discovered, err := service.Discover(source, "base/skills")
	if err != nil {
		t.Fatal(err)
	}

	if len(discovered) == 0 {
		t.Fatal("expected discovered skills")
	}

	if discovered[0].Location != "base/skills/analyze/SKILL.md" {
		t.Fatalf("expected sorted first skill, got %s", discovered[0].Location)
	}
}

func TestGenerateAvailableSkillsXML(t *testing.T) {
	t.Parallel()

	service := Service{}
	data, err := service.GenerateAvailableSkillsXML([]Skill{
		{
			Name:          "analyze",
			Description:   "Understand the current state.",
			Compatibility: "Designed for skills-compatible coding agents.",
			Metadata: Metadata{
				"category": {"analysis"},
				"tags":     {"discovery", "debugging"},
			},
			Location: "skills/analyze/SKILL.md",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(data, []byte("<available_skills>")) {
		t.Fatalf("expected XML root, got %s", string(data))
	}
	if !bytes.Contains(data, []byte("<name>analyze</name>")) {
		t.Fatalf("expected skill XML, got %s", string(data))
	}
	if !bytes.Contains(data, []byte("<compatibility>Designed for skills-compatible coding agents.</compatibility>")) {
		t.Fatalf("expected compatibility XML, got %s", string(data))
	}
	if !bytes.Contains(data, []byte("<entry key=\"category\">")) {
		t.Fatalf("expected metadata XML, got %s", string(data))
	}
}

func TestGenerateAvailableSkillsJSON(t *testing.T) {
	t.Parallel()

	service := Service{}
	data, err := service.GenerateAvailableSkillsJSON([]Skill{
		{
			Name:          "analyze",
			Description:   "Understand the current state.",
			Compatibility: "Designed for skills-compatible coding agents.",
			Metadata: Metadata{
				"category":      []string{"analysis"},
				"stack_signals": []string{"go", "cli"},
			},
			Location: "skills/analyze/SKILL.md",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(data, []byte(`"skills"`)) {
		t.Fatalf("expected skills JSON root, got %s", string(data))
	}
	if !bytes.Contains(data, []byte(`"stack_signals"`)) {
		t.Fatalf("expected snake_case metadata key, got %s", string(data))
	}
	if !bytes.Contains(data, []byte(`"location": "skills/analyze/SKILL.md"`)) {
		t.Fatalf("expected location JSON, got %s", string(data))
	}
}

func TestGenerateAvailableSkillsSummaryMarkdown(t *testing.T) {
	t.Parallel()

	service := Service{}
	data, err := service.GenerateAvailableSkillsSummaryMarkdown([]Skill{
		{
			Name:          "analyze",
			Description:   "Understand the current state.",
			Compatibility: "Designed for skills-compatible coding agents.",
			Metadata: Metadata{
				"category":    []string{"analysis"},
				"repo_shapes": []string{"app", "monorepo"},
			},
			Location: "skills/analyze/SKILL.md",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(data, []byte("# Available Skills")) {
		t.Fatalf("expected summary title, got %s", string(data))
	}
	if !bytes.Contains(data, []byte("## analyze")) {
		t.Fatalf("expected skill section, got %s", string(data))
	}
	if !bytes.Contains(data, []byte("repo_shapes=app, monorepo")) {
		t.Fatalf("expected formatted metadata, got %s", string(data))
	}
}

func TestValidateRejectsInvalidFrontmatter(t *testing.T) {
	t.Parallel()

	_, err := Validate("skills/bad/SKILL.md", []byte("not-frontmatter"))
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestDiscoverSkipsDirectoriesWithoutSkillFile(t *testing.T) {
	root := t.TempDir()
	mustWriteSkillFile(t, filepath.Join(root, "skills", "alpha", "SKILL.md"), "---\nname: alpha\ndescription: Alpha skill.\ncompatibility: Designed for skills-compatible coding agents.\nmetadata:\n  category: workflow\n---\n")
	if err := os.MkdirAll(filepath.Join(root, "skills", "empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	discovered, err := Service{}.Discover(templates.NewFilesystemSource(root), "skills")
	if err != nil {
		t.Fatal(err)
	}
	if len(discovered) != 1 || discovered[0].Name != "alpha" {
		t.Fatalf("unexpected discovered skills: %+v", discovered)
	}
	if discovered[0].Compatibility == "" {
		t.Fatal("expected discovered compatibility")
	}
	if got := discovered[0].Metadata["category"]; len(got) != 1 || got[0] != "workflow" {
		t.Fatalf("unexpected discovered metadata: %#v", discovered[0].Metadata)
	}
}

func TestDiscoverErrorsWhenNoValidSkillTemplatesFound(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "skills", "empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := Service{}.Discover(templates.NewFilesystemSource(root), "skills")
	if err == nil {
		t.Fatal("expected no valid skills error")
	}
}

func TestIsNotExistRecognizesFilesystemErrors(t *testing.T) {
	if !isNotExist(fs.ErrNotExist) {
		t.Fatal("expected fs.ErrNotExist to match")
	}
	if isNotExist(errors.New("different error")) {
		t.Fatal("did not expect unrelated error to match")
	}
}

func TestValidateRejectsMissingFieldsAndInvalidNames(t *testing.T) {
	cases := []string{
		"---\nname: \ndescription: desc\n---\n",
		"---\nname: Not-Kebab\ndescription: desc\n---\n",
		"---\nname: valid-name\ndescription: " + strings.Repeat("x", 141) + "\n---\n",
	}
	for _, body := range cases {
		if _, err := Validate("skills/bad/SKILL.md", []byte(body)); err == nil {
			t.Fatalf("expected validation error for body %q", body)
		}
	}
}

func TestParseFrontmatterSupportsCompatibilityAndMetadata(t *testing.T) {
	t.Parallel()

	fm, err := ParseFrontmatter([]byte(`---
name: test-skill
description: Validate metadata parsing.
compatibility: Designed for skills-compatible coding agents.
metadata:
  category: testing
  tags: [qa, regression]
  applies-to: [frontend, backend]
  repo_shapes: [app, monorepo]
---
`))
	if err != nil {
		t.Fatal(err)
	}

	if fm.Compatibility == "" {
		t.Fatal("expected compatibility to be parsed")
	}
	if got := fm.Metadata["category"]; len(got) != 1 || got[0] != "testing" {
		t.Fatalf("unexpected category metadata: %#v", got)
	}
	if got := fm.Metadata["tags"]; len(got) != 2 || got[0] != "qa" || got[1] != "regression" {
		t.Fatalf("unexpected tags metadata: %#v", got)
	}
	if got := fm.Metadata["repo_shapes"]; len(got) != 2 || got[0] != "app" || got[1] != "monorepo" {
		t.Fatalf("unexpected repo_shapes metadata: %#v", got)
	}
}

func TestValidateRejectsInvalidMetadata(t *testing.T) {
	t.Parallel()

	cases := []string{
		"---\nname: valid-name\ndescription: desc\nmetadata:\n  invalid key: value\n---\n",
		"---\nname: valid-name\ndescription: desc\nmetadata:\n  category:\n---\n",
		"---\nname: valid-name\ndescription: desc\ncompatibility: " + strings.Repeat("x", 141) + "\n---\n",
	}
	for _, body := range cases {
		if _, err := Validate("skills/bad/SKILL.md", []byte(body)); err == nil {
			t.Fatalf("expected validation error for body %q", body)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func mustWriteSkillFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

package skills

import (
	"bytes"
	"errors"
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

	discovered, err := service.Discover(source, "skills")
	if err != nil {
		t.Fatal(err)
	}

	if len(discovered) == 0 {
		t.Fatal("expected discovered skills")
	}

	if discovered[0].Location != "skills/analyze/SKILL.md" {
		t.Fatalf("expected sorted first skill, got %s", discovered[0].Location)
	}
}

func TestGenerateAvailableSkillsXML(t *testing.T) {
	t.Parallel()

	service := Service{}
	data, err := service.GenerateAvailableSkillsXML([]Skill{
		{
			Name:        "analyze",
			Description: "Understand the current state.",
			Location:    "skills/analyze/SKILL.md",
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
	mustWriteSkillFile(t, filepath.Join(root, "skills", "alpha", "SKILL.md"), "---\nname: alpha\ndescription: Alpha skill.\n---\n")
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
	if !isNotExist(errors.New("open skill: no such file")) {
		t.Fatal("expected no such file error to match")
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

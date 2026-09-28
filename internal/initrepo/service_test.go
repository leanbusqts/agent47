package initrepo

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/leanbusqts/agent47/internal/resolve"
	internalruntime "github.com/leanbusqts/agent47/internal/runtime"
	"github.com/leanbusqts/agent47/internal/templates"
)

func TestRunForceReplacesLegacyCLIContentAndPreservesUnrelatedFiles(t *testing.T) {
	root := t.TempDir()
	mustWriteInitFile(t, filepath.Join(root, agentsFile), "old agents\n")
	mustWriteInitFile(t, filepath.Join(root, "rules", "rules-cli.yaml"), "old cli\n")
	mustWriteInitFile(t, filepath.Join(root, "rules", "custom.yaml"), "custom\n")
	mustWriteInitFile(t, filepath.Join(root, ".agents", "state.txt"), "state\n")
	mustWriteInitFile(t, filepath.Join(root, ".agents", "specs", "spec.yml"), "legacy task\n")
	mustWriteInitFile(t, filepath.Join(root, "specs", "spec.yml"), "older task\n")
	mustWriteInitFile(t, filepath.Join(root, "skills", "custom", "SKILL.md"), "skill\n")
	mustWriteInitFile(t, filepath.Join(root, "prompts", "custom.txt"), "prompt\n")
	mustWriteInitFile(t, filepath.Join(root, "README.md"), "readme\n")

	service := testService(map[string]string{
		agentsFile:                   "new agents\n",
		"rules/rules-cli.yaml":       "new cli\n",
		"rules/security-global.yaml": "security\n",
	})
	set := resolve.InstallSet{Rules: []string{"security-global.yaml", "rules-cli.yaml"}}

	plan, err := service.Plan(root, set, true)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"rules/security-global.yaml"}; !reflect.DeepEqual(plan.Create, want) {
		t.Fatalf("unexpected create plan: got %v want %v", plan.Create, want)
	}
	if want := []string{"AGENTS.md", "rules/rules-cli.yaml"}; !reflect.DeepEqual(plan.Update, want) {
		t.Fatalf("unexpected update plan: got %v want %v", plan.Update, want)
	}
	if want := []string{".agents/specs/spec.yml", "prompts/", "rules/custom.yaml", "skills/", "specs/spec.yml"}; !reflect.DeepEqual(plan.Remove, want) {
		t.Fatalf("unexpected remove plan: got %v want %v", plan.Remove, want)
	}

	if err := service.Run(context.Background(), Options{Force: true, WorkDir: root, InstallSet: set}); err != nil {
		t.Fatal(err)
	}

	assertInitFile(t, filepath.Join(root, agentsFile), "new agents\n")
	assertInitFile(t, filepath.Join(root, "rules", "rules-cli.yaml"), "new cli\n")
	assertInitFile(t, filepath.Join(root, "rules", "security-global.yaml"), "security\n")
	assertInitFile(t, filepath.Join(root, ".agents", "state.txt"), "state\n")
	assertInitFile(t, filepath.Join(root, "README.md"), "readme\n")
	for _, path := range []string{
		filepath.Join(root, "rules", "custom.yaml"),
		filepath.Join(root, "skills"),
		filepath.Join(root, "prompts"),
		filepath.Join(root, "specs", "spec.yml"),
		filepath.Join(root, ".agents", "specs", "spec.yml"),
	} {
		assertInitNotExists(t, path)
	}
}

func TestRunKeepsExistingManagedFilesWithoutForce(t *testing.T) {
	root := t.TempDir()
	mustWriteInitFile(t, filepath.Join(root, agentsFile), "project agents\n")
	mustWriteInitFile(t, filepath.Join(root, "rules", "rules-cli.yaml"), "project cli\n")

	service := testService(map[string]string{
		agentsFile:             "template agents\n",
		"rules/rules-cli.yaml": "template cli\n",
	})
	set := resolve.InstallSet{Rules: []string{"rules-cli.yaml"}}
	plan, err := service.Plan(root, set, false)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"AGENTS.md", "rules/rules-cli.yaml"}; !reflect.DeepEqual(plan.Keep, want) {
		t.Fatalf("unexpected keep plan: got %v want %v", plan.Keep, want)
	}
	if err := service.Run(context.Background(), Options{WorkDir: root, InstallSet: set}); err != nil {
		t.Fatal(err)
	}
	assertInitFile(t, filepath.Join(root, agentsFile), "project agents\n")
	assertInitFile(t, filepath.Join(root, "rules", "rules-cli.yaml"), "project cli\n")
}

func TestRunRollsBackAllCommittedFilesWhenLaterRenameFails(t *testing.T) {
	root := t.TempDir()
	mustWriteInitFile(t, filepath.Join(root, agentsFile), "old agents\n")
	mustWriteInitFile(t, filepath.Join(root, "rules", "rules-cli.yaml"), "old cli\n")
	mustWriteInitFile(t, filepath.Join(root, "skills", "legacy", "SKILL.md"), "old skill\n")

	service := testService(map[string]string{
		agentsFile:             "new agents\n",
		"rules/rules-cli.yaml": "new cli\n",
	})
	service.beforeMutation = func(operation, path string) error {
		if operation == mutationInstall && path == "rules/rules-cli.yaml" {
			return errors.New("injected rename failure")
		}
		return nil
	}

	err := service.Run(context.Background(), Options{
		Force:      true,
		WorkDir:    root,
		InstallSet: resolve.InstallSet{Rules: []string{"rules-cli.yaml"}},
	})
	if err == nil || !strings.Contains(err.Error(), "injected rename failure") {
		t.Fatalf("expected injected failure, got %v", err)
	}
	assertInitFile(t, filepath.Join(root, agentsFile), "old agents\n")
	assertInitFile(t, filepath.Join(root, "rules", "rules-cli.yaml"), "old cli\n")
	assertInitFile(t, filepath.Join(root, "skills", "legacy", "SKILL.md"), "old skill\n")
}

func TestRunRollsBackCommittedFilesWhenContextIsCanceled(t *testing.T) {
	root := t.TempDir()
	mustWriteInitFile(t, filepath.Join(root, agentsFile), "old agents\n")
	mustWriteInitFile(t, filepath.Join(root, "rules", "rules-cli.yaml"), "old cli\n")

	ctx, cancel := context.WithCancel(context.Background())
	service := testService(map[string]string{
		agentsFile:             "new agents\n",
		"rules/rules-cli.yaml": "new cli\n",
	})
	service.beforeMutation = func(operation, path string) error {
		if operation == mutationInstall && path == agentsFile {
			cancel()
		}
		return nil
	}

	err := service.Run(ctx, Options{
		Force:      true,
		WorkDir:    root,
		InstallSet: resolve.InstallSet{Rules: []string{"rules-cli.yaml"}},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	assertInitFile(t, filepath.Join(root, agentsFile), "old agents\n")
	assertInitFile(t, filepath.Join(root, "rules", "rules-cli.yaml"), "old cli\n")
	assertNoInitTemps(t, root)
}

func TestRunRestoresLegacyCleanupWhenContextIsCanceledDuringStaging(t *testing.T) {
	root := t.TempDir()
	mustWriteInitFile(t, filepath.Join(root, "skills", "legacy", "SKILL.md"), "legacy skill\n")
	ctx, cancel := context.WithCancel(context.Background())
	service := testService(map[string]string{agentsFile: "agents\n"})
	service.beforeMutation = func(operation, path string) error {
		if operation == mutationBackup && path == "skills/" {
			cancel()
		}
		return nil
	}

	err := service.Run(ctx, Options{Force: true, WorkDir: root})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	assertInitFile(t, filepath.Join(root, "skills", "legacy", "SKILL.md"), "legacy skill\n")
	assertInitNotExists(t, filepath.Join(root, agentsFile))
	assertNoInitTemps(t, root)
}

func TestRunDoesNotOverwriteConcurrentCreationWithoutForce(t *testing.T) {
	root := t.TempDir()
	service := testService(map[string]string{agentsFile: "template agents\n"})
	service.beforeMutation = func(operation, path string) error {
		if operation == mutationInstall && path == agentsFile {
			mustWriteInitFile(t, filepath.Join(root, agentsFile), "concurrent creation\n")
		}
		return nil
	}

	err := service.Run(context.Background(), Options{WorkDir: root})
	if err == nil || !strings.Contains(err.Error(), "changed concurrently") {
		t.Fatalf("expected concurrent creation error, got %v", err)
	}
	assertInitFile(t, filepath.Join(root, agentsFile), "concurrent creation\n")
	assertNoInitTemps(t, root)
}

func TestRunForceRestoresConcurrentEditDetectedAfterBackup(t *testing.T) {
	root := t.TempDir()
	mustWriteInitFile(t, filepath.Join(root, agentsFile), "original\n")
	service := testService(map[string]string{agentsFile: "template agents\n"})
	service.beforeMutation = func(operation, path string) error {
		if operation == mutationInstall && path == agentsFile {
			if err := os.WriteFile(filepath.Join(root, agentsFile), []byte("concurrent edit\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return nil
	}

	err := service.Run(context.Background(), Options{Force: true, WorkDir: root})
	if err == nil || !strings.Contains(err.Error(), "changed concurrently") {
		t.Fatalf("expected concurrent edit error, got %v", err)
	}
	assertInitFile(t, filepath.Join(root, agentsFile), "concurrent edit\n")
	assertNoInitTemps(t, root)
}

func TestRollbackLeavesConcurrentEditUntouchedAndRetainsRecoveryBackup(t *testing.T) {
	root := t.TempDir()
	mustWriteInitFile(t, filepath.Join(root, agentsFile), "original agents\n")
	mustWriteInitFile(t, filepath.Join(root, "rules", "rules-cli.yaml"), "original cli\n")
	service := testService(map[string]string{
		agentsFile:             "new agents\n",
		"rules/rules-cli.yaml": "new cli\n",
	})
	service.beforeMutation = func(operation, path string) error {
		switch {
		case operation == mutationInstall && path == "rules/rules-cli.yaml":
			return errors.New("injected later failure")
		case operation == mutationRestore && path == agentsFile:
			if err := os.WriteFile(filepath.Join(root, agentsFile), []byte("user edit during rollback\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return nil
	}

	err := service.Run(context.Background(), Options{
		Force:      true,
		WorkDir:    root,
		InstallSet: resolve.InstallSet{Rules: []string{"rules-cli.yaml"}},
	})
	if err == nil || !strings.Contains(err.Error(), "rollback incomplete") || !strings.Contains(err.Error(), "recovery files retained") {
		t.Fatalf("expected rollback conflict with recovery path, got %v", err)
	}
	assertInitFile(t, filepath.Join(root, agentsFile), "user edit during rollback\n")
	assertInitFile(t, filepath.Join(root, "rules", "rules-cli.yaml"), "original cli\n")
	if matches, globErr := filepath.Glob(filepath.Join(root, ".agent47-init-*-backup-AGENTS.md")); globErr != nil || len(matches) != 1 {
		t.Fatalf("expected retained original backup, matches=%v err=%v", matches, globErr)
	}
}

func TestRunRejectsRulesDirectorySwapWithoutWritingOutsideRepository(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("descriptor-relative confinement is only available on Darwin and Linux")
	}
	root := t.TempDir()
	outside := t.TempDir()
	mustWriteInitFile(t, filepath.Join(root, agentsFile), "old agents\n")
	mustWriteInitFile(t, filepath.Join(root, "rules", "rules-cli.yaml"), "old cli\n")
	mustWriteInitFile(t, filepath.Join(outside, "rules-cli.yaml"), "outside sentinel\n")
	service := testService(map[string]string{
		agentsFile:             "new agents\n",
		"rules/rules-cli.yaml": "new cli\n",
	})
	swapped := false
	service.beforeMutation = func(operation, path string) error {
		if !swapped && operation == mutationBackup && path == "rules/" {
			swapped = true
			if err := os.Rename(filepath.Join(root, "rules"), filepath.Join(root, "rules-original")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, filepath.Join(root, "rules")); err != nil {
				t.Fatal(err)
			}
		}
		return nil
	}

	err := service.Run(context.Background(), Options{
		Force:      true,
		WorkDir:    root,
		InstallSet: resolve.InstallSet{Rules: []string{"rules-cli.yaml"}},
	})
	if err == nil || !strings.Contains(err.Error(), "managed target changed concurrently: rules/") {
		t.Fatalf("expected rules swap error, got %v", err)
	}
	assertInitFile(t, filepath.Join(root, agentsFile), "old agents\n")
	assertInitFile(t, filepath.Join(root, "rules-original", "rules-cli.yaml"), "old cli\n")
	assertInitFile(t, filepath.Join(outside, "rules-cli.yaml"), "outside sentinel\n")
}

func TestPlanValidatesAssemblyButAllowsResolveAuthorizedDynamicRule(t *testing.T) {
	repo := t.TempDir()
	baseManifest := `[rule_templates]
rules-cross.yaml

[managed_targets]
AGENTS.md
rules/rules-cross.yaml

[generated_targets]
.agent47/context.md

[preserved_targets]
README.md

[force_cleanup_targets]
rules/
skills/
prompts/
specs/spec.yml
.agents/specs/spec.yml

[required_template_files]
AGENTS.md
manifest.txt

[required_template_dirs]
rules
`
	mustWriteInitFile(t, filepath.Join(repo, "templates", "manifest.txt"), baseManifest)
	mustWriteInitFile(t, filepath.Join(repo, "templates", "base", "manifest.txt"), baseManifest)
	mustWriteInitFile(t, filepath.Join(repo, "templates", "base", agentsFile), "agents\n")
	mustWriteInitFile(t, filepath.Join(repo, "templates", "base", "rules", "rules-cross.yaml"), "cross\n")
	mustWriteInitFile(t, filepath.Join(repo, "templates", "base", "rules", "security-go.yaml"), "go security\n")

	service, err := New(runtimeConfigForTemplates(repo))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := service.Plan(t.TempDir(), resolve.InstallSet{Bundles: []string{"base"}, Rules: []string{"security-go.yaml"}}, false)
	if err != nil {
		t.Fatalf("dynamic resolve-authorized rule should be accepted: %v", err)
	}
	want := []string{"AGENTS.md", "rules/security-go.yaml"}
	if !reflect.DeepEqual(plan.Create, want) {
		t.Fatalf("unexpected plan: got %v want %v", plan.Create, want)
	}

	mustWriteInitFile(t, filepath.Join(repo, "templates", "bundles", "broken", "manifest.txt"), "")
	_, err = service.Plan(t.TempDir(), resolve.InstallSet{Bundles: []string{"base", "broken"}}, false)
	if err == nil || !strings.Contains(err.Error(), "invalid bundle manifest") {
		t.Fatalf("expected invalid selected assembly, got %v", err)
	}
}

func TestRunRollsBackGeneratedContextWithPolicyFailure(t *testing.T) {
	service := testService(map[string]string{
		agentsFile:             "agents\n",
		"rules/rules-cli.yaml": "cli rule\n",
	})
	root := t.TempDir()
	service.beforeMutation = func(operation, path string) error {
		if operation == mutationInstall && path == agentsFile {
			return errors.New("injected policy failure")
		}
		return nil
	}

	err := service.Run(context.Background(), Options{
		WorkDir:    root,
		InstallSet: resolve.InstallSet{Rules: []string{"rules-cli.yaml"}},
		Context:    mustBuildContext(t, root).Content,
	})
	if err == nil {
		t.Fatal("expected injected failure")
	}
	for _, path := range []string{agentsFile, "rules", ".agent47"} {
		if _, statErr := os.Stat(filepath.Join(root, path)); !os.IsNotExist(statErr) {
			t.Fatalf("expected %s to be rolled back, got %v", path, statErr)
		}
	}
}

func TestPlanRejectsSymlinkedManagedDirectoryAndUnsafeRuleNames(t *testing.T) {
	service := testService(map[string]string{agentsFile: "agents\n"})

	for _, name := range []string{"../escape.yaml", `..\\escape.yaml`, "not-yaml.txt", "bad name.yaml"} {
		t.Run(name, func(t *testing.T) {
			_, err := service.Plan(t.TempDir(), resolve.InstallSet{Rules: []string{name}}, false)
			if err == nil || !strings.Contains(err.Error(), "unsafe rule template name") {
				t.Fatalf("expected unsafe rule name error for %q, got %v", name, err)
			}
		})
	}

	typeRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(typeRoot, agentsFile), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := service.Plan(typeRoot, resolve.InstallSet{}, false)
	if err == nil || !strings.Contains(err.Error(), "must be a regular file") {
		t.Fatalf("expected target type rejection, got %v", err)
	}

	if runtime.GOOS == "windows" {
		return
	}
	replaceRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(replaceRoot, "rules", "rules-cli.yaml"), 0o755); err != nil {
		t.Fatal(err)
	}
	replaceService := testService(map[string]string{
		agentsFile:             "agents\n",
		"rules/rules-cli.yaml": "cli\n",
	})
	if err := replaceService.Run(context.Background(), Options{Force: true, WorkDir: replaceRoot, InstallSet: resolve.InstallSet{Rules: []string{"rules-cli.yaml"}}}); err != nil {
		t.Fatalf("force should replace malformed legacy rule entries: %v", err)
	}
	assertInitFile(t, filepath.Join(replaceRoot, "rules", "rules-cli.yaml"), "cli\n")

	specCollisionRoot := t.TempDir()
	mustWriteInitFile(t, filepath.Join(specCollisionRoot, "specs", "spec.yml", "keep.txt"), "keep\n")
	err = service.Run(context.Background(), Options{Force: true, WorkDir: specCollisionRoot})
	if err == nil || !strings.Contains(err.Error(), "legacy cleanup file target must not be a directory") {
		t.Fatalf("expected legacy task-spec directory collision, got %v", err)
	}
	assertInitFile(t, filepath.Join(specCollisionRoot, "specs", "spec.yml", "keep.txt"), "keep\n")

	symlinkCleanupRoot := t.TempDir()
	symlinkTarget := t.TempDir()
	mustWriteInitFile(t, filepath.Join(symlinkTarget, "sentinel.txt"), "outside\n")
	if err := os.Symlink(symlinkTarget, filepath.Join(symlinkCleanupRoot, "skills")); err != nil {
		t.Fatal(err)
	}
	if err := service.Run(context.Background(), Options{Force: true, WorkDir: symlinkCleanupRoot}); err != nil {
		t.Fatalf("force should remove a legacy namespace symlink without following it: %v", err)
	}
	assertInitNotExists(t, filepath.Join(symlinkCleanupRoot, "skills"))
	assertInitFile(t, filepath.Join(symlinkTarget, "sentinel.txt"), "outside\n")

	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "rules")); err != nil {
		t.Fatal(err)
	}
	_, err = service.Plan(root, resolve.InstallSet{}, false)
	if err == nil || !strings.Contains(err.Error(), "must not be a symlink") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
}

func testService(files map[string]string) *Service {
	source := memorySource{files: make(map[string][]byte, len(files))}
	for path, body := range files {
		source.files[path] = []byte(body)
	}
	return &Service{
		sourceFor: func(resolve.InstallSet) templates.Source { return source },
		openRoot:  openConfinedRoot,
	}
}

type memorySource struct {
	files map[string][]byte
}

func (s memorySource) ReadFile(path string) ([]byte, error) {
	data, ok := s.files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return append([]byte{}, data...), nil
}

func (memorySource) ReadDir(string) ([]fs.DirEntry, error) { return nil, fs.ErrNotExist }
func (memorySource) Stat(string) (fs.FileInfo, error)      { return nil, fs.ErrNotExist }

func mustWriteInitFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertInitFile(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("unexpected content for %s: got %q want %q", path, data, want)
	}
}

func assertInitNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected path to be absent: %s (err=%v)", path, err)
	}
}

func assertNoInitTemps(t *testing.T, roots ...string) {
	t.Helper()
	for _, root := range roots {
		var matches []string
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if strings.HasPrefix(entry.Name(), ".agent47-init-") || strings.HasPrefix(entry.Name(), ".agent47-force-") {
				matches = append(matches, path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		sort.Strings(matches)
		if len(matches) != 0 {
			t.Fatalf("unexpected init transaction files under %s: %v", root, matches)
		}
	}
}

func runtimeConfigForTemplates(repo string) internalruntime.Config {
	return internalruntime.Config{TemplateMode: internalruntime.TemplateModeFilesystem, RepoRoot: repo}
}

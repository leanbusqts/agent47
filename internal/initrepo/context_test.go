package initrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/leanbusqts/agent47/internal/analyze"
	"github.com/leanbusqts/agent47/internal/contextmap"
)

func TestContextServiceCreatesKeepsAndUpdatesGeneratedContext(t *testing.T) {
	root := t.TempDir()
	mustWriteInitFile(t, filepath.Join(root, "go.mod"), "module example.com/project\n")
	first := mustBuildContext(t, root)
	service := NewContextService()

	action, err := service.Run(context.Background(), ContextOptions{WorkDir: root, Content: first.Content})
	if err != nil || action != contextmap.ActionCreate {
		t.Fatalf("expected create, action=%q err=%v", action, err)
	}
	assertInitFile(t, filepath.Join(root, contextmap.TargetPath), string(first.Content))
	before, err := os.Stat(filepath.Join(root, contextmap.TargetPath))
	if err != nil {
		t.Fatal(err)
	}

	action, err = service.Run(context.Background(), ContextOptions{WorkDir: root, Content: first.Content})
	if err != nil || action != contextmap.ActionCurrent {
		t.Fatalf("expected current, action=%q err=%v", action, err)
	}
	after, err := os.Stat(filepath.Join(root, contextmap.TargetPath))
	if err != nil {
		t.Fatal(err)
	}
	if !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("current context was rewritten")
	}

	mustWriteInitFile(t, filepath.Join(root, "internal", "service.go"), "package internal\n")
	second := mustBuildContext(t, root)
	action, err = service.Run(context.Background(), ContextOptions{WorkDir: root, Content: second.Content})
	if err != nil || action != contextmap.ActionUpdate {
		t.Fatalf("expected update, action=%q err=%v", action, err)
	}
	assertInitFile(t, filepath.Join(root, contextmap.TargetPath), string(second.Content))
}

func TestContextServiceRequiresForceForManualOrUnknownContent(t *testing.T) {
	root := t.TempDir()
	mustWriteInitFile(t, filepath.Join(root, "go.mod"), "module example.com/project\n")
	document := mustBuildContext(t, root)
	service := NewContextService()
	if _, err := service.Run(context.Background(), ContextOptions{WorkDir: root, Content: document.Content}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, contextmap.TargetPath)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("manual edit\n"); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := service.Run(context.Background(), ContextOptions{WorkDir: root, Content: document.Content}); !errors.Is(err, contextmap.ErrModifiedContext) {
		t.Fatalf("expected manual edit protection, got %v", err)
	}
	action, err := service.Run(context.Background(), ContextOptions{WorkDir: root, Content: document.Content, Force: true})
	if err != nil || action != contextmap.ActionUpdate {
		t.Fatalf("expected forced update, action=%q err=%v", action, err)
	}
	assertInitFile(t, path, string(document.Content))

	mustWriteInitFile(t, path, "user-owned context\n")
	if _, err := service.Run(context.Background(), ContextOptions{WorkDir: root, Content: document.Content}); !errors.Is(err, contextmap.ErrUnknownContext) {
		t.Fatalf("expected unknown context protection, got %v", err)
	}
}

func TestContextServiceRejectsSymlinkedTargets(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	document := mustBuildContext(t, root)
	if err := os.Symlink(outside, filepath.Join(root, ".agent47")); err != nil {
		t.Fatal(err)
	}
	if _, err := NewContextService().Run(context.Background(), ContextOptions{WorkDir: root, Content: document.Content, Force: true}); err == nil {
		t.Fatal("expected symlinked .agent47 to be rejected")
	}
	if _, err := os.Stat(filepath.Join(outside, "context.md")); !os.IsNotExist(err) {
		t.Fatal("symlink target was modified")
	}

	root = t.TempDir()
	document = mustBuildContext(t, root)
	if err := os.Mkdir(filepath.Join(root, ".agent47"), 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(outside, "sentinel.md")
	if err := os.WriteFile(sentinel, []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sentinel, filepath.Join(root, contextmap.TargetPath)); err != nil {
		t.Fatal(err)
	}
	if _, err := NewContextService().Run(context.Background(), ContextOptions{WorkDir: root, Content: document.Content, Force: true}); err == nil {
		t.Fatal("expected symlinked context.md to be rejected")
	}
	assertInitFile(t, sentinel, "outside\n")
}

func TestContextServiceRollsBackCreatedDirectoryOnFailure(t *testing.T) {
	root := t.TempDir()
	document := mustBuildContext(t, root)
	service := NewContextService()
	service.service.beforeMutation = func(operation, path string) error {
		if operation == mutationInstall && path == contextmap.TargetPath {
			return errors.New("injected failure")
		}
		return nil
	}
	if _, err := service.Run(context.Background(), ContextOptions{WorkDir: root, Content: document.Content}); err == nil {
		t.Fatal("expected injected failure")
	}
	if _, err := os.Stat(filepath.Join(root, ".agent47")); !os.IsNotExist(err) {
		t.Fatalf("expected created directory rollback, got %v", err)
	}
}

func mustBuildContext(t *testing.T, root string) contextmap.Document {
	t.Helper()
	document, err := contextmap.Build(context.Background(), root, analyze.AnalysisResult{RepoShape: "single-project"}, contextmap.BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return document
}

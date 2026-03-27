package fsx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBasicFilesystemOperations(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := Service{}
	dir := filepath.Join(root, "nested", "dir")
	path := filepath.Join(dir, "file.txt")

	if err := svc.MkdirAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := svc.WriteFileAtomic(path, []byte("hello"), 0o640); err != nil {
		t.Fatal(err)
	}
	data, err := svc.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected file data: %s", string(data))
	}
	if !svc.Exists(path) {
		t.Fatalf("expected file to exist: %s", path)
	}
	if !svc.IsDir(dir) {
		t.Fatalf("expected dir to exist: %s", dir)
	}

	renamed := filepath.Join(dir, "renamed.txt")
	if err := svc.Rename(path, renamed); err != nil {
		t.Fatal(err)
	}
	if err := svc.Remove(renamed); err != nil {
		t.Fatal(err)
	}
	if svc.Exists(renamed) {
		t.Fatalf("expected file to be removed: %s", renamed)
	}

	if err := svc.RemoveAll(filepath.Join(root, "nested")); err != nil {
		t.Fatal(err)
	}
	if svc.Exists(dir) {
		t.Fatalf("expected dir to be removed: %s", dir)
	}
}

func TestWriteFileAtomicHonorsInjectedFailure(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "fail.txt")
	t.Setenv("AGENT47_ENABLE_TEST_HOOKS", "true")
	t.Setenv("AGENT47_FAIL_WRITE_TARGET", target)

	svc := Service{}
	if err := svc.WriteFileAtomic(target, []byte("data"), 0o644); err == nil {
		t.Fatal("expected injected write failure")
	}
	if svc.Exists(target) {
		t.Fatalf("did not expect failed target to exist: %s", target)
	}
}

func TestWriteFileAtomicOverwritesAndPreservesMode(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "file.txt")
	svc := Service{}
	if err := svc.WriteFileAtomic(target, []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := svc.WriteFileAtomic(target, []byte("second"), 0o640); err != nil {
		t.Fatal(err)
	}
	assertFileContains(t, target, "second")
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("unexpected mode: %v", info.Mode().Perm())
	}
}

func TestWriteFileAtomicReplacesExistingFileContents(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "file.txt")
	svc := Service{}
	mustWriteFile(t, target, "old\n")

	if err := svc.WriteFileAtomic(target, []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	assertFileContains(t, target, "new")
}

func TestCopyFileAndCopyDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := Service{}

	srcFile := filepath.Join(root, "src.txt")
	dstFile := filepath.Join(root, "dst", "copied.txt")
	mustWriteFile(t, srcFile, "copy me\n")
	if err := svc.CopyFile(srcFile, dstFile); err != nil {
		t.Fatal(err)
	}
	assertFileContains(t, dstFile, "copy me")

	srcDir := filepath.Join(root, "srcdir")
	dstDir := filepath.Join(root, "dstdir")
	mustWriteFile(t, filepath.Join(srcDir, "a.txt"), "a\n")
	mustWriteFile(t, filepath.Join(srcDir, "nested", "b.txt"), "b\n")
	if err := svc.CopyDir(srcDir, dstDir); err != nil {
		t.Fatal(err)
	}
	assertFileContains(t, filepath.Join(dstDir, "a.txt"), "a")
	assertFileContains(t, filepath.Join(dstDir, "nested", "b.txt"), "b")
}

func TestCopyFilePreservesSourceMode(t *testing.T) {
	root := t.TempDir()
	svc := Service{}
	srcFile := filepath.Join(root, "src.txt")
	dstFile := filepath.Join(root, "dst.txt")
	if err := os.WriteFile(srcFile, []byte("copy"), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := svc.CopyFile(srcFile, dstFile); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dstFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o750 {
		t.Fatalf("unexpected mode: %v", info.Mode().Perm())
	}
}

func TestCopyFileRejectsDirectorySource(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := Service{}
	srcDir := filepath.Join(root, "srcdir")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := svc.CopyFile(srcDir, filepath.Join(root, "dst.txt")); err == nil {
		t.Fatal("expected directory copy failure")
	}
}

func TestCopyDirRejectsFileSource(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := Service{}
	srcFile := filepath.Join(root, "src.txt")
	mustWriteFile(t, srcFile, "x\n")

	if err := svc.CopyDir(srcFile, filepath.Join(root, "dst")); err == nil {
		t.Fatal("expected non-directory copy failure")
	}
}

func TestCopyFileFailsWhenSourceMissing(t *testing.T) {
	svc := Service{}
	if err := svc.CopyFile(filepath.Join(t.TempDir(), "missing"), filepath.Join(t.TempDir(), "dst")); err == nil {
		t.Fatal("expected missing source error")
	}
}

func TestCopyFileFailsWhenDestinationIsDirectory(t *testing.T) {
	root := t.TempDir()
	svc := Service{}
	srcFile := filepath.Join(root, "src.txt")
	dstDir := filepath.Join(root, "dst")
	mustWriteFile(t, srcFile, "copy me\n")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := svc.CopyFile(srcFile, dstDir); err == nil {
		t.Fatal("expected destination directory error")
	}
}

func TestCopyDirFailsWhenSourceMissing(t *testing.T) {
	svc := Service{}
	if err := svc.CopyDir(filepath.Join(t.TempDir(), "missing"), filepath.Join(t.TempDir(), "dst")); err == nil {
		t.Fatal("expected missing source error")
	}
}

func TestCopyFileHonorsInjectedFailure(t *testing.T) {
	root := t.TempDir()
	svc := Service{}
	srcFile := filepath.Join(root, "src.txt")
	dstFile := filepath.Join(root, "dst.txt")
	mustWriteFile(t, srcFile, "copy me\n")
	t.Setenv("AGENT47_ENABLE_TEST_HOOKS", "true")
	t.Setenv("AGENT47_FAIL_COPY_TARGET", dstFile)

	if err := svc.CopyFile(srcFile, dstFile); err == nil {
		t.Fatal("expected injected copy failure")
	}
}

func TestReplaceDirAtomicRejectsExistingTargetWithoutForce(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := Service{}
	target := filepath.Join(root, "target")
	stage := filepath.Join(root, "stage")
	mustWriteFile(t, filepath.Join(target, "old.txt"), "old\n")
	mustWriteFile(t, filepath.Join(stage, "new.txt"), "new\n")

	if _, err := svc.ReplaceDirAtomic(stage, target, false); err == nil {
		t.Fatal("expected existing target failure")
	}
}

func TestReplaceDirAtomicMovesStageWhenTargetMissing(t *testing.T) {
	root := t.TempDir()
	svc := Service{}
	target := filepath.Join(root, "target")
	stage := filepath.Join(root, "stage")
	mustWriteFile(t, filepath.Join(stage, "new.txt"), "new\n")

	result, err := svc.ReplaceDirAtomic(stage, target, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.BackupPath != "" {
		t.Fatalf("did not expect backup path, got %s", result.BackupPath)
	}
	assertFileContains(t, filepath.Join(target, "new.txt"), "new")
}

func TestSymlinkAtomicCreatesLinkOnSuccess(t *testing.T) {
	root := t.TempDir()
	svc := Service{}
	target := filepath.Join(root, "target.txt")
	link := filepath.Join(root, "link.txt")
	mustWriteFile(t, target, "target\n")

	if err := svc.SymlinkAtomic(target, link); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(link)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := filepath.EvalSymlinks(target)
	if err != nil {
		expected = target
	}
	if resolved != expected {
		t.Fatalf("expected link to point to %s, got %s", target, resolved)
	}
}

func TestSymlinkAtomicReplacesExistingLinkOnSuccess(t *testing.T) {
	root := t.TempDir()
	svc := Service{}
	oldTarget := filepath.Join(root, "old.txt")
	newTarget := filepath.Join(root, "new.txt")
	link := filepath.Join(root, "link.txt")
	mustWriteFile(t, oldTarget, "old\n")
	mustWriteFile(t, newTarget, "new\n")
	if err := os.Symlink(oldTarget, link); err != nil {
		t.Fatal(err)
	}

	if err := svc.SymlinkAtomic(newTarget, link); err != nil {
		t.Fatal(err)
	}

	resolved, err := filepath.EvalSymlinks(link)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := filepath.EvalSymlinks(newTarget)
	if err != nil {
		expected = newTarget
	}
	if resolved != expected {
		t.Fatalf("expected link to point to %s, got %s", newTarget, resolved)
	}
}

func TestTestHookFailDirSwapMarkerOnlyFailsOnce(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	marker := filepath.Join(root, "marker")
	t.Setenv("AGENT47_ENABLE_TEST_HOOKS", "true")
	t.Setenv("AGENT47_FAIL_DIR_SWAP_TARGET", target)
	t.Setenv("AGENT47_FAIL_DIR_SWAP_MARKER", marker)

	if !testHookFailDirSwap(target) {
		t.Fatal("expected first call to fail")
	}
	if testHookFailDirSwap(target) {
		t.Fatal("expected second call to stop failing after marker exists")
	}
}

func TestReplaceDirAtomicCreatesBackupOnForce(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "templates")
	stage := filepath.Join(root, ".templates-stage")
	mustWriteFile(t, filepath.Join(target, "AGENTS.md"), "old template\n")
	mustWriteFile(t, filepath.Join(stage, "AGENTS.md"), "new template\n")

	svc := Service{}
	result, err := svc.ReplaceDirAtomic(stage, target, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.BackupPath == "" {
		t.Fatal("expected backup path")
	}
	assertFileContains(t, filepath.Join(target, "AGENTS.md"), "new template")
	assertFileContains(t, filepath.Join(result.BackupPath, "AGENTS.md"), "old template")
}

func TestReplaceDirAtomicRemovesPreviousBackupsOnForce(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "templates")
	stage := filepath.Join(root, ".templates-stage")
	oldBackup := filepath.Join(root, "templates.bak.old")
	mustWriteFile(t, filepath.Join(target, "AGENTS.md"), "old template\n")
	mustWriteFile(t, filepath.Join(stage, "AGENTS.md"), "new template\n")
	mustWriteFile(t, filepath.Join(oldBackup, "stale.txt"), "stale\n")

	svc := Service{}
	result, err := svc.ReplaceDirAtomic(stage, target, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.BackupPath == oldBackup {
		t.Fatal("expected a fresh backup path")
	}
	if _, err := os.Stat(oldBackup); !os.IsNotExist(err) {
		t.Fatalf("expected previous backup removal, err=%v", err)
	}
}

func TestReplaceDirAtomicRestoresBackupOnInjectedFailure(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "templates")
	stage := filepath.Join(root, ".templates-stage")
	marker := filepath.Join(root, "marker")
	mustWriteFile(t, filepath.Join(target, "AGENTS.md"), "old template\n")
	mustWriteFile(t, filepath.Join(stage, "AGENTS.md"), "new template\n")

	t.Setenv("AGENT47_ENABLE_TEST_HOOKS", "true")
	t.Setenv("AGENT47_FAIL_DIR_SWAP_TARGET", target)
	t.Setenv("AGENT47_FAIL_DIR_SWAP_MARKER", marker)

	svc := Service{}
	_, err := svc.ReplaceDirAtomic(stage, target, true)
	if err == nil {
		t.Fatal("expected injected failure")
	}
	assertFileContains(t, filepath.Join(target, "AGENTS.md"), "old template")
}

func TestSymlinkAtomicPreservesExistingLinkOnInjectedFailure(t *testing.T) {
	root := t.TempDir()
	linkPath := filepath.Join(root, "afs")
	oldTarget := filepath.Join(root, "old-afs")
	newTarget := filepath.Join(root, "new-afs")
	mustWriteFile(t, oldTarget, "old\n")
	mustWriteFile(t, newTarget, "new\n")
	if err := os.Symlink(oldTarget, linkPath); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AGENT47_ENABLE_TEST_HOOKS", "true")
	t.Setenv("AGENT47_FAIL_SYMLINK_TARGET", linkPath)

	svc := Service{}
	if err := svc.SymlinkAtomic(newTarget, linkPath); err == nil {
		t.Fatal("expected injected failure")
	}

	resolved, err := filepath.EvalSymlinks(linkPath)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := filepath.EvalSymlinks(oldTarget)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != expected {
		t.Fatalf("expected old link target, got %s", resolved)
	}
}

func mustWriteFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertFileContains(t *testing.T, path string, fragment string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), fragment) {
		t.Fatalf("expected %s to contain %q, got %s", path, fragment, string(data))
	}
}

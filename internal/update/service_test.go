package update

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func TestCacheRoundTripPreservesSpecialCharacters(t *testing.T) {
	baseDir := t.TempDir()
	cfg := runtime.Config{
		Version:         "1.2.3",
		UpdateCacheFile: filepath.Join(baseDir, "cache", "update.cache"),
	}

	service := New(cli.NewOutput(ioDiscard{}, ioDiscard{}))
	record := CacheRecord{
		CheckedAt:     time.Now(),
		Status:        "error",
		Method:        `remote "quoted"`,
		LocalVersion:  "1.2.3",
		LatestVersion: "2.0.0",
		Message:       `problem with "quotes" and \ slashes`,
	}

	service.saveCache(cfg, record)
	loaded, ok := service.loadCache(cfg)
	if !ok {
		t.Fatal("expected cache record to load")
	}

	if loaded.Status != record.Status {
		t.Fatalf("expected status %q, got %q", record.Status, loaded.Status)
	}
	if loaded.Method != record.Method {
		t.Fatalf("expected method %q, got %q", record.Method, loaded.Method)
	}
	if loaded.LocalVersion != record.LocalVersion {
		t.Fatalf("expected local version %q, got %q", record.LocalVersion, loaded.LocalVersion)
	}
	if loaded.LatestVersion != record.LatestVersion {
		t.Fatalf("expected latest version %q, got %q", record.LatestVersion, loaded.LatestVersion)
	}
	if loaded.Message != record.Message {
		t.Fatalf("expected message %q, got %q", record.Message, loaded.Message)
	}
}

func TestGitCheckWarnsWhenUpstreamIsNotConfigured(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	repoRoot := t.TempDir()
	runGitCommand(t, repoRoot, "init")
	runGitCommand(t, repoRoot, "config", "user.name", "agent47-test")
	runGitCommand(t, repoRoot, "config", "user.email", "agent47@example.com")
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitCommand(t, repoRoot, "add", "README.md")
	runGitCommand(t, repoRoot, "commit", "-m", "initial")

	var stdout bytes.Buffer
	service := New(cli.NewOutput(&stdout, ioDiscard{}))
	cfg := runtime.Config{
		Version:         "vtest",
		RepoRoot:        repoRoot,
		UpdateCacheFile: filepath.Join(t.TempDir(), "update.cache"),
	}

	if err := service.Check(context.Background(), cfg, CheckOptions{Force: true}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !strings.Contains(stdout.String(), "Cannot check for updates: git upstream not configured") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestGitCheckHonorsCanceledContext(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	repoRoot := newTrackedRepo(t)
	var stdout bytes.Buffer
	service := New(cli.NewOutput(&stdout, ioDiscard{}))
	cfg := runtime.Config{
		Version:         "vtest",
		RepoRoot:        repoRoot,
		UpdateCacheFile: filepath.Join(t.TempDir(), "update.cache"),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := service.Check(ctx, cfg, CheckOptions{Force: true}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !strings.Contains(stdout.String(), "git update check canceled") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestGitCheckDetectsBehindUpstream(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	baseDir := t.TempDir()
	origin := filepath.Join(baseDir, "origin.git")
	seed := filepath.Join(baseDir, "seed")
	clone := filepath.Join(baseDir, "clone")

	runGitCommand(t, baseDir, "init", "--bare", origin)
	runGitCommand(t, baseDir, "clone", origin, seed)
	runGitCommand(t, seed, "config", "user.name", "agent47-test")
	runGitCommand(t, seed, "config", "user.email", "agent47@example.com")
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitCommand(t, seed, "add", "README.md")
	runGitCommand(t, seed, "commit", "-m", "initial")
	runGitCommand(t, seed, "branch", "-M", "main")
	runGitCommand(t, seed, "push", "-u", "origin", "main")

	runGitCommand(t, baseDir, "clone", origin, clone)
	runGitCommand(t, clone, "branch", "--set-upstream-to=origin/main", "main")

	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitCommand(t, seed, "commit", "-am", "update")
	runGitCommand(t, seed, "push", "origin", "main")
	var stdout bytes.Buffer
	service := New(cli.NewOutput(&stdout, ioDiscard{}))
	cfg := runtime.Config{
		Version:         "vtest",
		RepoRoot:        clone,
		UpdateCacheFile: filepath.Join(t.TempDir(), "update.cache"),
	}

	if err := service.Check(context.Background(), cfg, CheckOptions{Force: true}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Update available from git upstream") {
		t.Fatalf("expected git upstream update message, got %s", output)
	}
	if !strings.Contains(output, "after git fetch") {
		t.Fatalf("expected forced git fetch marker, got %s", output)
	}
}

func TestGitCheckDetectsTrackingCurrent(t *testing.T) {
	repoRoot := newTrackedRepo(t)

	var stdout bytes.Buffer
	service := New(cli.NewOutput(&stdout, ioDiscard{}))
	cfg := runtime.Config{
		Version:         "vtest",
		RepoRoot:        repoRoot,
		UpdateCacheFile: filepath.Join(t.TempDir(), "update.cache"),
	}

	if err := service.Check(context.Background(), cfg, CheckOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Git tracking reference is current") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestGitCheckDetectsLocalAhead(t *testing.T) {
	repoRoot := newTrackedRepo(t)
	mustWriteRepoFile(t, filepath.Join(repoRoot, "README.md"), "local change\n")
	runGitCommand(t, repoRoot, "commit", "-am", "local change")

	var stdout bytes.Buffer
	service := New(cli.NewOutput(&stdout, ioDiscard{}))
	cfg := runtime.Config{
		Version:         "vtest",
		RepoRoot:        repoRoot,
		UpdateCacheFile: filepath.Join(t.TempDir(), "update.cache"),
	}

	if err := service.Check(context.Background(), cfg, CheckOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Local copy is ahead of git upstream") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestGitCheckDetectsDivergedHistory(t *testing.T) {
	baseDir := t.TempDir()
	origin := filepath.Join(baseDir, "origin.git")
	seed := filepath.Join(baseDir, "seed")
	clone := filepath.Join(baseDir, "clone")

	runGitCommand(t, baseDir, "init", "--bare", origin)
	runGitCommand(t, baseDir, "clone", origin, seed)
	runGitCommand(t, seed, "config", "user.name", "agent47-test")
	runGitCommand(t, seed, "config", "user.email", "agent47@example.com")
	mustWriteRepoFile(t, filepath.Join(seed, "README.md"), "v1\n")
	runGitCommand(t, seed, "add", "README.md")
	runGitCommand(t, seed, "commit", "-m", "initial")
	runGitCommand(t, seed, "branch", "-M", "main")
	runGitCommand(t, seed, "push", "-u", "origin", "main")

	runGitCommand(t, baseDir, "clone", origin, clone)
	runGitCommand(t, clone, "branch", "--set-upstream-to=origin/main", "main")

	mustWriteRepoFile(t, filepath.Join(seed, "README.md"), "remote change\n")
	runGitCommand(t, seed, "commit", "-am", "remote change")
	runGitCommand(t, seed, "push", "origin", "main")

	mustWriteRepoFile(t, filepath.Join(clone, "README.md"), "local change\n")
	runGitCommand(t, clone, "commit", "-am", "local change")

	var stdout bytes.Buffer
	service := New(cli.NewOutput(&stdout, ioDiscard{}))
	cfg := runtime.Config{
		Version:         "vtest",
		RepoRoot:        clone,
		UpdateCacheFile: filepath.Join(t.TempDir(), "update.cache"),
	}

	if err := service.Check(context.Background(), cfg, CheckOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "diverged") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestLoadCacheRejectsFutureTimestamp(t *testing.T) {
	cfg := runtime.Config{
		Version:         "1.2.3",
		UpdateCacheFile: filepath.Join(t.TempDir(), "cache", "update.cache"),
	}
	service := New(cli.NewOutput(ioDiscard{}, ioDiscard{}))
	service.saveCache(cfg, CacheRecord{
		CheckedAt:    time.Now().Add(2 * time.Hour),
		LocalVersion: "1.2.3",
		Status:       "up-to-date",
	})

	if _, ok := service.loadCache(cfg); ok {
		t.Fatal("expected cache miss for future timestamp")
	}
}

func TestLoadCacheRejectsLegacyRemoteCacheWithoutSource(t *testing.T) {
	cfg := runtime.Config{
		Version:         "1.2.3",
		UpdateCacheFile: filepath.Join(t.TempDir(), "cache", "update.cache"),
	}
	t.Setenv("AGENT47_VERSION_URL", "https://example.com/VERSION")

	service := New(cli.NewOutput(ioDiscard{}, ioDiscard{}))
	service.saveCache(cfg, CacheRecord{
		CheckedAt:     time.Now(),
		LocalVersion:  "1.2.3",
		LatestVersion: "2.0.0",
		Status:        "update-available",
	})

	data, err := os.ReadFile(cfg.UpdateCacheFile)
	if err != nil {
		t.Fatal(err)
	}
	var legacy CacheRecord
	if err := json.Unmarshal(data, &legacy); err != nil {
		t.Fatal(err)
	}
	legacy.Source = ""
	legacyData, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg.UpdateCacheFile, legacyData, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, ok := service.loadCache(cfg); ok {
		t.Fatal("expected cache miss for legacy remote cache without source")
	}
}

func TestRemoteCheckRejectsInvalidURL(t *testing.T) {
	service := New(cli.NewOutput(ioDiscard{}, ioDiscard{}))
	rec := CacheRecord{LocalVersion: "1.2.3"}
	if err := service.remoteCheck(context.Background(), "http://[::1", &rec); err == nil {
		t.Fatal("expected remote check failure")
	}
	if rec.Status != "error" {
		t.Fatalf("expected error status, got %s", rec.Status)
	}
}

func TestRemoteCheckRejectsEmptyFileVersion(t *testing.T) {
	versionFile := filepath.Join(t.TempDir(), "VERSION")
	if err := os.WriteFile(versionFile, []byte(" \n "), 0o644); err != nil {
		t.Fatal(err)
	}
	service := New(cli.NewOutput(ioDiscard{}, ioDiscard{}))
	rec := CacheRecord{LocalVersion: "1.2.3"}
	err := service.remoteCheck(context.Background(), "file://"+versionFile, &rec)
	if err == nil {
		t.Fatal("expected remote check failure for empty body")
	}
	if rec.Message != "empty VERSION response" {
		t.Fatalf("unexpected message: %s", rec.Message)
	}
}

func TestRemoteCheckHandlesHTTPSuccess(t *testing.T) {
	service := New(cli.NewOutput(ioDiscard{}, ioDiscard{}))
	service.HTTPClient = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("2.0.0\n")),
			}, nil
		}),
	}

	rec := CacheRecord{LocalVersion: "1.2.3"}
	if err := service.remoteCheck(context.Background(), "https://example.com/VERSION", &rec); err != nil {
		t.Fatalf("unexpected remote check error: %v", err)
	}
	if rec.Status != "update-available" || rec.LatestVersion != "2.0.0" {
		t.Fatalf("unexpected record: %+v", rec)
	}
}

func TestRemoteCheckHandlesHTTPStatusFailure(t *testing.T) {
	service := New(cli.NewOutput(ioDiscard{}, ioDiscard{}))
	service.HTTPClient = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       io.NopCloser(strings.NewReader("boom")),
			}, nil
		}),
	}

	rec := CacheRecord{LocalVersion: "1.2.3"}
	if err := service.remoteCheck(context.Background(), "https://example.com/VERSION", &rec); err == nil {
		t.Fatal("expected remote check failure")
	}
	if rec.Status != "error" {
		t.Fatalf("expected error status, got %s", rec.Status)
	}
}

func TestRemoteCheckHandlesBodyReadFailure(t *testing.T) {
	service := New(cli.NewOutput(ioDiscard{}, ioDiscard{}))
	service.HTTPClient = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       errReadCloser{},
			}, nil
		}),
	}

	rec := CacheRecord{LocalVersion: "1.2.3"}
	if err := service.remoteCheck(context.Background(), "https://example.com/VERSION", &rec); err == nil {
		t.Fatal("expected body read failure")
	}
	if rec.Status != "error" {
		t.Fatalf("expected error status, got %s", rec.Status)
	}
}

func TestCheckUsesValidCache(t *testing.T) {
	var stdout bytes.Buffer
	service := New(cli.NewOutput(&stdout, ioDiscard{}))
	cfg := runtime.Config{
		Version:         "1.2.3",
		UpdateCacheFile: filepath.Join(t.TempDir(), "cache", "update.cache"),
	}
	service.saveCache(cfg, CacheRecord{
		CheckedAt:     time.Now(),
		Status:        "up-to-date",
		LocalVersion:  cfg.Version,
		LatestVersion: cfg.Version,
		Message:       "cached",
	})

	if err := service.Check(context.Background(), cfg, CheckOptions{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Using cached update check") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestCheckReportsNoUpdateSourceAvailable(t *testing.T) {
	var stdout bytes.Buffer
	service := New(cli.NewOutput(&stdout, ioDiscard{}))
	cfg := runtime.Config{
		Version:         "1.2.3",
		UpdateCacheFile: filepath.Join(t.TempDir(), "cache", "update.cache"),
	}

	if err := service.Check(context.Background(), cfg, CheckOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "no update source available") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestFinishRemoteMarksUpToDate(t *testing.T) {
	service := New(cli.NewOutput(ioDiscard{}, ioDiscard{}))
	rec := CacheRecord{LocalVersion: "1.2.3", LatestVersion: "1.2.3"}
	if err := service.finishRemote(&rec); err != nil {
		t.Fatal(err)
	}
	if rec.Status != "up-to-date" {
		t.Fatalf("unexpected status: %+v", rec)
	}
}

func TestFinishRemoteDoesNotRecommendDowngradeWhenRemoteIsOlder(t *testing.T) {
	service := New(cli.NewOutput(ioDiscard{}, ioDiscard{}))
	rec := CacheRecord{LocalVersion: "1.2.3", LatestVersion: "1.2.2"}
	if err := service.finishRemote(&rec); err != nil {
		t.Fatal(err)
	}
	if rec.Status != "version-differs" {
		t.Fatalf("unexpected status: %+v", rec)
	}
}

func TestCompareVersionHandlesSemverLikeValues(t *testing.T) {
	cases := []struct {
		local      string
		remote     string
		wantCmp    int
		comparable bool
	}{
		{local: "1.2.3", remote: "1.2.4", wantCmp: -1, comparable: true},
		{local: "v1.2.3", remote: "1.2.3", wantCmp: 0, comparable: true},
		{local: "2.0", remote: "1.9.9", wantCmp: 1, comparable: true},
		{local: "dev", remote: "1.0.0", comparable: false},
	}

	for _, tc := range cases {
		gotCmp, gotComparable := compareVersion(tc.local, tc.remote)
		if gotComparable != tc.comparable {
			t.Fatalf("unexpected comparable result for %s vs %s", tc.local, tc.remote)
		}
		if gotComparable && gotCmp != tc.wantCmp {
			t.Fatalf("unexpected compare result for %s vs %s: got %d want %d", tc.local, tc.remote, gotCmp, tc.wantCmp)
		}
	}
}

func TestParseAheadBehindHandlesValidAndInvalidOutputs(t *testing.T) {
	cases := []struct {
		name    string
		input   []byte
		wantErr bool
		ahead   int
		behind  int
	}{
		{name: "valid", input: []byte("3\t2"), ahead: 3, behind: 2},
		{name: "missing-column", input: []byte("3"), wantErr: true},
		{name: "invalid-int", input: []byte("x\t2"), wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ahead, behind, err := parseAheadBehind(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if ahead != tc.ahead || behind != tc.behind {
				t.Fatalf("unexpected ahead/behind: %d/%d", ahead, behind)
			}
		})
	}
}

func TestTrackingMessageCoversStates(t *testing.T) {
	cases := []struct {
		ahead  int
		behind int
		fetch  bool
		want   string
	}{
		{ahead: 0, behind: 0, fetch: false, want: "local checkout matches origin/main; remote fetch not performed"},
		{ahead: 2, behind: 0, fetch: false, want: "local checkout is ahead of origin/main by 2 commit(s); remote fetch not performed"},
		{ahead: 0, behind: 3, fetch: true, want: "origin/main is ahead by 3 commit(s) after git fetch"},
		{ahead: 1, behind: 1, fetch: false, want: "local checkout and origin/main have diverged"},
	}

	for _, tc := range cases {
		got := trackingMessage("origin/main", tc.ahead, tc.behind, tc.fetch)
		if !strings.Contains(got, tc.want) {
			t.Fatalf("unexpected message %q for ahead=%d behind=%d", got, tc.ahead, tc.behind)
		}
	}
}

func TestPrintCoversStatuses(t *testing.T) {
	cases := []struct {
		name   string
		record CacheRecord
		want   string
	}{
		{name: "up-to-date", record: CacheRecord{Status: "up-to-date", LatestVersion: "1.2.3"}, want: "Up to date"},
		{name: "update", record: CacheRecord{Status: "update-available", LatestVersion: "2.0.0"}, want: "Update available"},
		{name: "tracking-current", record: CacheRecord{Status: "git-tracking-current", Message: "tracking current"}, want: "tracking current"},
		{name: "local-ahead", record: CacheRecord{Status: "local-ahead", Message: "ahead"}, want: "ahead"},
		{name: "git-behind", record: CacheRecord{Status: "git-behind", Message: "behind"}, want: "behind"},
		{name: "git-diverged", record: CacheRecord{Status: "git-diverged", Message: "diverged"}, want: "diverged"},
		{name: "error", record: CacheRecord{Status: "error", Message: "boom"}, want: "boom"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			service := New(cli.NewOutput(&stdout, ioDiscard{}))
			service.print(tc.record, runtime.Config{RepoRoot: "/tmp/repo"})
			if !strings.Contains(stdout.String(), tc.want) {
				t.Fatalf("unexpected output: %s", stdout.String())
			}
		})
	}
}

func TestHTTPClientUsesDefaultWhenUnset(t *testing.T) {
	service := New(cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if service.httpClient() == nil {
		t.Fatal("expected default http client")
	}
	custom := &http.Client{}
	service.HTTPClient = custom
	if service.httpClient() != custom {
		t.Fatal("expected injected client to be returned")
	}
}

func TestLoadCacheRejectsInvalidJSON(t *testing.T) {
	cfg := runtime.Config{
		Version:         "1.2.3",
		UpdateCacheFile: filepath.Join(t.TempDir(), "cache", "update.cache"),
	}
	if err := os.MkdirAll(filepath.Dir(cfg.UpdateCacheFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg.UpdateCacheFile, []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := New(cli.NewOutput(ioDiscard{}, ioDiscard{}))
	if _, ok := service.loadCache(cfg); ok {
		t.Fatal("expected invalid cache JSON to be rejected")
	}
}

func TestLoadCacheRejectsUnexpectedSourceWhenNoRemoteConfigured(t *testing.T) {
	cfg := runtime.Config{
		Version:         "1.2.3",
		UpdateCacheFile: filepath.Join(t.TempDir(), "cache", "update.cache"),
	}
	service := New(cli.NewOutput(ioDiscard{}, ioDiscard{}))
	service.saveCache(cfg, CacheRecord{
		CheckedAt:    time.Now(),
		LocalVersion: cfg.Version,
		Status:       "up-to-date",
		Source:       "remote:https://example.com/VERSION",
	})

	if _, ok := service.loadCache(cfg); ok {
		t.Fatal("expected source mismatch cache miss")
	}
}

func TestSaveCacheCreatesParentDirectory(t *testing.T) {
	cfg := runtime.Config{
		Version:         "1.2.3",
		UpdateCacheFile: filepath.Join(t.TempDir(), "nested", "cache", "update.cache"),
	}
	service := New(cli.NewOutput(ioDiscard{}, ioDiscard{}))
	service.saveCache(cfg, CacheRecord{
		CheckedAt:    time.Now(),
		LocalVersion: cfg.Version,
		Status:       "up-to-date",
	})

	if _, err := os.Stat(cfg.UpdateCacheFile); err != nil {
		t.Fatalf("expected cache file to be written: %v", err)
	}
}

func TestGitCheckFailsOutsideGitCheckout(t *testing.T) {
	service := New(cli.NewOutput(ioDiscard{}, ioDiscard{}))
	rec := CacheRecord{LocalVersion: "1.2.3"}
	err := service.gitCheck(context.Background(), runtime.Config{RepoRoot: t.TempDir()}, &rec, false)
	if err == nil {
		t.Fatal("expected git checkout error")
	}
	if rec.Message != "agent47 not installed from a git checkout" {
		t.Fatalf("unexpected message: %s", rec.Message)
	}
}

func newTrackedRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	baseDir := t.TempDir()
	origin := filepath.Join(baseDir, "origin.git")
	seed := filepath.Join(baseDir, "seed")
	clone := filepath.Join(baseDir, "clone")

	runGitCommand(t, baseDir, "init", "--bare", origin)
	runGitCommand(t, baseDir, "clone", origin, seed)
	runGitCommand(t, seed, "config", "user.name", "agent47-test")
	runGitCommand(t, seed, "config", "user.email", "agent47@example.com")
	mustWriteRepoFile(t, filepath.Join(seed, "README.md"), "v1\n")
	runGitCommand(t, seed, "add", "README.md")
	runGitCommand(t, seed, "commit", "-m", "initial")
	runGitCommand(t, seed, "branch", "-M", "main")
	runGitCommand(t, seed, "push", "-u", "origin", "main")
	runGitCommand(t, baseDir, "clone", origin, clone)
	runGitCommand(t, clone, "branch", "--set-upstream-to=origin/main", "main")
	return clone
}

func mustWriteRepoFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (errReadCloser) Close() error             { return nil }

func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
}

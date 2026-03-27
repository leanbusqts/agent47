package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/fsx"
	"github.com/leanbusqts/agent47/internal/install"
	"github.com/leanbusqts/agent47/internal/runtime"
)

const cacheTTL = 24 * time.Hour

type Service struct {
	FS         fsx.Service
	Out        cli.Output
	HTTPClient *http.Client
}

type CheckOptions struct {
	Force bool
}

type CacheRecord struct {
	CheckedAt     time.Time `json:"checked_at"`
	Status        string    `json:"status"`
	Method        string    `json:"method"`
	Source        string    `json:"source,omitempty"`
	LocalVersion  string    `json:"local_version"`
	LatestVersion string    `json:"latest_version"`
	Message       string    `json:"message"`
}

func New(out cli.Output) *Service {
	return &Service{FS: fsx.Service{}, Out: out}
}

func (s *Service) Check(ctx context.Context, cfg runtime.Config, opts CheckOptions) error {
	if !opts.Force {
		if rec, ok := s.loadCache(cfg); ok {
			s.Out.Info("Using cached update check (age %dh)", int(time.Since(rec.CheckedAt).Hours()))
			s.print(rec, cfg)
			return nil
		}
	}

	rec := CacheRecord{
		CheckedAt:    time.Now(),
		Source:       sourceKey(cfg),
		LocalVersion: cfg.Version,
	}

	if url := os.Getenv("AGENT47_VERSION_URL"); url != "" {
		if err := s.remoteCheck(ctx, url, &rec); err == nil {
			s.saveCache(cfg, rec)
			s.print(rec, cfg)
			return nil
		}
	}

	if cfg.RepoRoot != "" {
		if err := s.gitCheck(ctx, cfg, &rec, opts.Force); err == nil {
			s.print(rec, cfg)
			return nil
		}
	}

	if rec.Status == "" {
		rec.Status = "error"
		rec.Message = "no update source available"
	}
	s.print(rec, cfg)
	return nil
}

func (s *Service) loadCache(cfg runtime.Config) (CacheRecord, bool) {
	data, err := s.FS.ReadFile(cfg.UpdateCacheFile)
	if err != nil {
		return CacheRecord{}, false
	}

	var rec CacheRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return CacheRecord{}, false
	}
	if rec.LocalVersion != "" && rec.LocalVersion != cfg.Version {
		return CacheRecord{}, false
	}
	expectedSource := sourceKey(cfg)
	if expectedSource != "none" {
		if rec.Source == "" || rec.Source != expectedSource {
			return CacheRecord{}, false
		}
	} else if rec.Source != "" && rec.Source != expectedSource {
		return CacheRecord{}, false
	}
	age := time.Since(rec.CheckedAt)
	if age < 0 || age >= cacheTTL {
		return CacheRecord{}, false
	}

	return rec, true
}

func (s *Service) saveCache(cfg runtime.Config, rec CacheRecord) {
	if err := s.FS.MkdirAll(filepath.Dir(cfg.UpdateCacheFile)); err != nil {
		return
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return
	}
	_ = s.FS.WriteFileAtomic(cfg.UpdateCacheFile, data, 0o644)
}

func (s *Service) remoteCheck(ctx context.Context, url string, rec *CacheRecord) error {
	rec.Method = "remote"

	if strings.HasPrefix(url, "file://") {
		path := strings.TrimPrefix(url, "file://")
		data, err := os.ReadFile(path)
		if err != nil {
			rec.Status = "error"
			rec.Message = fmt.Sprintf("failed to read VERSION from %s", url)
			return err
		}
		rec.LatestVersion = strings.TrimSpace(string(data))
		return s.finishRemote(rec)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		rec.Status = "error"
		rec.Message = err.Error()
		return err
	}
	resp, err := s.httpClient().Do(req)
	if err != nil {
		rec.Status = "error"
		rec.Message = err.Error()
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		rec.Status = "error"
		rec.Message = fmt.Sprintf("failed to download VERSION from %s", url)
		return errors.New(rec.Message)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		rec.Status = "error"
		rec.Message = err.Error()
		return err
	}

	rec.LatestVersion = strings.TrimSpace(string(body))
	return s.finishRemote(rec)
}

func (s *Service) finishRemote(rec *CacheRecord) error {
	if rec.LatestVersion == "" {
		rec.Status = "error"
		rec.Message = "empty VERSION response"
		return errors.New(rec.Message)
	}
	if rec.LocalVersion == rec.LatestVersion {
		rec.Status = "up-to-date"
		rec.Message = "remote VERSION matches local"
		return nil
	}
	if cmp, ok := compareVersion(rec.LocalVersion, rec.LatestVersion); ok {
		switch {
		case cmp < 0:
			rec.Status = "update-available"
			rec.Message = "remote VERSION is newer"
		case cmp > 0:
			rec.Status = "version-differs"
			rec.Message = "local VERSION is newer than remote"
		default:
			rec.Status = "up-to-date"
			rec.Message = "remote VERSION matches local"
		}
		return nil
	}
	rec.Status = "update-available"
	rec.Message = "remote VERSION differs"
	return nil
}

func (s *Service) gitCheck(ctx context.Context, cfg runtime.Config, rec *CacheRecord, fetch bool) error {
	rec.Method = "git"
	gitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if _, err := exec.LookPath("git"); err != nil {
		rec.Status = "error"
		rec.Message = "git not available"
		return err
	}

	if _, err := os.Stat(filepath.Join(cfg.RepoRoot, ".git")); err != nil {
		rec.Status = "error"
		rec.Message = "agent47 not installed from a git checkout"
		return err
	}

	if _, err := runGit(gitCtx, cfg.RepoRoot, "rev-parse", "HEAD"); err != nil {
		rec.Status = "error"
		rec.Message = gitErrorMessage(err, "unable to resolve git HEAD")
		return err
	}

	upstreamRef, err := runGit(gitCtx, cfg.RepoRoot, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	if err != nil {
		rec.Status = "error"
		rec.Message = gitErrorMessage(err, "git upstream not configured")
		return err
	}

	upstream := strings.TrimSpace(string(upstreamRef))
	if fetch {
		if _, err := runGit(gitCtx, cfg.RepoRoot, "fetch", "--quiet"); err != nil {
			rec.Status = "error"
			rec.Message = gitErrorMessage(err, "git fetch failed")
			return err
		}
	}

	counts, err := runGit(gitCtx, cfg.RepoRoot, "rev-list", "--left-right", "--count", fmt.Sprintf("HEAD...%s", upstream))
	if err != nil {
		rec.Status = "error"
		rec.Message = gitErrorMessage(err, "unable to compare local git checkout with upstream")
		return err
	}

	ahead, behind, err := parseAheadBehind(counts)
	if err != nil {
		rec.Status = "error"
		rec.Message = "unexpected git comparison output"
		return err
	}

	switch {
	case ahead == 0 && behind == 0:
		rec.Status = "git-tracking-current"
		rec.Message = trackingMessage(upstream, ahead, behind, fetch)
	case ahead > 0 && behind == 0:
		rec.Status = "local-ahead"
		rec.Message = trackingMessage(upstream, ahead, behind, fetch)
	case ahead == 0 && behind > 0:
		rec.Status = "git-behind"
		rec.Message = trackingMessage(upstream, ahead, behind, fetch)
	default:
		rec.Status = "git-diverged"
		rec.Message = trackingMessage(upstream, ahead, behind, fetch)
	}

	return nil
}

func (s *Service) print(rec CacheRecord, cfg runtime.Config) {
	switch rec.Status {
	case "up-to-date":
		s.Out.OK("Up to date (version %s)", rec.LocalVersion)
	case "update-available":
		s.Out.Printf("Update available: %s -> %s\n", rec.LocalVersion, rec.LatestVersion)
		s.Out.Info(install.UpdateInstructions(cfg))
	case "version-differs":
		s.Out.Info("Remote VERSION differs from local: %s vs %s", rec.LocalVersion, rec.LatestVersion)
	case "git-tracking-current":
		s.Out.Info("Git tracking reference is current: %s", rec.Message)
	case "local-ahead":
		s.Out.Info("Local copy is ahead of git upstream; no update needed (%s)", rec.Message)
	case "git-behind":
		s.Out.Printf("Update available from git upstream: %s\n", rec.Message)
		s.Out.Info(install.UpdateInstructions(cfg))
	case "git-diverged":
		s.Out.Warn("Local git checkout has diverged from upstream: %s", rec.Message)
	default:
		s.Out.Warn("Cannot check for updates: %s", rec.Message)
	}
}

func compareVersion(local, remote string) (int, bool) {
	localParts, ok := parseVersionParts(local)
	if !ok {
		return 0, false
	}
	remoteParts, ok := parseVersionParts(remote)
	if !ok {
		return 0, false
	}

	maxLen := len(localParts)
	if len(remoteParts) > maxLen {
		maxLen = len(remoteParts)
	}
	for i := 0; i < maxLen; i++ {
		localValue := 0
		if i < len(localParts) {
			localValue = localParts[i]
		}
		remoteValue := 0
		if i < len(remoteParts) {
			remoteValue = remoteParts[i]
		}
		switch {
		case localValue < remoteValue:
			return -1, true
		case localValue > remoteValue:
			return 1, true
		}
	}
	return 0, true
}

func parseVersionParts(raw string) ([]int, bool) {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "v")
	value = strings.TrimPrefix(value, "V")
	if value == "" {
		return nil, false
	}

	parts := strings.Split(value, ".")
	result := make([]int, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			return nil, false
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, false
		}
		result = append(result, n)
	}
	return result, true
}

func runGit(ctx context.Context, repoRoot string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repoRoot}, args...)...)
	return cmd.Output()
}

func gitErrorMessage(err error, fallback string) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "git update check canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "git update check timed out"
	default:
		return fallback
	}
}

func parseAheadBehind(raw []byte) (int, int, error) {
	fields := strings.Fields(string(raw))
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("unexpected rev-list count output: %q", strings.TrimSpace(string(raw)))
	}

	ahead, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, 0, err
	}
	behind, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, err
	}
	return ahead, behind, nil
}

func trackingMessage(upstream string, ahead, behind int, fetched bool) string {
	if fetched {
		switch {
		case ahead == 0 && behind == 0:
			return fmt.Sprintf("local checkout matches %s after git fetch", upstream)
		case ahead > 0 && behind == 0:
			return fmt.Sprintf("local checkout is ahead of %s by %d commit(s) after git fetch", upstream, ahead)
		case ahead == 0 && behind > 0:
			return fmt.Sprintf("%s is ahead by %d commit(s) after git fetch", upstream, behind)
		default:
			return fmt.Sprintf("local checkout and %s have diverged (%d ahead, %d behind) after git fetch", upstream, ahead, behind)
		}
	}

	switch {
	case ahead == 0 && behind == 0:
		return fmt.Sprintf("local checkout matches %s; remote fetch not performed", upstream)
	case ahead > 0 && behind == 0:
		return fmt.Sprintf("local checkout is ahead of %s by %d commit(s); remote fetch not performed", upstream, ahead)
	case ahead == 0 && behind > 0:
		return fmt.Sprintf("%s is ahead by %d commit(s) according to the local tracking ref", upstream, behind)
	default:
		return fmt.Sprintf("local checkout and %s have diverged (%d ahead, %d behind) according to the local tracking ref", upstream, ahead, behind)
	}
}

func sourceKey(cfg runtime.Config) string {
	if url := os.Getenv("AGENT47_VERSION_URL"); url != "" {
		return "remote:" + url
	}
	if cfg.RepoRoot != "" {
		return "git:" + filepath.Clean(cfg.RepoRoot)
	}
	return "none"
}

func (s *Service) httpClient() *http.Client {
	if s.HTTPClient != nil {
		return s.HTTPClient
	}
	return &http.Client{Timeout: 10 * time.Second}
}

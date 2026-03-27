package update

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func TestLoadCacheInvalidatesWhenUpdateSourceChanges(t *testing.T) {
	cacheFile := filepath.Join(t.TempDir(), "cache", "update.cache")
	cfg := runtime.Config{
		Version:         "1.2.3",
		RepoRoot:        filepath.Join(t.TempDir(), "repo"),
		UpdateCacheFile: cacheFile,
	}

	t.Setenv("AGENT47_VERSION_URL", "https://example.com/VERSION-a")
	service := New(cli.NewOutput(ioDiscard{}, ioDiscard{}))
	service.saveCache(cfg, CacheRecord{
		CheckedAt:     time.Now(),
		Status:        "update-available",
		Source:        sourceKey(cfg),
		LocalVersion:  "1.2.3",
		LatestVersion: "2.0.0",
		Message:       "cached",
	})

	t.Setenv("AGENT47_VERSION_URL", "https://example.com/VERSION-b")
	if _, ok := service.loadCache(cfg); ok {
		t.Fatal("expected cache miss after update source changed")
	}
}

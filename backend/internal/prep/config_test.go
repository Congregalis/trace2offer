package prep

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := loadConfig("/tmp/t2o-data", func(string) string { return "" })
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.Enabled {
		t.Fatalf("expected prep enabled by default")
	}
	if cfg.DefaultQuestionCount != defaultQuestionCount {
		t.Fatalf("expected default question count %d, got %d", defaultQuestionCount, cfg.DefaultQuestionCount)
	}
	if cfg.DataDir != filepath.Join("/tmp/t2o-data", "prep") {
		t.Fatalf("unexpected prep data dir: %q", cfg.DataDir)
	}

	gotScopes := scopeNames(cfg.SupportedScopes)
	wantScopes := []string{"topics", "companies", "leads"}
	if len(gotScopes) != len(wantScopes) {
		t.Fatalf("expected %d scopes, got %d", len(wantScopes), len(gotScopes))
	}
	for i := range wantScopes {
		if gotScopes[i] != wantScopes[i] {
			t.Fatalf("scope index %d expected %q, got %q", i, wantScopes[i], gotScopes[i])
		}
	}
}

func TestLoadConfigWithOverrides(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		envPrepEnabled:              "false",
		envPrepDefaultQuestionCount: "12",
		envPrepDataDir:              "custom-prep",
	}
	cfg, err := loadConfig("/tmp/t2o-data", func(key string) string {
		return values[key]
	})
	if err != nil {
		t.Fatalf("load config with override: %v", err)
	}
	if cfg.Enabled {
		t.Fatalf("expected prep disabled from override")
	}
	if cfg.DefaultQuestionCount != 12 {
		t.Fatalf("expected question count 12, got %d", cfg.DefaultQuestionCount)
	}
	if cfg.DataDir != filepath.Join("/tmp/t2o-data", "custom-prep") {
		t.Fatalf("unexpected prep data dir: %q", cfg.DataDir)
	}
}

func TestLoadConfigInvalidEnabled(t *testing.T) {
	t.Parallel()

	_, err := loadConfig("/tmp/t2o-data", func(key string) string {
		if key == envPrepEnabled {
			return "not_bool"
		}
		return ""
	})
	if err == nil {
		t.Fatal("expected error for invalid bool config")
	}
	if !strings.Contains(err.Error(), envPrepEnabled) {
		t.Fatalf("expected error mentioning %s, got %v", envPrepEnabled, err)
	}
}

func TestLoadConfigInvalidQuestionCount(t *testing.T) {
	t.Parallel()

	_, err := loadConfig("/tmp/t2o-data", func(key string) string {
		if key == envPrepDefaultQuestionCount {
			return "0"
		}
		return ""
	})
	if err == nil {
		t.Fatal("expected error for invalid default question count")
	}
	if !strings.Contains(err.Error(), envPrepDefaultQuestionCount) {
		t.Fatalf("expected error mentioning %s, got %v", envPrepDefaultQuestionCount, err)
	}
}

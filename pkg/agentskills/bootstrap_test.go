package agentskills

import (
	"strings"
	"testing"
)

func TestNewRequiresModel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.APIKey = "test-key"
	cfg.Model = "   "
	cfg.SkillsDirs = []string{"/path/that/does/not/exist"}

	_, err := New(nil, cfg)
	if err == nil {
		t.Fatal("expected error when model is empty")
	}
	if !strings.Contains(err.Error(), "Model is not set") {
		t.Fatalf("expected model error, got: %v", err)
	}
}

func TestDefaultConfigHasAllowedDir(t *testing.T) {
	cfg := DefaultConfig()
	if strings.TrimSpace(cfg.AllowedDir) == "" {
		t.Fatal("expected DefaultConfig.AllowedDir to be non-empty")
	}
}

func TestNewAllowsEmptySkillsDirs(t *testing.T) {
	cfg := DefaultConfig()
	cfg.APIKey = "test-key"
	cfg.Model = "gpt-4o-mini"
	cfg.SkillsDirs = nil

	app, err := New(nil, cfg)
	if err != nil {
		t.Fatalf("expected New to allow empty SkillsDirs, got error: %v", err)
	}
	if app == nil {
		t.Fatal("expected app to be initialized")
	}
}

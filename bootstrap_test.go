package main

import (
	"strings"
	"testing"
)

func TestNewAppRequiresModel(t *testing.T) {
	cfg := &Config{
		OpenAIAPIKey: "test-key",
		OpenAIModel:  "   ",
		// Non-existent dir verifies model validation happens before skill loading.
		SkillsDirs: []string{"/path/that/does/not/exist"},
	}

	_, err := NewApp(cfg)
	if err == nil {
		t.Fatal("expected error when OPENAI_MODEL is empty")
	}
	if !strings.Contains(err.Error(), "OPENAI_MODEL is not set") {
		t.Fatalf("expected OPENAI_MODEL error, got: %v", err)
	}
}

func TestDefaultAllowedDirNotEmpty(t *testing.T) {
	if strings.TrimSpace(defaultAllowedDir()) == "" {
		t.Fatal("expected defaultAllowedDir to be non-empty")
	}
}

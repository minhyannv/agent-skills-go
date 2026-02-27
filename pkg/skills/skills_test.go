// Tests for skill parsing and discovery.
package skills

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseSkillFile verifies front matter extraction.
func TestParseSkillFile(t *testing.T) {
	dir := t.TempDir()
	skillPath := filepath.Join(dir, "SKILL.md")
	content := `---
name: pdf
description: PDF processing skill
---
# Title
`
	if err := os.WriteFile(skillPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	skill, err := parseSkillFile(skillPath)
	if err != nil {
		t.Fatalf("parseSkillFile: %v", err)
	}
	if skill.Name != "pdf" {
		t.Fatalf("expected name pdf, got %q", skill.Name)
	}
	if skill.Description != "PDF processing skill" {
		t.Fatalf("expected description, got %q", skill.Description)
	}
	if skill.SkillFilePath != skillPath {
		t.Fatalf("expected path %q, got %q", skillPath, skill.SkillFilePath)
	}
}

// TestLoadSkillsFromDirSorted ensures deterministic ordering.
func TestLoadSkillsFromDirSorted(t *testing.T) {
	dir := t.TempDir()
	alphaDir := filepath.Join(dir, "alpha")
	betaDir := filepath.Join(dir, "beta")
	if err := os.MkdirAll(alphaDir, 0o755); err != nil {
		t.Fatalf("mkdir alpha: %v", err)
	}
	if err := os.MkdirAll(betaDir, 0o755); err != nil {
		t.Fatalf("mkdir beta: %v", err)
	}

	alphaSkill := `---
name: alpha
description: First
---
`
	betaSkill := `---
name: beta
description: Second
---
`
	if err := os.WriteFile(filepath.Join(betaDir, "SKILL.md"), []byte(betaSkill), 0o644); err != nil {
		t.Fatalf("write beta SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(alphaDir, "SKILL.md"), []byte(alphaSkill), 0o644); err != nil {
		t.Fatalf("write alpha SKILL.md: %v", err)
	}

	skills, err := loadSkillsFromDir(dir)
	if err != nil {
		t.Fatalf("loadSkillsFromDir: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}
	if skills[0].Name != "alpha" || skills[1].Name != "beta" {
		t.Fatalf("expected sorted skills [alpha beta], got [%s %s]", skills[0].Name, skills[1].Name)
	}
}

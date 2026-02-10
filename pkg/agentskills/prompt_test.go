// Tests for prompt generation helpers.
package agentskills

import (
	"strings"
	"testing"
)

// TestToPromptMarkdown validates markdown formatting of skills.
func TestToPromptMarkdown(t *testing.T) {
	skills := []*skill{
		{Name: "pdf", Description: "PDF tools", SkillFilePath: "/tmp/pdf"},
		{Name: "docx", Description: "DOCX tools", SkillFilePath: "/tmp/docx/SKILL.md"},
	}

	md := toPromptMarkdown(skills)
	if md == "" {
		t.Fatal("expected markdown output")
	}
	if !containsAll(md, []string{
		"## Available Skills",
		"<available_skills>",
		"<name>",
		"pdf",
		"<description>",
		"PDF tools",
		"<location>",
		"/tmp/pdf/SKILL.md",
		"/tmp/docx/SKILL.md",
	}) {
		t.Fatalf("markdown missing expected content:\n%s", md)
	}
}

// TestBuildSystemPrompt verifies system prompt composition.
func TestBuildSystemPrompt(t *testing.T) {
	skills := []*skill{
		{Name: "xlsx", Description: "Excel tools", SkillFilePath: "/tmp/xlsx/SKILL.md"},
	}
	prompt := buildSystemPrompt(skills)
	if prompt == "" {
		t.Fatal("expected prompt output")
	}
	if !containsAll(prompt, []string{
		"Tools available",
		"Available Skills",
		"Skill Selection Rules",
		"xlsx",
	}) {
		t.Fatalf("prompt missing expected content:\n%s", prompt)
	}
}

// containsAll reports whether all substrings exist in text.
func containsAll(text string, needles []string) bool {
	for _, needle := range needles {
		if !strings.Contains(text, needle) {
			return false
		}
	}
	return true
}

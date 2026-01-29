// System prompt assembly for skill-aware conversations.
package main

import (
	"fmt"
	"strings"
)

// BuildSystemPrompt constructs the system prompt, including tool and skill metadata.
func BuildSystemPrompt(skills []*Skill) string {
	var sb strings.Builder
	sb.WriteString("You are a tool-using assistant.")
	sb.WriteString("\nTools available: read_file, run_shell, run_python, run_go.")
	sb.WriteString("\n\n## Skill Use Protocol")
	sb.WriteString("\n- If a skill is relevant (or the user mentions a skill name like `$pdf`), use `read_file` to open its `SKILL.md` from the listed location.")
	sb.WriteString("\n- Do not assume skill details from the short description; always read the skill doc before using its scripts or workflow.")
	sb.WriteString("\n- Keep context small: read only what you need, and prefer skill-provided scripts/templates over re-implementing.")

	if md := ToPromptMarkdown(skills); md != "" {
		sb.WriteString("\n\n")
		sb.WriteString(md)
	}

	return strings.TrimSpace(sb.String())
}

// ToPromptMarkdown renders a markdown listing of available skills.
func ToPromptMarkdown(skills []*Skill) string {
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Available Skills\n")
	sb.WriteString("Use the skills below when relevant. Read each skill's `SKILL.md` with `read_file` before executing a skill.\n\n")

	for _, skill := range skills {
		name := sanitizeMarkdown(skill.Name)
		desc := sanitizeMarkdown(skill.Description)
		location := sanitizeMarkdown(skill.SkillFilePath)
		if desc == "" {
			desc = "No description provided."
		}
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n  - Location: %s\n", name, desc, location))
	}

	return strings.TrimSpace(sb.String())
}

// sanitizeMarkdown keeps markdown fields single-line and trimmed.
func sanitizeMarkdown(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return strings.TrimSpace(value)
}

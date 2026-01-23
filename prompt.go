// System prompt assembly for skill-aware conversations.
package main

import (
	"fmt"
	"strings"
)

// BuildSystemPrompt constructs the system prompt, including tool and skill metadata.
func BuildSystemPrompt(skills []*Skill) string {
	var sb strings.Builder
	sb.WriteString("You are a tool-using assistant. Use the tools to read skill docs, run scripts, and manipulate files when needed.")
	sb.WriteString("\nTools available: read_file, write_file, run_shell, run_python, run_go.")

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
	sb.WriteString("Use the skills below when relevant. Read the `SKILL.md` file before executing scripts.\n\n")

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

// RunShellTool implementation.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/openai/openai-go"
)

// RunShellTool implements the run_shell tool.
type RunShellTool struct {
	ctx ToolContext
}

// Name returns the tool name used by the model.
func (t *RunShellTool) Name() string {
	return "run_shell"
}

// Definition returns the OpenAI tool schema for run_shell.
func (t *RunShellTool) Definition() openai.ChatCompletionToolParam {
	return openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        "run_shell",
			Description: openai.String("Run a shell command using bash"),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "Shell command to run.",
					},
					"working_dir": map[string]any{
						"type":        "string",
						"description": "Working directory for the command.",
					},
					"timeout_seconds": map[string]any{
						"type":        "integer",
						"description": "Timeout in seconds before the command is terminated.",
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

// Execute runs a run_shell request.
func (t *RunShellTool) Execute(argText string) (string, error) {
	var args struct {
		Command        string `json:"command"`
		WorkingDir     string `json:"working_dir"`
		TimeoutSeconds int64  `json:"timeout_seconds"`
	}
	if err := json.Unmarshal([]byte(argText), &args); err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] run_shell: failed to parse arguments: %v", err)
		}
		return marshalToolResponse("run_shell", nil, err)
	}
	if t.ctx.Verbose {
		log.Printf("[verbose] run_shell: command_bytes=%d, working_dir=%s, timeout=%ds", len(args.Command), args.WorkingDir, args.TimeoutSeconds)
	}
	if args.Command == "" {
		return marshalToolResponse("run_shell", nil, errors.New("command is required"))
	}

	// Validate working directory
	validatedWorkingDir, err := validateWorkingDirWithAllowedDirs(args.WorkingDir, t.ctx.AllowedDirs)
	if err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] run_shell: working directory validation failed: %v", err)
		}
		return marshalToolResponse("run_shell", nil, fmt.Errorf("working directory validation failed: %w", err))
	}

	timeout := time.Duration(args.TimeoutSeconds) * time.Second
	command := args.Command
	if isDangerousCommand(command) {
		if t.ctx.Verbose {
			log.Printf("[verbose] run_shell: dangerous command blocked: %s", command)
		}
		return marshalToolResponse("run_shell", nil, fmt.Errorf("dangerous command not allowed: %s", command))
	}

	result := runCommand("bash", []string{"-lc", command}, validatedWorkingDir, timeout, t.ctx.Verbose)
	if t.ctx.Verbose {
		log.Printf("[verbose] run_shell: completed, exit_code=%d, duration=%dms", result.ExitCode, result.DurationMs)
	}
	return marshalToolResponse("run_shell", result, nil)
}

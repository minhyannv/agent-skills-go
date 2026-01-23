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

func (t *RunShellTool) Name() string {
	return "run_shell"
}

func (t *RunShellTool) Definition() openai.ChatCompletionToolParam {
	return openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        "run_shell",
			Description: openai.String("Run a shell command or script using bash"),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{
						"type": "string",
					},
					"working_dir": map[string]any{
						"type": "string",
					},
					"timeout_seconds": map[string]any{
						"type": "integer",
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

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
		log.Printf("[verbose] run_shell: command=%s, working_dir=%s, timeout=%ds", args.Command, args.WorkingDir, args.TimeoutSeconds)
	}
	if args.Command == "" {
		return marshalToolResponse("run_shell", nil, errors.New("command is required"))
	}

	// Check for dangerous commands
	if isDangerousCommand(args.Command) {
		if t.ctx.Verbose {
			log.Printf("[verbose] run_shell: dangerous command blocked: %s", args.Command)
		}
		return marshalToolResponse("run_shell", nil, fmt.Errorf("dangerous command not allowed: %s", args.Command))
	}

	// Validate working directory
	validatedWorkingDir, err := validateWorkingDir(args.WorkingDir, t.ctx.AllowedDir)
	if err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] run_shell: working directory validation failed: %v", err)
		}
		return marshalToolResponse("run_shell", nil, fmt.Errorf("working directory validation failed: %w", err))
	}

	timeout := time.Duration(args.TimeoutSeconds) * time.Second
	result := runCommand("bash", []string{"-lc", args.Command}, validatedWorkingDir, timeout, t.ctx.Verbose)
	if t.ctx.Verbose {
		log.Printf("[verbose] run_shell: completed, exit_code=%d, duration=%dms", result.ExitCode, result.DurationMs)
	}
	return marshalToolResponse("run_shell", result, nil)
}

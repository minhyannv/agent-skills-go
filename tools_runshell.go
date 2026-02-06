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
			Description: openai.String("Run a shell command or script using bash"),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path to a Shell script file (use either path or command).",
					},
					"command": map[string]any{
						"type":        "string",
						"description": "Shell command to run (use either path or command).",
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
			},
		},
	}
}

// Execute runs a run_shell request.
func (t *RunShellTool) Execute(argText string) (string, error) {
	var args struct {
		Path           string `json:"path"`
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
		log.Printf("[verbose] run_shell: path=%s, command_bytes=%d, working_dir=%s, timeout=%ds", args.Path, len(args.Command), args.WorkingDir, args.TimeoutSeconds)
	}
	if args.Path == "" && args.Command == "" {
		return marshalToolResponse("run_shell", nil, errors.New("path or command is required"))
	}
	if args.Path != "" && args.Command != "" {
		return marshalToolResponse("run_shell", nil, errors.New("provide either path or command, not both"))
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
	if args.Path != "" {
		// Validate script path
		validatedPath, err := validatePathWithAllowedDirs(args.Path, t.ctx.AllowedDirs)
		if err != nil {
			if t.ctx.Verbose {
				log.Printf("[verbose] run_shell: path validation failed: %v", err)
			}
			return marshalToolResponse("run_shell", nil, fmt.Errorf("path validation failed: %w", err))
		}
		if err := validateFileExists(validatedPath); err != nil {
			if t.ctx.Verbose {
				log.Printf("[verbose] run_shell: file validation failed: %v", err)
			}
			return marshalToolResponse("run_shell", nil, err)
		}

		result := runCommand("bash", []string{validatedPath}, validatedWorkingDir, timeout, t.ctx.Verbose)
		if t.ctx.Verbose {
			log.Printf("[verbose] run_shell: completed, exit_code=%d, duration=%dms", result.ExitCode, result.DurationMs)
		}
		return marshalToolResponse("run_shell", result, nil)
	}

	command := args.Command

	// Check for dangerous commands
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

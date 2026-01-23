// RunPythonTool implementation.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/openai/openai-go"
)

// RunPythonTool implements the run_python tool.
type RunPythonTool struct {
	ctx ToolContext
}

func (t *RunPythonTool) Name() string {
	return "run_python"
}

func (t *RunPythonTool) Definition() openai.ChatCompletionToolParam {
	return openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        "run_python",
			Description: openai.String("Run a Python script from file"),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type": "string",
					},
					"args": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "string",
						},
					},
					"working_dir": map[string]any{
						"type": "string",
					},
					"timeout_seconds": map[string]any{
						"type": "integer",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

func (t *RunPythonTool) Execute(argText string) (string, error) {
	var args struct {
		Path           string   `json:"path"`
		Args           []string `json:"args"`
		WorkingDir     string   `json:"working_dir"`
		TimeoutSeconds int64    `json:"timeout_seconds"`
	}
	if err := json.Unmarshal([]byte(argText), &args); err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] run_python: failed to parse arguments: %v", err)
		}
		return marshalToolResponse("run_python", nil, err)
	}
	if t.ctx.Verbose {
		log.Printf("[verbose] run_python: path=%s, args=%v, working_dir=%s, timeout=%ds", args.Path, args.Args, args.WorkingDir, args.TimeoutSeconds)
	}
	if args.Path == "" {
		return marshalToolResponse("run_python", nil, errors.New("path is required"))
	}

	python, err := resolvePython()
	if err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] run_python: failed to resolve python: %v", err)
		}
		return marshalToolResponse("run_python", nil, err)
	}
	if t.ctx.Verbose {
		log.Printf("[verbose] run_python: using python=%s", python)
	}

	// Validate script path
	validatedPath, err := validatePath(args.Path, t.ctx.AllowedDir)
	if err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] run_python: path validation failed: %v", err)
		}
		return marshalToolResponse("run_python", nil, fmt.Errorf("path validation failed: %w", err))
	}

	// Validate working directory
	validatedWorkingDir, err := validateWorkingDir(args.WorkingDir, t.ctx.AllowedDir)
	if err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] run_python: working directory validation failed: %v", err)
		}
		return marshalToolResponse("run_python", nil, fmt.Errorf("working directory validation failed: %w", err))
	}

	timeout := time.Duration(args.TimeoutSeconds) * time.Second
	result := runCommand(python, append([]string{validatedPath}, args.Args...), validatedWorkingDir, timeout, t.ctx.Verbose)
	if t.ctx.Verbose {
		log.Printf("[verbose] run_python: completed, exit_code=%d, duration=%dms", result.ExitCode, result.DurationMs)
	}
	return marshalToolResponse("run_python", result, nil)
}

// resolvePython locates a python interpreter.
func resolvePython() (string, error) {
	if path, err := exec.LookPath("python3"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("python"); err == nil {
		return path, nil
	}
	return "", errors.New("python executable not found")
}

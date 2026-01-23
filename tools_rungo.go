// RunGoTool implementation.
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

// RunGoTool implements the run_go tool.
type RunGoTool struct {
	ctx ToolContext
}

func (t *RunGoTool) Name() string {
	return "run_go"
}

func (t *RunGoTool) Definition() openai.ChatCompletionToolParam {
	return openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        "run_go",
			Description: openai.String("Run a Go script from file"),
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

func (t *RunGoTool) Execute(argText string) (string, error) {
	var args struct {
		Path           string   `json:"path"`
		Args           []string `json:"args"`
		WorkingDir     string   `json:"working_dir"`
		TimeoutSeconds int64    `json:"timeout_seconds"`
	}
	if err := json.Unmarshal([]byte(argText), &args); err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] run_go: failed to parse arguments: %v", err)
		}
		return marshalToolResponse("run_go", nil, err)
	}
	if t.ctx.Verbose {
		log.Printf("[verbose] run_go: path=%s, args=%v, working_dir=%s, timeout=%ds", args.Path, args.Args, args.WorkingDir, args.TimeoutSeconds)
	}
	if args.Path == "" {
		return marshalToolResponse("run_go", nil, errors.New("path is required"))
	}

	goBinary, err := resolveGo()
	if err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] run_go: failed to resolve go: %v", err)
		}
		return marshalToolResponse("run_go", nil, err)
	}
	if t.ctx.Verbose {
		log.Printf("[verbose] run_go: using go=%s", goBinary)
	}

	// Validate script path
	validatedPath, err := validatePath(args.Path, t.ctx.AllowedDir)
	if err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] run_go: path validation failed: %v", err)
		}
		return marshalToolResponse("run_go", nil, fmt.Errorf("path validation failed: %w", err))
	}

	// Validate working directory
	validatedWorkingDir, err := validateWorkingDir(args.WorkingDir, t.ctx.AllowedDir)
	if err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] run_go: working directory validation failed: %v", err)
		}
		return marshalToolResponse("run_go", nil, fmt.Errorf("working directory validation failed: %w", err))
	}

	timeout := time.Duration(args.TimeoutSeconds) * time.Second
	result := runCommand(goBinary, append([]string{"run", validatedPath}, args.Args...), validatedWorkingDir, timeout, t.ctx.Verbose)
	if t.ctx.Verbose {
		log.Printf("[verbose] run_go: completed, exit_code=%d, duration=%dms", result.ExitCode, result.DurationMs)
	}
	return marshalToolResponse("run_go", result, nil)
}

// resolveGo locates the go executable.
func resolveGo() (string, error) {
	if path, err := exec.LookPath("go"); err == nil {
		return path, nil
	}
	return "", errors.New("go executable not found")
}

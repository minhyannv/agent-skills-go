// RunGoTool implementation.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/openai/openai-go"
)

// RunGoTool implements the run_go tool.
type RunGoTool struct {
	ctx ToolContext
}

// Name returns the tool name used by the model.
func (t *RunGoTool) Name() string {
	return "run_go"
}

// Definition returns the OpenAI tool schema for run_go.
func (t *RunGoTool) Definition() openai.ChatCompletionToolParam {
	return openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        "run_go",
			Description: openai.String("Run a Go script from file or code"),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path to a Go source file (use either path or code).",
					},
					"code": map[string]any{
						"type":        "string",
						"description": "Inline Go code to execute (use either path or code).",
					},
					"args": map[string]any{
						"type":        "array",
						"description": "Arguments passed to the program.",
						"items": map[string]any{
							"type": "string",
						},
					},
					"working_dir": map[string]any{
						"type":        "string",
						"description": "Working directory for program execution.",
					},
					"timeout_seconds": map[string]any{
						"type":        "integer",
						"description": "Timeout in seconds before the program is terminated.",
					},
				},
			},
		},
	}
}

// Execute runs a run_go request.
func (t *RunGoTool) Execute(argText string) (string, error) {
	var args struct {
		Path           string   `json:"path"`
		Code           string   `json:"code"`
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
		log.Printf("[verbose] run_go: path=%s, code_bytes=%d, args=%v, working_dir=%s, timeout=%ds", args.Path, len(args.Code), args.Args, args.WorkingDir, args.TimeoutSeconds)
	}
	if args.Path == "" && args.Code == "" {
		return marshalToolResponse("run_go", nil, errors.New("path or code is required"))
	}
	if args.Path != "" && args.Code != "" {
		return marshalToolResponse("run_go", nil, errors.New("provide either path or code, not both"))
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

	// Validate working directory
	validatedWorkingDir, err := validateWorkingDirWithAllowedDirs(args.WorkingDir, t.ctx.AllowedDirs)
	if err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] run_go: working directory validation failed: %v", err)
		}
		return marshalToolResponse("run_go", nil, fmt.Errorf("working directory validation failed: %w", err))
	}

	scriptPath := args.Path
	if args.Code != "" {
		tempDir, err := chooseTempDir(validatedWorkingDir, t.ctx.AllowedDirs)
		if err != nil {
			if t.ctx.Verbose {
				log.Printf("[verbose] run_go: temp dir selection failed: %v", err)
			}
			return marshalToolResponse("run_go", nil, err)
		}
		tempPath, err := writeTempFile(tempDir, "run_go_*.go", args.Code)
		if err != nil {
			if t.ctx.Verbose {
				log.Printf("[verbose] run_go: temp file write failed: %v", err)
			}
			return marshalToolResponse("run_go", nil, err)
		}
		defer func() {
			_ = os.Remove(tempPath)
		}()
		scriptPath = tempPath
	}

	// Validate script path
	validatedPath, err := validatePathWithAllowedDirs(scriptPath, t.ctx.AllowedDirs)
	if err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] run_go: path validation failed: %v", err)
		}
		return marshalToolResponse("run_go", nil, fmt.Errorf("path validation failed: %w", err))
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

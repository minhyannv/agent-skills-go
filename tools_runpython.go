// RunPythonTool implementation.
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

// RunPythonTool implements the run_python tool.
type RunPythonTool struct {
	ctx ToolContext
}

// Name returns the tool name used by the model.
func (t *RunPythonTool) Name() string {
	return "run_python"
}

// Definition returns the OpenAI tool schema for run_python.
func (t *RunPythonTool) Definition() openai.ChatCompletionToolParam {
	return openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        "run_python",
			Description: openai.String("Run a Python script from file or inline code"),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path to a Python script file (use either path or code).",
					},
					"code": map[string]any{
						"type":        "string",
						"description": "Inline Python code to execute (use either path or code).",
					},
					"args": map[string]any{
						"type":        "array",
						"description": "Arguments passed to the script.",
						"items": map[string]any{
							"type": "string",
						},
					},
					"working_dir": map[string]any{
						"type":        "string",
						"description": "Working directory for script execution.",
					},
					"timeout_seconds": map[string]any{
						"type":        "integer",
						"description": "Timeout in seconds before the script is terminated.",
					},
				},
			},
		},
	}
}

// Execute runs a run_python request.
func (t *RunPythonTool) Execute(argText string) (string, error) {
	var args struct {
		Path           string   `json:"path"`
		Code           string   `json:"code"`
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
		log.Printf("[verbose] run_python: path=%s, code_bytes=%d, args=%v, working_dir=%s, timeout=%ds", args.Path, len(args.Code), args.Args, args.WorkingDir, args.TimeoutSeconds)
	}
	if args.Path == "" && args.Code == "" {
		return marshalToolResponse("run_python", nil, errors.New("path or code is required"))
	}
	if args.Path != "" && args.Code != "" {
		return marshalToolResponse("run_python", nil, errors.New("provide either path or code, not both"))
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

	// Validate working directory
	validatedWorkingDir, err := validateWorkingDirWithAllowedDirs(args.WorkingDir, t.ctx.AllowedDirs)
	if err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] run_python: working directory validation failed: %v", err)
		}
		return marshalToolResponse("run_python", nil, fmt.Errorf("working directory validation failed: %w", err))
	}

	scriptPath := args.Path
	if args.Code != "" {
		tempDir, err := chooseTempDir(validatedWorkingDir, t.ctx.AllowedDirs)
		if err != nil {
			if t.ctx.Verbose {
				log.Printf("[verbose] run_python: temp dir selection failed: %v", err)
			}
			return marshalToolResponse("run_python", nil, err)
		}
		tempPath, err := writeTempFile(tempDir, "run_python_*.py", args.Code)
		if err != nil {
			if t.ctx.Verbose {
				log.Printf("[verbose] run_python: temp file write failed: %v", err)
			}
			return marshalToolResponse("run_python", nil, err)
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
			log.Printf("[verbose] run_python: path validation failed: %v", err)
		}
		return marshalToolResponse("run_python", nil, fmt.Errorf("path validation failed: %w", err))
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

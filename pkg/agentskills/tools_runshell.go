package agentskills

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/openai/openai-go"
)

type runShellTool struct {
	ctx toolContext
}

func (t *runShellTool) name() string {
	return "run_shell"
}

func (t *runShellTool) definition() openai.ChatCompletionToolParam {
	return openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        "run_shell",
			Description: openai.String("Run a command without shell expansion"),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "Command to run.",
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

func (t *runShellTool) execute(argText string) (string, error) {
	var args struct {
		Command        string `json:"command"`
		WorkingDir     string `json:"working_dir"`
		TimeoutSeconds int64  `json:"timeout_seconds"`
	}
	if err := json.Unmarshal([]byte(argText), &args); err != nil {
		t.ctx.debugf("[verbose] run_shell: failed to parse arguments: %v", err)
		return marshalToolResponse("run_shell", nil, err)
	}
	t.ctx.debugf("[verbose] run_shell: command_bytes=%d, working_dir=%s, timeout=%ds", len(args.Command), args.WorkingDir, args.TimeoutSeconds)
	if args.Command == "" {
		return marshalToolResponse("run_shell", nil, errors.New("command is required"))
	}
	if blockedToken, blocked := containsBlockedShellSyntax(args.Command); blocked {
		return marshalToolResponse("run_shell", nil, fmt.Errorf("shell control syntax not allowed: %q", blockedToken))
	}

	argv, err := parseCommandLine(args.Command)
	if err != nil {
		return marshalToolResponse("run_shell", nil, fmt.Errorf("invalid command: %w", err))
	}
	if len(argv) == 0 {
		return marshalToolResponse("run_shell", nil, errors.New("command is required"))
	}

	// Validate working directory
	validatedWorkingDir, err := validateWorkingDirWithAllowedDirs(args.WorkingDir, t.ctx.AllowedDirs)
	if err != nil {
		t.ctx.debugf("[verbose] run_shell: working directory validation failed: %v", err)
		return marshalToolResponse("run_shell", nil, fmt.Errorf("working directory validation failed: %w", err))
	}

	timeout := time.Duration(args.TimeoutSeconds) * time.Second
	if isShellExecutable(argv[0]) {
		return marshalToolResponse("run_shell", nil, fmt.Errorf("shell executables are not allowed: %s", argv[0]))
	}
	if isDangerousExecutable(argv[0]) {
		t.ctx.debugf("[verbose] run_shell: dangerous command blocked: %s", argv[0])
		return marshalToolResponse("run_shell", nil, fmt.Errorf("dangerous command not allowed: %s", argv[0]))
	}

	result := t.ctx.runCommand(argv[0], argv[1:], validatedWorkingDir, timeout)
	t.ctx.debugf("[verbose] run_shell: completed, exit_code=%d, duration=%dms", result.ExitCode, result.DurationMs)
	return marshalToolResponse("run_shell", result, nil)
}

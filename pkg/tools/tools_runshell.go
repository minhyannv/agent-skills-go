package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/openai/openai-go"
)

type runShellTool struct {
	ctx Context
}

// commandResult captures command execution metadata and output.
type commandResult struct {
	Command    string   `json:"command"`
	Args       []string `json:"args,omitempty"`
	WorkingDir string   `json:"working_dir,omitempty"`
	ExitCode   int      `json:"exit_code"`
	Stdout     string   `json:"stdout,omitempty"`
	Stderr     string   `json:"stderr,omitempty"`
	DurationMs int64    `json:"duration_ms"`
	Error      string   `json:"error,omitempty"`
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

// runCommand executes a command with timeout and captures stdout/stderr.
func (ctx Context) runCommand(command string, args []string, workingDir string, timeout time.Duration) commandResult {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	ctx.debugf("[verbose] runCommand: command=%s, args=%v, working_dir=%s, timeout=%v", command, args, workingDir, timeout)
	execCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, command, args...)
	cmd.Env = sanitizedEnv()
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start).Milliseconds()

	exitCode := 0
	errText := ""
	if err != nil {
		errText = err.Error()
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else if errors.Is(err, context.DeadlineExceeded) || errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			exitCode = -1
			ctx.debugf("[verbose] runCommand: timeout exceeded after %v", timeout)
		} else {
			exitCode = -1
		}
		ctx.debugf("[verbose] runCommand: error occurred: %v (exit_code=%d)", err, exitCode)
	}

	stdoutLen := stdout.Len()
	stderrLen := stderr.Len()
	ctx.debugf("[verbose] runCommand: completed, exit_code=%d, duration=%dms, stdout=%d bytes, stderr=%d bytes", exitCode, duration, stdoutLen, stderrLen)
	if stderrLen > 0 {
		stderrPreview := stderr.String()
		if len(stderrPreview) > 500 {
			ctx.debugf("[verbose] runCommand: stderr preview: %s...", stderrPreview[:500])
		} else {
			ctx.debugf("[verbose] runCommand: stderr: %s", stderrPreview)
		}
	}

	return commandResult{
		Command:    command,
		Args:       args,
		WorkingDir: workingDir,
		ExitCode:   exitCode,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		DurationMs: duration,
		Error:      errText,
	}
}

// sanitizedEnv keeps only low-risk environment variables for subprocesses.
func sanitizedEnv() []string {
	allowedPrefixes := []string{
		"PATH=",
		"HOME=",
		"USER=",
		"LOGNAME=",
		"SHELL=",
		"TMPDIR=",
		"TMP=",
		"TEMP=",
		"LANG=",
		"LC_",
		"TERM=",
		"PWD=",
	}

	env := make([]string, 0, len(allowedPrefixes))
	for _, kv := range os.Environ() {
		for _, prefix := range allowedPrefixes {
			if strings.HasPrefix(kv, prefix) {
				env = append(env, kv)
				break
			}
		}
	}
	return env
}

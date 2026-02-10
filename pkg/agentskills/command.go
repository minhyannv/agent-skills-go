// Command execution helpers for tool runners.
package agentskills

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"
)

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

// runCommand executes a command with timeout and captures stdout/stderr.
func (ctx toolContext) runCommand(command string, args []string, workingDir string, timeout time.Duration) commandResult {
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

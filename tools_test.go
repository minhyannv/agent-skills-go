// Tests for tool execution helpers.
package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// toolResponseTest is a minimal response shape for assertions.
type toolResponseTest struct {
	OK   bool            `json:"ok"`
	Tool string          `json:"tool"`
	Data json.RawMessage `json:"data"`
	Err  string          `json:"error"`
}

// TestToolReadWriteFile validates read/write behavior and truncation.
func TestToolReadWriteFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "note.txt")
	toolCtx := ToolContext{
		MaxReadBytes: defaultMaxReadBytes,
		Verbose:      false,
		AllowedDirs:  []string{dir}, // Set allowed dir to test directory
		Ctx:          context.Background(),
	}
	writeTool := &WriteFileTool{ctx: toolCtx}
	readTool := &ReadFileTool{ctx: toolCtx}

	writeArgs := `{"path":"` + filePath + `","content":"hello","overwrite":false}`
	writeResp, err := writeTool.Execute(writeArgs)
	if err != nil {
		t.Fatalf("writeFile: %v", err)
	}

	var writeRespData toolResponseTest
	if err := json.Unmarshal([]byte(writeResp), &writeRespData); err != nil {
		t.Fatalf("unmarshal write response: %v", err)
	}
	if !writeRespData.OK {
		t.Fatalf("write failed: %s", writeRespData.Err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected file content: %q", string(data))
	}

	readArgs := `{"path":"` + filePath + `","max_bytes":3}`
	readResp, err := readTool.Execute(readArgs)
	if err != nil {
		t.Fatalf("readFile: %v", err)
	}
	var readRespData toolResponseTest
	if err := json.Unmarshal([]byte(readResp), &readRespData); err != nil {
		t.Fatalf("unmarshal read response: %v", err)
	}
	if !readRespData.OK {
		t.Fatalf("read failed: %s", readRespData.Err)
	}
	var readData struct {
		Path      string `json:"path"`
		Bytes     int    `json:"bytes"`
		Truncated bool   `json:"truncated"`
		Content   string `json:"content"`
	}
	if err := json.Unmarshal(readRespData.Data, &readData); err != nil {
		t.Fatalf("unmarshal read data: %v", err)
	}
	if readData.Content != "hel" || !readData.Truncated {
		t.Fatalf("unexpected read data: %+v", readData)
	}
}

// TestToolWriteFileNoOverwrite ensures overwrite=false is enforced.
func TestToolWriteFileNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "note.txt")
	toolCtx := ToolContext{
		MaxReadBytes: defaultMaxReadBytes,
		Verbose:      false,
		AllowedDirs:  []string{dir}, // Set allowed dir to test directory
		Ctx:          context.Background(),
	}
	writeTool := &WriteFileTool{ctx: toolCtx}

	writeArgs := `{"path":"` + filePath + `","content":"first","overwrite":false}`
	_, err := writeTool.Execute(writeArgs)
	if err != nil {
		t.Fatalf("writeFile: %v", err)
	}

	writeAgain, err := writeTool.Execute(writeArgs)
	if err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	var resp toolResponseTest
	if err := json.Unmarshal([]byte(writeAgain), &resp); err != nil {
		t.Fatalf("unmarshal write response: %v", err)
	}
	if resp.OK {
		t.Fatalf("expected overwrite failure, got ok")
	}
}

// TestToolRunShell verifies command execution and output capture.
func TestToolRunShell(t *testing.T) {
	toolCtx := ToolContext{
		MaxReadBytes: defaultMaxReadBytes,
		Verbose:      false,
		AllowedDirs:  nil, // No restriction for this test
		Ctx:          context.Background(),
	}
	shellTool := &RunShellTool{ctx: toolCtx}
	args := `{"command":"echo hello"}`
	resp, err := shellTool.Execute(args)
	if err != nil {
		t.Fatalf("runShell: %v", err)
	}
	var toolResp toolResponseTest
	if err := json.Unmarshal([]byte(resp), &toolResp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !toolResp.OK {
		t.Fatalf("runShell failed: %s", toolResp.Err)
	}
	var result commandResult
	if err := json.Unmarshal(toolResp.Data, &result); err != nil {
		t.Fatalf("unmarshal command result: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "hello") {
		t.Fatalf("unexpected stdout: %q", result.Stdout)
	}
}

// TestToolRunShellQuotes verifies quoted arguments are parsed correctly.
func TestToolRunShellQuotes(t *testing.T) {
	toolCtx := ToolContext{
		MaxReadBytes: defaultMaxReadBytes,
		Verbose:      false,
		AllowedDirs:  nil,
		Ctx:          context.Background(),
	}
	shellTool := &RunShellTool{ctx: toolCtx}
	args := `{"command":"echo \"hello world\""}`

	resp, err := shellTool.Execute(args)
	if err != nil {
		t.Fatalf("runShell: %v", err)
	}
	var toolResp toolResponseTest
	if err := json.Unmarshal([]byte(resp), &toolResp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !toolResp.OK {
		t.Fatalf("runShell failed: %s", toolResp.Err)
	}
	var result commandResult
	if err := json.Unmarshal(toolResp.Data, &result); err != nil {
		t.Fatalf("unmarshal command result: %v", err)
	}
	if !strings.Contains(result.Stdout, "hello world") {
		t.Fatalf("unexpected stdout: %q", result.Stdout)
	}
}

// TestToolRunShellSanitizedEnv ensures sensitive env vars are not inherited by subprocesses.
func TestToolRunShellSanitizedEnv(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "secret-for-test")
	toolCtx := ToolContext{
		MaxReadBytes: defaultMaxReadBytes,
		Verbose:      false,
		AllowedDirs:  nil,
		Ctx:          context.Background(),
	}
	shellTool := &RunShellTool{ctx: toolCtx}
	args := `{"command":"env"}`

	resp, err := shellTool.Execute(args)
	if err != nil {
		t.Fatalf("runShell: %v", err)
	}
	var toolResp toolResponseTest
	if err := json.Unmarshal([]byte(resp), &toolResp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !toolResp.OK {
		t.Fatalf("runShell failed: %s", toolResp.Err)
	}
	var result commandResult
	if err := json.Unmarshal(toolResp.Data, &result); err != nil {
		t.Fatalf("unmarshal command result: %v", err)
	}
	if strings.Contains(result.Stdout, "OPENAI_API_KEY=secret-for-test") {
		t.Fatalf("sensitive env variable leaked to subprocess")
	}
}

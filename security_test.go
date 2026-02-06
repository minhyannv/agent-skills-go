// Tests for security utilities.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestValidatePath tests path validation.
func TestValidatePath(t *testing.T) {
	allowedDir := t.TempDir()

	tests := []struct {
		name      string
		path      string
		allowedDir string
		wantErr   bool
	}{
		{
			name:      "valid path within allowed dir",
			path:      "test.txt",
			allowedDir: allowedDir,
			wantErr:   false,
		},
		{
			name:      "path traversal attempt",
			path:      "../../etc/passwd",
			allowedDir: allowedDir,
			wantErr:   true,
		},
		{
			name:      "empty path",
			path:      "",
			allowedDir: allowedDir,
			wantErr:   true,
		},
		{
			name:      "no restriction when allowedDir is empty",
			path:      "/tmp/test.txt",
			allowedDir: "",
			wantErr:   false,
		},
		{
			name:      "path outside allowed dir",
			path:      "/tmp/test.txt",
			allowedDir: allowedDir,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For empty path test, use it directly
			testPath := tt.path
			if tt.path != "" && tt.allowedDir != "" && !filepath.IsAbs(tt.path) {
				testPath = filepath.Join(tt.allowedDir, tt.path)
			}

			_, err := validatePath(testPath, tt.allowedDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestIsDangerousCommand tests dangerous command detection.
func TestIsDangerousCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{
			name:    "dangerous command rm",
			command: "rm -rf /tmp/test",
			want:    true,
		},
		{
			name:    "dangerous command dd",
			command: "dd if=/dev/zero of=/dev/sda",
			want:    true,
		},
		{
			name:    "safe command echo",
			command: "echo hello",
			want:    false,
		},
		{
			name:    "safe command ls",
			command: "ls -la",
			want:    false,
		},
		{
			name:    "empty command",
			command: "",
			want:    false,
		},
		{
			name:    "dangerous command with path",
			command: "/usr/bin/rm -rf /tmp",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDangerousCommand(tt.command); got != tt.want {
				t.Errorf("isDangerousCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestValidateFileExists tests file existence validation.
func TestValidateFileExists(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	dirPath := dir

	// Create a test file
	if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "existing file",
			path:    filePath,
			wantErr: false,
		},
		{
			name:    "directory instead of file",
			path:    dirPath,
			wantErr: true,
		},
		{
			name:    "non-existent file",
			path:    filepath.Join(dir, "nonexistent.txt"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFileExists(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFileExists() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestToolReadFileSecurity tests security restrictions in read_file.
func TestToolReadFileSecurity(t *testing.T) {
	allowedDir := t.TempDir()
	testFile := filepath.Join(allowedDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	toolCtx := ToolContext{
		MaxReadBytes: defaultMaxReadBytes,
		Verbose:      false,
		AllowedDirs:  []string{allowedDir},
		Ctx:          nil,
	}
	readTool := &ReadFileTool{ctx: toolCtx}

	// Test path traversal attempt
	args := `{"path":"../test.txt"}`
	resp, err := readTool.Execute(args)
	if err != nil {
		t.Fatalf("readFile returned error: %v", err)
	}

	var result toolResponseTest
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if result.OK {
		t.Error("expected path traversal to fail, but it succeeded")
	}
}

// TestToolRunShellSecurity tests security restrictions in run_shell.
func TestToolRunShellSecurity(t *testing.T) {
	toolCtx := ToolContext{
		MaxReadBytes: defaultMaxReadBytes,
		Verbose:      false,
		AllowedDirs:  nil,
		Ctx:          nil,
	}
	shellTool := &RunShellTool{ctx: toolCtx}

	// Test dangerous command
	args := `{"command":"rm -rf /tmp/test"}`
	resp, err := shellTool.Execute(args)
	if err != nil {
		t.Fatalf("runShell returned error: %v", err)
	}

	var result toolResponseTest
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if result.OK {
		t.Error("expected dangerous command to be blocked, but it succeeded")
	}
}

// TestToolRunShellPathSecurity tests path restrictions in run_shell.
func TestToolRunShellPathSecurity(t *testing.T) {
	allowedDir := t.TempDir()
	disallowedDir := t.TempDir()
	scriptPath := filepath.Join(disallowedDir, "script.sh")
	if err := os.WriteFile(scriptPath, []byte("echo hi"), 0o644); err != nil {
		t.Fatalf("failed to create script: %v", err)
	}

	toolCtx := ToolContext{
		MaxReadBytes: defaultMaxReadBytes,
		Verbose:      false,
		AllowedDirs:  []string{allowedDir},
		Ctx:          nil,
	}
	shellTool := &RunShellTool{ctx: toolCtx}

	args := `{"path":"` + scriptPath + `"}`
	resp, err := shellTool.Execute(args)
	if err != nil {
		t.Fatalf("runShell returned error: %v", err)
	}

	var result toolResponseTest
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if result.OK {
		t.Error("expected path restriction to fail, but it succeeded")
	}
}

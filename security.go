// Security utilities for path validation and command safety checks.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// validatePath ensures a path is safe and within allowed directory.
func validatePath(path string, allowedDir string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Clean the path to resolve any . or .. components
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("path traversal not allowed: %s", path)
	}

	// If no allowed directory is set, allow any path (backward compatibility)
	if allowedDir == "" {
		absPath, err := filepath.Abs(cleanPath)
		if err != nil {
			return "", fmt.Errorf("invalid path: %w", err)
		}
		return absPath, nil
	}

	// Resolve absolute paths
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	absAllowed, err := filepath.Abs(allowedDir)
	if err != nil {
		return "", fmt.Errorf("invalid allowed directory: %w", err)
	}

	// Ensure the path is within the allowed directory
	if !strings.HasPrefix(absPath, absAllowed) {
		return "", fmt.Errorf("path outside allowed directory: %s (allowed: %s)", absPath, absAllowed)
	}

	return absPath, nil
}

// validateWorkingDir ensures a working directory is safe and within allowed directory.
func validateWorkingDir(workingDir string, allowedDir string) (string, error) {
	if workingDir == "" {
		return "", nil // Empty working dir is allowed (uses current dir)
	}

	return validatePath(workingDir, allowedDir)
}

// dangerousCommands is a list of commands that should be restricted.
var dangerousCommands = []string{
	"rm", "rmdir", "dd", "mkfs", "fdisk", "shutdown", "reboot", "halt",
	"poweroff", "init", "killall", "kill", "pkill", "killall5",
	"chmod", "chown", "chgrp", "mount", "umount", "mkfs", "fdisk",
	"parted", "sfdisk", "wipefs", "mkfs.ext", "mkfs.vfat", "mkfs.ntfs",
}

// isDangerousCommand checks if a command is potentially dangerous.
func isDangerousCommand(cmd string) bool {
	if cmd == "" {
		return false
	}

	// Split command into parts
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false
	}

	// Get the base command name (first part)
	baseCmd := filepath.Base(parts[0])

	// Check against dangerous commands list
	for _, dangerous := range dangerousCommands {
		if baseCmd == dangerous {
			return true
		}
	}

	return false
}

// validateFileExists checks if a file exists and is not a directory.
func validateFileExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory: %s", path)
	}
	return nil
}

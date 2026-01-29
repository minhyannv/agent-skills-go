// Security utilities for path validation and command safety checks.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// normalizeAllowedDirs returns a sorted, deduplicated list of absolute directories.
func normalizeAllowedDirs(allowedDirs []string) []string {
	normalized := make([]string, 0, len(allowedDirs))
	seen := map[string]struct{}{}
	for _, dir := range allowedDirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		abs = filepath.Clean(abs)
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		normalized = append(normalized, abs)
	}
	slices.Sort(normalized)
	return normalized
}

// validatePathWithAllowedDirs ensures a path is safe and within one of the allowed directories.
// If allowedDirs is empty, any path is permitted (backward compatibility).
func validatePathWithAllowedDirs(path string, allowedDirs []string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Clean the path to resolve any . or .. components
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("path traversal not allowed: %s", path)
	}

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	roots := normalizeAllowedDirs(allowedDirs)
	if len(roots) == 0 {
		return absPath, nil
	}

	for _, root := range roots {
		rel, err := filepath.Rel(root, absPath)
		if err != nil {
			continue
		}
		if rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..") {
			return absPath, nil
		}
	}

	return "", fmt.Errorf("path outside allowed directories: %s (allowed: %s)", absPath, strings.Join(roots, ", "))
}

// validatePath ensures a path is safe and within allowed directory.
func validatePath(path string, allowedDir string) (string, error) {
	if strings.TrimSpace(allowedDir) == "" {
		return validatePathWithAllowedDirs(path, nil)
	}
	return validatePathWithAllowedDirs(path, []string{allowedDir})
}

// validateWorkingDir ensures a working directory is safe and within allowed directory.
func validateWorkingDir(workingDir string, allowedDir string) (string, error) {
	if workingDir == "" {
		return "", nil // Empty working dir is allowed (uses current dir)
	}

	return validatePath(workingDir, allowedDir)
}

// validateWorkingDirWithAllowedDirs ensures a working directory is safe and within allowed directories.
func validateWorkingDirWithAllowedDirs(workingDir string, allowedDirs []string) (string, error) {
	if workingDir == "" {
		return "", nil // Empty working dir is allowed (uses current dir)
	}

	return validatePathWithAllowedDirs(workingDir, allowedDirs)
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

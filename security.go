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
	if hasParentTraversal(cleanPath) {
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

// hasParentTraversal reports whether a path contains a parent directory segment.
func hasParentTraversal(cleanPath string) bool {
	if cleanPath == ".." {
		return true
	}
	for _, part := range strings.Split(cleanPath, string(filepath.Separator)) {
		if part == ".." {
			return true
		}
	}
	return false
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
var dangerousCommands = map[string]struct{}{
	"rm":         {},
	"rmdir":      {},
	"dd":         {},
	"mkfs":       {},
	"fdisk":      {},
	"shutdown":   {},
	"reboot":     {},
	"halt":       {},
	"poweroff":   {},
	"init":       {},
	"killall":    {},
	"kill":       {},
	"pkill":      {},
	"killall5":   {},
	"chmod":      {},
	"chown":      {},
	"chgrp":      {},
	"mount":      {},
	"umount":     {},
	"parted":     {},
	"sfdisk":     {},
	"wipefs":     {},
	"mkfs.ext":   {},
	"mkfs.vfat":  {},
	"mkfs.ntfs":  {},
	"mkfs.ext2":  {},
	"mkfs.ext3":  {},
	"mkfs.ext4":  {},
	"mkfs.xfs":   {},
	"mkfs.btrfs": {},
}

// shellExecutables are blocked to prevent nested shell execution.
var shellExecutables = map[string]struct{}{
	"sh":   {},
	"bash": {},
	"zsh":  {},
	"dash": {},
	"fish": {},
}

// isDangerousCommand checks if a command is potentially dangerous.
func isDangerousCommand(cmd string) bool {
	executable, ok := firstExecutableFromCommand(cmd)
	if !ok {
		return false
	}
	return isDangerousExecutable(executable)
}

// firstExecutableFromCommand parses a command line and returns its executable.
func firstExecutableFromCommand(cmd string) (string, bool) {
	args, err := parseCommandLine(cmd)
	if err != nil || len(args) == 0 {
		return "", false
	}
	return args[0], true
}

// isDangerousExecutable checks whether an executable name is in the deny list.
func isDangerousExecutable(executable string) bool {
	baseCmd := strings.ToLower(filepath.Base(strings.TrimSpace(executable)))
	if baseCmd == "" {
		return false
	}
	_, blocked := dangerousCommands[baseCmd]
	return blocked
}

// isShellExecutable reports whether executable is a shell interpreter.
func isShellExecutable(executable string) bool {
	baseCmd := strings.ToLower(filepath.Base(strings.TrimSpace(executable)))
	if baseCmd == "" {
		return false
	}
	_, isShell := shellExecutables[baseCmd]
	return isShell
}

// containsBlockedShellSyntax checks for shell control operators and expansions.
func containsBlockedShellSyntax(command string) (string, bool) {
	blocked := []string{"&&", "||", ";", "|", ">", "<", "`", "$(", "\n", "\r"}
	for _, token := range blocked {
		if strings.Contains(command, token) {
			return token, true
		}
	}
	return "", false
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

// parseCommandLine parses a command string into argv without shell execution.
func parseCommandLine(input string) ([]string, error) {
	var (
		args     []string
		current  strings.Builder
		inSingle bool
		inDouble bool
		escaped  bool
	)

	flush := func() {
		if current.Len() == 0 {
			return
		}
		args = append(args, current.String())
		current.Reset()
	}

	for _, r := range input {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\' && !inSingle:
			escaped = true
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case (r == ' ' || r == '\t') && !inSingle && !inDouble:
			flush()
		default:
			current.WriteRune(r)
		}
	}

	if escaped {
		return nil, fmt.Errorf("unterminated escape in command")
	}
	if inSingle || inDouble {
		return nil, fmt.Errorf("unterminated quote in command")
	}
	flush()

	return args, nil
}

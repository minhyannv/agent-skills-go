// Helpers for inline code execution.
package main

import (
	"errors"
	"os"
)

// chooseTempDir selects a directory for temporary code files.
func chooseTempDir(validatedWorkingDir string, allowedDirs []string) (string, error) {
	if validatedWorkingDir != "" {
		return validatedWorkingDir, nil
	}
	roots := normalizeAllowedDirs(allowedDirs)
	if len(roots) > 0 {
		return roots[0], nil
	}
	if len(allowedDirs) > 0 {
		return "", errors.New("no valid allowed_dir available for temp file")
	}
	return "", nil
}

// writeTempFile creates a temp file with the provided content and returns its path.
func writeTempFile(dir string, pattern string, content string) (string, error) {
	file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	name := file.Name()
	if _, err := file.WriteString(content); err != nil {
		_ = file.Close()
		_ = os.Remove(name)
		return "", err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	return name, nil
}

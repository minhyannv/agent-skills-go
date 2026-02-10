package agentskills

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/openai/openai-go"
)

type readFileTool struct {
	ctx toolContext
}

func (t *readFileTool) name() string {
	return "read_file"
}

func (t *readFileTool) definition() openai.ChatCompletionToolParam {
	return openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        "read_file",
			Description: openai.String("Read a file from disk"),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path to the file on disk.",
					},
					"max_bytes": map[string]any{
						"type":        "integer",
						"description": "Maximum bytes to read (defaults to tool limit).",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

func (t *readFileTool) execute(argText string) (string, error) {
	var args struct {
		Path     string `json:"path"`
		MaxBytes int64  `json:"max_bytes"`
	}
	if err := json.Unmarshal([]byte(argText), &args); err != nil {
		t.ctx.debugf("[verbose] read_file: failed to parse arguments: %v", err)
		return marshalToolResponse("read_file", nil, err)
	}
	t.ctx.debugf("[verbose] read_file: path=%s, max_bytes=%d", args.Path, args.MaxBytes)
	if args.Path == "" {
		return marshalToolResponse("read_file", nil, errors.New("path is required"))
	}

	// Validate and sanitize path
	validatedPath, err := validatePathWithAllowedDirs(args.Path, t.ctx.AllowedDirs)
	if err != nil {
		t.ctx.debugf("[verbose] read_file: path validation failed: %v", err)
		return marshalToolResponse("read_file", nil, fmt.Errorf("path validation failed: %w", err))
	}

	// Check if file exists and is not a directory
	if err := validateFileExists(validatedPath); err != nil {
		t.ctx.debugf("[verbose] read_file: file validation failed: %v", err)
		return marshalToolResponse("read_file", nil, err)
	}

	info, err := os.Stat(validatedPath)
	if err != nil {
		t.ctx.debugf("[verbose] read_file: stat failed: %v", err)
		return marshalToolResponse("read_file", nil, err)
	}

	t.ctx.debugf("[verbose] read_file: file size=%d bytes", info.Size())

	maxBytes := args.MaxBytes
	if maxBytes <= 0 {
		maxBytes = t.ctx.MaxReadBytes
	}
	if maxBytes <= 0 {
		return marshalToolResponse("read_file", nil, errors.New("max_bytes must be greater than 0"))
	}

	file, err := os.Open(validatedPath)
	if err != nil {
		t.ctx.debugf("[verbose] read_file: open failed: %v", err)
		return marshalToolResponse("read_file", nil, err)
	}
	defer func() { _ = file.Close() }()

	limitedReader := io.LimitReader(file, maxBytes+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		t.ctx.debugf("[verbose] read_file: read failed: %v", err)
		return marshalToolResponse("read_file", nil, err)
	}

	truncated := false
	if int64(len(data)) > maxBytes {
		truncated = true
		originalLen := len(data)
		data = data[:maxBytes]
		t.ctx.debugf("[verbose] read_file: truncated from %d to %d bytes", originalLen, maxBytes)
	}

	result := struct {
		Path      string `json:"path"`
		Bytes     int    `json:"bytes"`
		Truncated bool   `json:"truncated"`
		Content   string `json:"content"`
	}{
		Path:      validatedPath,
		Bytes:     len(data),
		Truncated: truncated,
		Content:   string(data),
	}
	t.ctx.debugf("[verbose] read_file: success, read %d bytes (truncated=%v)", result.Bytes, truncated)
	return marshalToolResponse("read_file", result, nil)
}

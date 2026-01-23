// ReadFileTool implementation.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/openai/openai-go"
)

// ReadFileTool implements the read_file tool.
type ReadFileTool struct {
	ctx ToolContext
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Definition() openai.ChatCompletionToolParam {
	return openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        "read_file",
			Description: openai.String("Read a file from disk"),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type": "string",
					},
					"max_bytes": map[string]any{
						"type": "integer",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

func (t *ReadFileTool) Execute(argText string) (string, error) {
	var args struct {
		Path     string `json:"path"`
		MaxBytes int64  `json:"max_bytes"`
	}
	if err := json.Unmarshal([]byte(argText), &args); err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] read_file: failed to parse arguments: %v", err)
		}
		return marshalToolResponse("read_file", nil, err)
	}
	if t.ctx.Verbose {
		log.Printf("[verbose] read_file: path=%s, max_bytes=%d", args.Path, args.MaxBytes)
	}
	if args.Path == "" {
		return marshalToolResponse("read_file", nil, errors.New("path is required"))
	}

	// Validate and sanitize path
	validatedPath, err := validatePath(args.Path, t.ctx.AllowedDir)
	if err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] read_file: path validation failed: %v", err)
		}
		return marshalToolResponse("read_file", nil, fmt.Errorf("path validation failed: %w", err))
	}

	// Check if file exists and is not a directory
	if err := validateFileExists(validatedPath); err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] read_file: file validation failed: %v", err)
		}
		return marshalToolResponse("read_file", nil, err)
	}

	info, err := os.Stat(validatedPath)
	if err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] read_file: stat failed: %v", err)
		}
		return marshalToolResponse("read_file", nil, err)
	}

	if t.ctx.Verbose {
		log.Printf("[verbose] read_file: file size=%d bytes", info.Size())
	}

	data, err := os.ReadFile(validatedPath)
	if err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] read_file: read failed: %v", err)
		}
		return marshalToolResponse("read_file", nil, err)
	}

	maxBytes := args.MaxBytes
	if maxBytes <= 0 {
		maxBytes = t.ctx.MaxReadBytes
	}

	truncated := false
	if int64(len(data)) > maxBytes {
		truncated = true
		data = data[:maxBytes]
		if t.ctx.Verbose {
			log.Printf("[verbose] read_file: truncated from %d to %d bytes", len(data), maxBytes)
		}
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
	if t.ctx.Verbose {
		log.Printf("[verbose] read_file: success, read %d bytes (truncated=%v)", result.Bytes, truncated)
	}
	return marshalToolResponse("read_file", result, nil)
}

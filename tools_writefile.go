// WriteFileTool implementation.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/openai/openai-go"
)

// WriteFileTool implements the write_file tool.
type WriteFileTool struct {
	ctx ToolContext
}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Definition() openai.ChatCompletionToolParam {
	return openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        "write_file",
			Description: openai.String("Write content to a file on disk"),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type": "string",
					},
					"content": map[string]any{
						"type": "string",
					},
					"overwrite": map[string]any{
						"type": "boolean",
					},
				},
				"required": []string{"path", "content"},
			},
		},
	}
}

func (t *WriteFileTool) Execute(argText string) (string, error) {
	var args struct {
		Path      string `json:"path"`
		Content   string `json:"content"`
		Overwrite bool   `json:"overwrite"`
	}
	if err := json.Unmarshal([]byte(argText), &args); err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] write_file: failed to parse arguments: %v", err)
		}
		return marshalToolResponse("write_file", nil, err)
	}
	if t.ctx.Verbose {
		log.Printf("[verbose] write_file: path=%s, bytes=%d, overwrite=%v", args.Path, len(args.Content), args.Overwrite)
	}
	if args.Path == "" {
		return marshalToolResponse("write_file", nil, errors.New("path is required"))
	}

	// Validate and sanitize path
	validatedPath, err := validatePath(args.Path, t.ctx.AllowedDir)
	if err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] write_file: path validation failed: %v", err)
		}
		return marshalToolResponse("write_file", nil, fmt.Errorf("path validation failed: %w", err))
	}

	if !args.Overwrite {
		if _, err := os.Stat(validatedPath); err == nil {
			if t.ctx.Verbose {
				log.Printf("[verbose] write_file: file already exists and overwrite=false")
			}
			return marshalToolResponse("write_file", nil, fmt.Errorf("file exists: %s", validatedPath))
		}
	}

	dir := filepath.Dir(validatedPath)
	if dir != "." && dir != "" {
		if t.ctx.Verbose {
			log.Printf("[verbose] write_file: creating directory: %s", dir)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			if t.ctx.Verbose {
				log.Printf("[verbose] write_file: mkdir failed: %v", err)
			}
			return marshalToolResponse("write_file", nil, err)
		}
	}

	if err := os.WriteFile(validatedPath, []byte(args.Content), 0o644); err != nil {
		if t.ctx.Verbose {
			log.Printf("[verbose] write_file: write failed: %v", err)
		}
		return marshalToolResponse("write_file", nil, err)
	}

	result := struct {
		Path  string `json:"path"`
		Bytes int    `json:"bytes"`
	}{
		Path:  validatedPath,
		Bytes: len(args.Content),
	}
	if t.ctx.Verbose {
		log.Printf("[verbose] write_file: success, wrote %d bytes", result.Bytes)
	}
	return marshalToolResponse("write_file", result, nil)
}

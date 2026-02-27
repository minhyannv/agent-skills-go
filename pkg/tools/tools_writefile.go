package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/openai/openai-go"
)

type writeFileTool struct {
	ctx Context
}

func (t *writeFileTool) name() string {
	return "write_file"
}

func (t *writeFileTool) definition() openai.ChatCompletionToolParam {
	return openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        "write_file",
			Description: openai.String("Write content to a file on disk"),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path to write the file to.",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Full file contents to write.",
					},
					"overwrite": map[string]any{
						"type":        "boolean",
						"description": "Whether to overwrite if the file already exists.",
					},
				},
				"required": []string{"path", "content"},
			},
		},
	}
}

func (t *writeFileTool) execute(argText string) (string, error) {
	var args struct {
		Path      string `json:"path"`
		Content   string `json:"content"`
		Overwrite bool   `json:"overwrite"`
	}
	if err := json.Unmarshal([]byte(argText), &args); err != nil {
		t.ctx.debugf("[verbose] write_file: failed to parse arguments: %v", err)
		return marshalToolResponse("write_file", nil, err)
	}
	t.ctx.debugf("[verbose] write_file: path=%s, bytes=%d, overwrite=%v", args.Path, len(args.Content), args.Overwrite)
	if args.Path == "" {
		return marshalToolResponse("write_file", nil, errors.New("path is required"))
	}

	// Validate and sanitize path
	validatedPath, err := validatePathWithAllowedDirs(args.Path, t.ctx.AllowedDirs)
	if err != nil {
		t.ctx.debugf("[verbose] write_file: path validation failed: %v", err)
		return marshalToolResponse("write_file", nil, fmt.Errorf("path validation failed: %w", err))
	}

	if !args.Overwrite {
		if _, err := os.Stat(validatedPath); err == nil {
			t.ctx.debugf("[verbose] write_file: file already exists and overwrite=false")
			return marshalToolResponse("write_file", nil, fmt.Errorf("file exists: %s", validatedPath))
		}
	}

	dir := filepath.Dir(validatedPath)
	if dir != "." && dir != "" {
		t.ctx.debugf("[verbose] write_file: creating directory: %s", dir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.ctx.debugf("[verbose] write_file: mkdir failed: %v", err)
			return marshalToolResponse("write_file", nil, err)
		}
	}

	if err := os.WriteFile(validatedPath, []byte(args.Content), 0o644); err != nil {
		t.ctx.debugf("[verbose] write_file: write failed: %v", err)
		return marshalToolResponse("write_file", nil, err)
	}

	result := struct {
		Path  string `json:"path"`
		Bytes int    `json:"bytes"`
	}{
		Path:  validatedPath,
		Bytes: len(args.Content),
	}
	t.ctx.debugf("[verbose] write_file: success, wrote %d bytes", result.Bytes)
	return marshalToolResponse("write_file", result, nil)
}

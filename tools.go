// Tool interface and base implementations.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/openai/openai-go"
)

// defaultMaxReadBytes caps read_file output for safety.
const defaultMaxReadBytes int64 = 1024 * 1024

// Tool represents a tool that can be called by the model.
type Tool interface {
	// Definition returns the tool definition for OpenAI API.
	Definition() openai.ChatCompletionToolParam
	// Execute executes the tool with the given arguments.
	Execute(argText string) (string, error)
	// Name returns the tool name.
	Name() string
}

// ToolContext provides shared context for all tools.
type ToolContext struct {
	MaxReadBytes int64
	Verbose      bool
	// AllowedDirs restricts file operations to one of these directories.
	// When empty, no restriction is applied.
	AllowedDirs []string
	Ctx         context.Context
}

// Tools holds a collection of tools and provides execution.
type Tools struct {
	tools  map[string]Tool
	ctx    ToolContext
	params []openai.ChatCompletionToolParam
}

// toolResponse is the wrapper sent back to the model after tool execution.
type toolResponse struct {
	OK   bool        `json:"ok"`
	Tool string      `json:"tool,omitempty"`
	Data interface{} `json:"data,omitempty"`
	Err  string      `json:"error,omitempty"`
}

// NewTools creates a new Tools collection with all built-in tools.
func NewTools(ctx ToolContext) *Tools {
	t := &Tools{
		tools: make(map[string]Tool),
		ctx:   ctx,
	}

	// Register all built-in tools
	readFileTool := &ReadFileTool{ctx: ctx}
	writeFileTool := &WriteFileTool{ctx: ctx}
	runShellTool := &RunShellTool{ctx: ctx}
	runPythonTool := &RunPythonTool{ctx: ctx}
	runGoTool := &RunGoTool{ctx: ctx}

	t.Register(readFileTool)
	t.Register(writeFileTool)
	t.Register(runShellTool)
	t.Register(runPythonTool)
	t.Register(runGoTool)

	return t
}

// Register adds a tool to the collection.
func (t *Tools) Register(tool Tool) {
	t.tools[tool.Name()] = tool
	t.params = append(t.params, tool.Definition())
}

// Definitions returns all tool definitions for OpenAI API.
func (t *Tools) Definitions() []openai.ChatCompletionToolParam {
	return t.params
}

// Execute executes a tool call by name.
func (t *Tools) Execute(call openai.ChatCompletionMessageToolCall) (string, error) {
	// Check context cancellation
	if t.ctx.Ctx != nil {
		select {
		case <-t.ctx.Ctx.Done():
			return marshalToolResponse(call.Function.Name, nil, t.ctx.Ctx.Err())
		default:
		}
	}

	tool, ok := t.tools[call.Function.Name]
	if !ok {
		return marshalToolResponse(call.Function.Name, nil, fmt.Errorf("unknown tool: %s", call.Function.Name))
	}

	if t.ctx.Verbose {
		log.Printf("[verbose] Executing tool: %s", call.Function.Name)
	}

	return tool.Execute(call.Function.Arguments)
}

// marshalToolResponse encodes a tool response as JSON.
func marshalToolResponse(tool string, data interface{}, err error) (string, error) {
	resp := toolResponse{
		OK:   err == nil,
		Tool: tool,
		Data: data,
	}
	if err != nil {
		resp.Err = err.Error()
	}
	payload, marshalErr := json.Marshal(resp)
	if marshalErr != nil {
		return "", marshalErr
	}
	return string(payload), nil
}

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

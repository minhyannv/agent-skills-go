package agentskills

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go"
)

const defaultMaxReadBytes int64 = 1024 * 1024

type tool interface {
	definition() openai.ChatCompletionToolParam
	execute(argText string) (string, error)
	name() string
}

type toolContext struct {
	MaxReadBytes int64
	Verbose      bool
	AllowedDirs  []string
	Ctx          context.Context
	Logger       Logger
}

func (c toolContext) debugf(format string, args ...any) {
	debugf(c.Verbose, c.Logger, format, args...)
}

type tools struct {
	registry map[string]tool
	ctx      toolContext
	params   []openai.ChatCompletionToolParam
}

type toolResponse struct {
	OK   bool        `json:"ok"`
	Tool string      `json:"tool,omitempty"`
	Data interface{} `json:"data,omitempty"`
	Err  string      `json:"error,omitempty"`
}

func newTools(ctx toolContext) *tools {
	t := &tools{
		registry: make(map[string]tool),
		ctx:      ctx,
	}

	t.register(&readFileTool{ctx: ctx})
	t.register(&writeFileTool{ctx: ctx})
	t.register(&runShellTool{ctx: ctx})
	return t
}

func (t *tools) register(toolImpl tool) {
	t.registry[toolImpl.name()] = toolImpl
	t.params = append(t.params, toolImpl.definition())
	t.ctx.debugf("[verbose] registered tool: %s", toolImpl.name())
}

func (t *tools) definitions() []openai.ChatCompletionToolParam {
	return t.params
}

func (t *tools) execute(call openai.ChatCompletionMessageToolCall) (string, error) {
	if t.ctx.Ctx != nil {
		select {
		case <-t.ctx.Ctx.Done():
			return marshalToolResponse(call.Function.Name, nil, t.ctx.Ctx.Err())
		default:
		}
	}

	toolImpl, ok := t.registry[call.Function.Name]
	if !ok {
		return marshalToolResponse(call.Function.Name, nil, fmt.Errorf("unknown tool: %s", call.Function.Name))
	}

	return toolImpl.execute(call.Function.Arguments)
}

func marshalToolResponse(toolName string, data interface{}, err error) (string, error) {
	resp := toolResponse{
		OK:   err == nil,
		Tool: toolName,
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

package tools

import (
	"context"
	"encoding/json"
	"fmt"

	loggerpkg "github.com/minhyannv/agent-skills-go/pkg/logger"
	"github.com/openai/openai-go"
)

const DefaultMaxReadBytes int64 = 1024 * 1024

type tool interface {
	definition() openai.ChatCompletionToolParam
	execute(argText string) (string, error)
	name() string
}

type Context struct {
	MaxReadBytes int64
	Verbose      bool
	AllowedDirs  []string
	Ctx          context.Context
	Logger       loggerpkg.Logger
}

func (c Context) debugf(format string, args ...any) {
	loggerpkg.Debugf(c.Verbose, c.Logger, format, args...)
}

// Registry holds registered tools and handles execution.
type Registry struct {
	registry map[string]tool
	ctx      Context
	params   []openai.ChatCompletionToolParam
}

type toolResponse struct {
	OK   bool        `json:"ok"`
	Tool string      `json:"tool,omitempty"`
	Data interface{} `json:"data,omitempty"`
	Err  string      `json:"error,omitempty"`
}

// New builds a registry with the built-in tools.
func New(ctx Context) *Registry {
	if ctx.Logger == nil {
		ctx.Logger = loggerpkg.NopLogger{}
	}
	t := &Registry{
		registry: make(map[string]tool),
		ctx:      ctx,
	}

	t.register(&readFileTool{ctx: ctx})
	t.register(&writeFileTool{ctx: ctx})
	t.register(&runShellTool{ctx: ctx})
	return t
}

func (t *Registry) register(toolImpl tool) {
	t.registry[toolImpl.name()] = toolImpl
	t.params = append(t.params, toolImpl.definition())
	t.ctx.debugf("[verbose] registered tool: %s", toolImpl.name())
}

func (t *Registry) Definitions() []openai.ChatCompletionToolParam {
	return t.params
}

func (t *Registry) Execute(call openai.ChatCompletionMessageToolCall) (string, error) {
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

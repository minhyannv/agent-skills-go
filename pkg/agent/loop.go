package agent

import (
	"context"
	"errors"
	"fmt"
	configpkg "github.com/minhyannv/agent-skills-go/pkg/config"
	"github.com/minhyannv/agent-skills-go/pkg/prompt"
	"github.com/minhyannv/agent-skills-go/pkg/skills"
	"github.com/minhyannv/agent-skills-go/pkg/tools"
	"github.com/openai/openai-go/option"
	"path/filepath"
	"strings"

	loggerpkg "github.com/minhyannv/agent-skills-go/pkg/logger"
	"github.com/openai/openai-go"
)

// AgentLoop holds agent runtime state.
type AgentLoop struct {
	config       configpkg.Config
	client       openai.Client
	tools        *tools.Registry
	SystemPrompt string
	history      []openai.ChatCompletionMessageParamUnion

	ctx     context.Context
	logger  loggerpkg.Logger
	verbose bool
}

// New initializes an AgentLoop with the provided context, config, and dependencies.
func New(ctx context.Context, cfg configpkg.Config, opts ...AgentOption) (*AgentLoop, error) {
	cfg = configpkg.Normalize(cfg)
	deps := agentDeps{logger: loggerpkg.NopLogger{}}
	for _, opt := range opts {
		if opt != nil {
			opt(&deps)
		}
	}

	loggerpkg.Debug(cfg.Verbose, deps.logger, "agent_loop init", map[string]any{
		"skills_dirs": cfg.SkillsDirs,
		"max_turns":   cfg.MaxTurns,
		"allowed_dir": cfg.AllowedDir,
		"model":       cfg.Model,
		"base_url":    cfg.BaseURL,
	})
	if cfg.APIKey == "" {
		return nil, errors.New("APIKey is not set")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return nil, errors.New("Model is not set")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	loggerpkg.Debug(cfg.Verbose, deps.logger, "loading skills", map[string]any{
		"skills_dirs": cfg.SkillsDirs,
	})
	skillList, err := skills.LoadFromDirs(cfg.SkillsDirs)
	if err != nil {
		return nil, fmt.Errorf("load skills: %w", err)
	}
	if cfg.Verbose {
		loggerpkg.Debug(cfg.Verbose, deps.logger, "skills loaded", map[string]any{
			"count": len(skillList),
		})
		for _, skill := range skillList {
			loggerpkg.Debug(cfg.Verbose, deps.logger, "skill discovered", map[string]any{
				"name":        skill.Name,
				"path":        skill.SkillFilePath,
				"description": skill.Description,
			})
		}
	}

	systemPrompt := prompt.BuildSystemPrompt(skillList)
	if strings.TrimSpace(systemPrompt) == "" {
		return nil, errors.New("system prompt is empty")
	}
	loggerpkg.Debug(cfg.Verbose, deps.logger, "system prompt ready", map[string]any{
		"bytes": len(systemPrompt),
	})

	client := newOpenAIClient(cfg)

	allowedDirs := []string{}
	if cfg.AllowedDir != "" {
		allowedDirs = append(allowedDirs, cfg.AllowedDir)
		for _, dir := range cfg.SkillsDirs {
			if abs, err := filepath.Abs(dir); err == nil {
				allowedDirs = append(allowedDirs, abs)
			} else {
				allowedDirs = append(allowedDirs, dir)
			}
		}
	}
	loggerpkg.Debug(cfg.Verbose, deps.logger, "allowed dirs resolved", map[string]any{
		"allowed_dirs": allowedDirs,
	})

	toolCtx := tools.Context{
		MaxReadBytes: tools.DefaultMaxReadBytes,
		Verbose:      cfg.Verbose,
		AllowedDirs:  allowedDirs,
		Ctx:          ctx,
		Logger:       deps.logger,
	}
	registeredTools := tools.New(toolCtx)
	loggerpkg.Debug(cfg.Verbose, deps.logger, "tools registered", map[string]any{
		"count": len(registeredTools.Definitions()),
	})

	return &AgentLoop{
		config:       cfg,
		client:       client,
		tools:        registeredTools,
		SystemPrompt: systemPrompt,
		history:      []openai.ChatCompletionMessageParamUnion{openai.SystemMessage(systemPrompt)},

		ctx:     ctx,
		logger:  deps.logger,
		verbose: cfg.Verbose,
	}, nil
}

func newOpenAIClient(cfg configpkg.Config) openai.Client {
	opts := []option.RequestOption{}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}
	return openai.NewClient(opts...)
}

// runOnce performs one model completion request.
func (a *AgentLoop) runOnce(params openai.ChatCompletionNewParams) (openai.ChatCompletionMessage, error) {
	a.debugf("[verbose] iteration: sending request")
	completion, err := a.client.Chat.Completions.New(a.ctx, params)
	if err != nil {
		return openai.ChatCompletionMessage{}, err
	}
	if len(completion.Choices) == 0 {
		return openai.ChatCompletionMessage{}, errors.New("empty completion choices")
	}
	return completion.Choices[0].Message, nil
}

// runIteration executes iterative model/tool turns for one user interaction.
func (a *AgentLoop) runIteration(
	messages []openai.ChatCompletionMessageParamUnion,
	maxTurns int,
) (openai.ChatCompletionMessage, error) {
	currentMessages := append([]openai.ChatCompletionMessageParamUnion{}, messages...)

	for turn := 0; turn < maxTurns; turn++ {
		a.debugf("[verbose] iteration: %d/%d", turn+1, maxTurns)
		message, err := a.runOnce(a.newChatParams(currentMessages))
		if err != nil {
			return openai.ChatCompletionMessage{}, err
		}

		if len(message.ToolCalls) == 0 {
			return message, nil
		}

		// Persist the assistant tool-call turn before appending tool responses.
		currentMessages = append(currentMessages, message.ToParam())
		a.debugf("[verbose] iteration: assistant requested %d tool call(s)", len(message.ToolCalls))
		currentMessages = a.appendToolResponses(currentMessages, message.ToolCalls)
	}

	return openai.ChatCompletionMessage{}, errors.New("max turns reached before assistant produced a final response")
}

// Run processes one user input and returns a single final assistant message.
// Conversation state is persisted inside AgentLoop and can be reset via Reset.
func (a *AgentLoop) Run(userInput string) (openai.ChatCompletionMessage, error) {
	userInput = strings.TrimSpace(userInput)
	if userInput == "" {
		return openai.ChatCompletionMessage{}, errors.New("user input is required")
	}
	previousLen := len(a.history)
	a.history = append(a.history, openai.UserMessage(userInput))

	finalMessage, err := a.runIteration(a.history, a.config.MaxTurns)
	if err != nil {
		a.history = a.history[:previousLen]
		return openai.ChatCompletionMessage{}, err
	}

	a.history = append(a.history, finalMessage.ToParam())
	return finalMessage, nil
}

// Reset clears conversation history and keeps only the system prompt.
func (a *AgentLoop) Reset() {
	a.history = []openai.ChatCompletionMessageParamUnion{openai.SystemMessage(a.SystemPrompt)}
}

func (a *AgentLoop) debugf(format string, args ...any) {
	loggerpkg.Debugf(a.verbose, a.logger, format, args...)
}

func (a *AgentLoop) newChatParams(messages []openai.ChatCompletionMessageParamUnion) openai.ChatCompletionNewParams {
	return openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(a.config.Model),
		Messages: messages,
		Tools:    a.tools.Definitions(),
	}
}

func (a *AgentLoop) appendToolResponses(
	messages []openai.ChatCompletionMessageParamUnion,
	toolCalls []openai.ChatCompletionMessageToolCall,
) []openai.ChatCompletionMessageParamUnion {
	updated := messages
	for _, call := range toolCalls {
		output, err := a.tools.Execute(call)
		if err != nil {
			output = fmt.Sprintf(`{"ok":false,"error":%q}`, err.Error())
		}
		updated = append(updated, openai.ToolMessage(output, call.ID))
	}
	return updated
}

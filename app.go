// Application initialization and setup.
package main

import (
	"context"
	"log"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// App holds the application state and dependencies.
type App struct {
	Config       *Config
	Client       openai.Client
	Tools        *Tools
	SystemPrompt string
	Ctx          context.Context
}

// NewApp initializes and returns a new App instance.
func NewApp(config *Config) (*App, error) {
	// Validate API key
	if config.OpenAIAPIKey == "" {
		log.Fatal("OPENAI_API_KEY is not set")
	}

	// Load skills
	skills, err := LoadSkillsFromDir(config.SkillsDir)
	if err != nil {
		log.Fatalf("load skills: %v", err)
	}

	// Build system prompt
	systemPrompt := BuildSystemPrompt(skills)
	if strings.TrimSpace(systemPrompt) == "" {
		log.Fatal("system prompt is empty")
	}

	// Initialize OpenAI client
	client := newOpenAIClient(config)

	// Create context
	ctx := context.Background()

	// Create tool context
	toolCtx := ToolContext{
		MaxReadBytes: defaultMaxReadBytes,
		Verbose:      config.Verbose,
		AllowedDir:   config.AllowedDir,
		Ctx:          ctx,
	}

	// Build tools
	tools := NewTools(toolCtx)

	return &App{
		Config:       config,
		Client:       client,
		Tools:        tools,
		SystemPrompt: systemPrompt,
		Ctx:          ctx,
	}, nil
}

// newOpenAIClient builds a client with configuration from Config.
func newOpenAIClient(config *Config) openai.Client {
	opts := []option.RequestOption{}
	if config.OpenAIBaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.OpenAIBaseURL))
	}
	if config.OpenAIAPIKey != "" {
		opts = append(opts, option.WithAPIKey(config.OpenAIAPIKey))
	}
	return openai.NewClient(opts...)
}

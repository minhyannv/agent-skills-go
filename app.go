// Application initialization and setup.
package main

import (
	"context"
	"log"
	"path/filepath"
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
	skills, err := LoadSkillsFromDirs(config.SkillsDirs)
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
	allowedDirs := []string{}
	if strings.TrimSpace(config.AllowedDir) != "" {
		allowedDirs = append(allowedDirs, config.AllowedDir)
	}
	if strings.TrimSpace(config.AllowedDir) != "" {
		// When -allowed_dir is set, also allow the skills directories so the model can
		// read SKILL.md and run scripts shipped with skills.
		for _, dir := range config.SkillsDirs {
			if strings.TrimSpace(dir) == "" {
				continue
			}
			if abs, err := filepath.Abs(dir); err == nil {
				allowedDirs = append(allowedDirs, abs)
			} else {
				allowedDirs = append(allowedDirs, dir)
			}
		}
	}
	toolCtx := ToolContext{
		MaxReadBytes: defaultMaxReadBytes,
		Verbose:      config.Verbose,
		AllowedDirs:  allowedDirs,
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

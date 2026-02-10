// Application initialization and setup.
package main

import (
	"context"
	"errors"
	"fmt"
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
	if config.Verbose {
		log.Printf("[verbose] app init: skills_dirs=%v max_turns=%d stream=%v allowed_dir=%s model=%s base_url=%s", config.SkillsDirs, config.MaxTurns, config.Stream, config.AllowedDir, config.OpenAIModel, config.OpenAIBaseURL)
	}
	// Validate API key
	if config.OpenAIAPIKey == "" {
		return nil, errors.New("OPENAI_API_KEY is not set")
	}
	if strings.TrimSpace(config.OpenAIModel) == "" {
		return nil, errors.New("OPENAI_MODEL is not set")
	}

	// Load skills
	if config.Verbose {
		log.Printf("[verbose] loading skills")
	}
	skills, err := LoadSkillsFromDirs(config.SkillsDirs)
	if err != nil {
		return nil, fmt.Errorf("load skills: %w", err)
	}
	if config.Verbose {
		log.Printf("[verbose] loaded %d skill(s)", len(skills))
		for _, skill := range skills {
			log.Printf("[verbose] skill: name=%s path=%s description=%s", skill.Name, skill.SkillFilePath, skill.Description)
		}
	}

	// Build system prompt
	systemPrompt := BuildSystemPrompt(skills)
	if strings.TrimSpace(systemPrompt) == "" {
		return nil, errors.New("system prompt is empty")
	}
	if config.Verbose {
		log.Printf("[verbose] system prompt bytes=%d", len(systemPrompt))
	}

	// Initialize OpenAI client
	client := newOpenAIClient(config)

	// Create context
	ctx := context.Background()

	// Create tool context
	allowedDirs := []string{}
	if strings.TrimSpace(config.AllowedDir) != "" {
		allowedDirs = append(allowedDirs, config.AllowedDir)
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
	if config.Verbose {
		log.Printf("[verbose] allowed_dirs=%v", allowedDirs)
	}
	toolCtx := ToolContext{
		MaxReadBytes: defaultMaxReadBytes,
		Verbose:      config.Verbose,
		AllowedDirs:  allowedDirs,
		Ctx:          ctx,
	}

	// Build tools
	tools := NewTools(toolCtx)
	if config.Verbose {
		log.Printf("[verbose] tools registered=%d", len(tools.Definitions()))
	}

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

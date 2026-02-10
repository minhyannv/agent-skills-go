package agentskills

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// App holds agent runtime state.
type App struct {
	config       Config
	client       openai.Client
	tools        *tools
	systemPrompt string
	ctx          context.Context
	logger       Logger
	verbose      bool
}

// New initializes an App with the provided context and config.
func New(ctx context.Context, cfg Config) (*App, error) {
	cfg = normalizeConfig(cfg)
	debugf(cfg.Verbose, cfg.Logger, "[verbose] app init: skills_dirs=%v max_turns=%d stream=%v allowed_dir=%s model=%s base_url=%s", cfg.SkillsDirs, cfg.MaxTurns, cfg.Stream, cfg.AllowedDir, cfg.Model, cfg.BaseURL)
	if cfg.APIKey == "" {
		return nil, errors.New("APIKey is not set")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return nil, errors.New("Model is not set")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	debugf(cfg.Verbose, cfg.Logger, "[verbose] loading skills")
	skills, err := loadSkillsFromDirs(cfg.SkillsDirs)
	if err != nil {
		return nil, fmt.Errorf("load skills: %w", err)
	}
	if cfg.Verbose {
		debugf(cfg.Verbose, cfg.Logger, "[verbose] loaded %d skill(s)", len(skills))
		for _, skill := range skills {
			debugf(cfg.Verbose, cfg.Logger, "[verbose] skill: name=%s path=%s description=%s", skill.Name, skill.SkillFilePath, skill.Description)
		}
	}

	systemPrompt := buildSystemPrompt(skills)
	if strings.TrimSpace(systemPrompt) == "" {
		return nil, errors.New("system prompt is empty")
	}
	debugf(cfg.Verbose, cfg.Logger, "[verbose] system prompt bytes=%d", len(systemPrompt))

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
	debugf(cfg.Verbose, cfg.Logger, "[verbose] allowed_dirs=%v", allowedDirs)

	toolCtx := toolContext{
		MaxReadBytes: defaultMaxReadBytes,
		Verbose:      cfg.Verbose,
		AllowedDirs:  allowedDirs,
		Ctx:          ctx,
		Logger:       cfg.Logger,
	}
	registeredTools := newTools(toolCtx)
	debugf(cfg.Verbose, cfg.Logger, "[verbose] tools registered=%d", len(registeredTools.definitions()))

	return &App{
		config:       cfg,
		client:       client,
		tools:        registeredTools,
		systemPrompt: systemPrompt,
		ctx:          ctx,
		logger:       cfg.Logger,
		verbose:      cfg.Verbose,
	}, nil
}

func newOpenAIClient(cfg Config) openai.Client {
	opts := []option.RequestOption{}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}
	return openai.NewClient(opts...)
}

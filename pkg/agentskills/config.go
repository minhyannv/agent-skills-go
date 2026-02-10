package agentskills

import (
	"os"
	"strings"
)

// Config holds all runtime configuration for the agent.
type Config struct {
	SkillsDirs []string
	MaxTurns   int
	Stream     bool
	Verbose    bool
	AllowedDir string
	Logger     Logger

	APIKey  string
	BaseURL string
	Model   string
}

// DefaultConfig returns a baseline configuration without side effects.
func DefaultConfig() Config {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	return Config{
		SkillsDirs: nil,
		MaxTurns:   10,
		Stream:     false,
		Verbose:    false,
		AllowedDir: wd,
		Logger:     NopLogger{},
	}
}

func normalizeConfig(cfg Config) Config {
	cfg.AllowedDir = strings.TrimSpace(cfg.AllowedDir)
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	cfg.Model = strings.TrimSpace(cfg.Model)
	if cfg.Logger == nil {
		cfg.Logger = NopLogger{}
	}

	normalizedSkills := make([]string, 0, len(cfg.SkillsDirs))
	for _, dir := range cfg.SkillsDirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		normalizedSkills = append(normalizedSkills, dir)
	}
	cfg.SkillsDirs = normalizedSkills

	if cfg.MaxTurns <= 0 {
		cfg.MaxTurns = 1
	}
	return cfg
}

// Configuration management for the application.
package main

import (
	"flag"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application configuration from environment variables and command-line flags.
type Config struct {
	// Command-line flags
	SkillsDirs []string
	MaxTurns   int
	Stream     bool
	Verbose    bool
	AllowedDir string

	// Environment variables
	OpenAIAPIKey  string
	OpenAIBaseURL string
	OpenAIModel   string
}

// ParseConfig parses command-line flags and environment variables to create a Config.
func ParseConfig() *Config {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Read environment variables
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	baseURL := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	model := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))

	// Parse command-line flags
	var (
		skillsDirs = flag.String("skills_dirs", "./skills", "Comma-separated list of directories containing skills")
		maxTurns   = flag.Int("max_turns", 10, "Max tool-call turns")
		stream     = flag.Bool("stream", false, "Stream assistant output")
		verbose    = flag.Bool("verbose", false, "Verbose tool-call logging")
		allowedDir = flag.String("allowed_dir", "", "Base directory for file operations (empty = no restriction, recommended for security)")
	)
	flag.Parse()

	return &Config{
		SkillsDirs:    parseSkillsDirs(*skillsDirs),
		MaxTurns:      *maxTurns,
		Stream:        *stream,
		Verbose:       *verbose,
		AllowedDir:    *allowedDir,
		OpenAIAPIKey:  apiKey,
		OpenAIBaseURL: baseURL,
		OpenAIModel:   model,
	}
}

func parseSkillsDirs(value string) []string {
	parts := strings.Split(value, ",")
	dirs := make([]string, 0, len(parts))
	for _, part := range parts {
		dir := strings.TrimSpace(part)
		if dir == "" {
			continue
		}
		dirs = append(dirs, dir)
	}
	return dirs
}

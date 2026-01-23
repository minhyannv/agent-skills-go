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
	SkillsDir  string
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
		skillsDir  = flag.String("skills_dir", "examples/skills", "Directory containing skills")
		maxTurns   = flag.Int("max_turns", 20, "Max tool-call turns")
		stream     = flag.Bool("stream", false, "Stream assistant output")
		verbose    = flag.Bool("verbose", false, "Verbose tool-call logging")
		allowedDir = flag.String("allowed_dir", "", "Base directory for file operations (empty = no restriction, recommended for security)")
	)
	flag.Parse()

	return &Config{
		SkillsDir:     *skillsDir,
		MaxTurns:      *maxTurns,
		Stream:        *stream,
		Verbose:       *verbose,
		AllowedDir:    *allowedDir,
		OpenAIAPIKey:  apiKey,
		OpenAIBaseURL: baseURL,
		OpenAIModel:   model,
	}
}

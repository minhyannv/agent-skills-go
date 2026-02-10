package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/minhyannv/agent-skills-go/pkg/agentskills"
)

func parseCLIConfig() (agentskills.Config, error) {
	_ = godotenv.Load()

	defaults := agentskills.DefaultConfig()
	skillsDirs := make(stringSliceFlag, 0, len(defaults.SkillsDirs))
	for _, dir := range defaults.SkillsDirs {
		_ = skillsDirs.Set(dir)
	}

	flag.Var(&skillsDirs, "skills_dirs", "Skill directory. Repeat this flag for multiple directories; comma-separated values are not supported")
	maxTurns := flag.Int("max_turns", defaults.MaxTurns, "Max tool-call turns")
	stream := flag.Bool("stream", defaults.Stream, "Stream assistant output")
	verbose := flag.Bool("verbose", defaults.Verbose, "Verbose tool-call logging")
	allowedDir := flag.String("allowed_dir", defaults.AllowedDir, "Base directory for file operations (set empty to disable restriction)")
	flag.Parse()

	cfg := defaults
	cfg.SkillsDirs = skillsDirs.values()
	cfg.MaxTurns = *maxTurns
	cfg.Stream = *stream
	cfg.Verbose = *verbose
	cfg.AllowedDir = strings.TrimSpace(*allowedDir)
	cfg.APIKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	cfg.BaseURL = strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	cfg.Model = strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	cfg.Logger = agentskills.NewWriterLogger(os.Stderr)
	return cfg, nil
}

type stringSliceFlag []string

func (f *stringSliceFlag) String() string {
	if f == nil {
		return ""
	}
	return strings.Join(*f, ",")
}

func (f *stringSliceFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("empty skills directory")
	}
	if strings.Contains(value, ",") {
		return fmt.Errorf("comma-separated values are not supported for -skills_dirs; repeat the flag instead")
	}
	*f = append(*f, value)
	return nil
}

func (f stringSliceFlag) values() []string {
	out := make([]string, len(f))
	copy(out, f)
	return out
}

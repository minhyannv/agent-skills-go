// Package main provides a single-file CLI for AgentLoop.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/minhyannv/agent-skills-go/pkg/agent"
	configpkg "github.com/minhyannv/agent-skills-go/pkg/config"
	loggerpkg "github.com/minhyannv/agent-skills-go/pkg/logger"
)

// main is the program entry point.
func main() {
	config, err := parseCLIConfig()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	appLogger := loggerpkg.NewWriterLogger(os.Stderr)
	app, err := agent.New(context.Background(), config, agent.WithLogger(appLogger))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := runREPL(app, replOptions{
		Verbose: config.Verbose,
		Logger:  appLogger,
	}, os.Stdin, os.Stdout); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// parseCLIConfig loads env + flags into runtime config.
func parseCLIConfig() (configpkg.Config, error) {
	_ = godotenv.Load()

	defaults := configpkg.DefaultConfig()
	defaults.SkillsDirs = discoverDefaultSkills(defaults.AllowedDir)
	skillsDirs := make(stringSliceFlag, 0, len(defaults.SkillsDirs))
	for _, dir := range defaults.SkillsDirs {
		_ = skillsDirs.Set(dir)
	}

	flag.Var(&skillsDirs, "skills_dirs", "Skill directory. Repeat this flag for multiple directories; comma-separated values are not supported")
	maxTurns := flag.Int("max_turns", defaults.MaxTurns, "Max tool-call turns")
	verbose := flag.Bool("verbose", defaults.Verbose, "Verbose tool-call logging")
	allowedDir := flag.String("allowed_dir", defaults.AllowedDir, "Base directory for file operations (set empty to disable restriction)")
	flag.Parse()

	cfg := defaults
	cfg.SkillsDirs = skillsDirs.values()
	cfg.MaxTurns = *maxTurns
	cfg.Verbose = *verbose
	cfg.AllowedDir = strings.TrimSpace(*allowedDir)
	cfg.APIKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	cfg.BaseURL = strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	cfg.Model = strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	return cfg, nil
}

func discoverDefaultSkills(baseDir string) []string {
	candidates := []string{
		filepath.Join(baseDir, "skills", ".system", "skill-creator"),
		filepath.Join(baseDir, "skills", ".system", "skill-installer"),
	}

	out := make([]string, 0, len(candidates))
	for _, dir := range candidates {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		out = append(out, dir)
	}
	return out
}

// stringSliceFlag supports repeatable -skills_dirs flags.
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

// replOptions configures REPL behavior.
type replOptions struct {
	Verbose bool
	Logger  loggerpkg.Logger
}

// runREPL starts an interactive REPL session for the given app.
func runREPL(app *agent.AgentLoop, opts replOptions, in io.Reader, out io.Writer) error {
	if app == nil {
		return fmt.Errorf("agent loop is required")
	}
	if in == nil {
		return fmt.Errorf("input reader is required")
	}
	if out == nil {
		out = io.Discard
	}

	if opts.Verbose && opts.Logger != nil {
		loggerpkg.Debug(opts.Verbose, opts.Logger, "repl start", nil)
	}

	scanner := bufio.NewScanner(in)
	printWelcome(out)

	for {
		_, _ = fmt.Fprint(out, "> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if strings.HasPrefix(input, "/") {
			handled, shouldQuit := handleCommand(input, app, out)
			if shouldQuit {
				break
			}
			if handled {
				continue
			}
		}

		finalMessage, err := app.Run(input)
		if err != nil {
			_, _ = fmt.Fprintf(out, "Error: %v\n\n", err)
			continue
		}

		_, _ = fmt.Fprintf(out, "%s\n\n", finalMessage.Content)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	return nil
}

func printWelcome(out io.Writer) {
	_, _ = fmt.Fprintln(out, "=== Agent Skills Go - Interactive Mode ===")
	_, _ = fmt.Fprintln(out, "Type your message and press Enter. Commands:")
	_, _ = fmt.Fprintln(out, "  /help  - Show this help message")
	_, _ = fmt.Fprintln(out, "  /clear - Clear conversation history")
	_, _ = fmt.Fprintln(out, "  /quit  - Exit the program")
	_, _ = fmt.Fprintln(out, "  /exit  - Exit the program")
	_, _ = fmt.Fprintln(out)
}

func handleCommand(
	input string,
	app *agent.AgentLoop,
	out io.Writer,
) (bool, bool) {
	cmd := strings.ToLower(input)
	switch cmd {
	case "/help", "/h":
		printHelp(out)
		return true, false
	case "/clear", "/c":
		app.Reset()
		_, _ = fmt.Fprintln(out, "Conversation history cleared.")
		_, _ = fmt.Fprintln(out)
		return true, false
	case "/quit", "/exit", "/q":
		_, _ = fmt.Fprintln(out, "Goodbye!")
		return true, true
	default:
		_, _ = fmt.Fprintf(out, "Unknown command: %s. Type /help for available commands.\n\n", input)
		return true, false
	}
}

func printHelp(out io.Writer) {
	_, _ = fmt.Fprintln(out, "Commands:")
	_, _ = fmt.Fprintln(out, "  /help  - Show this help message")
	_, _ = fmt.Fprintln(out, "  /clear - Clear conversation history")
	_, _ = fmt.Fprintln(out, "  /quit  - Exit the program")
	_, _ = fmt.Fprintln(out, "  /exit  - Exit the program")
	_, _ = fmt.Fprintln(out)
}

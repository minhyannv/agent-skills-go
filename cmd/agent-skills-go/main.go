// Package main wires skill discovery, tool execution, and OpenAI chat completions.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/minhyannv/agent-skills-go/pkg/agentskills"
)

// main is the program entry point.
func main() {
	config, err := parseCLIConfig()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	app, err := agentskills.New(context.Background(), config)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := runREPL(app, replOptions{
		Stream:   config.Stream,
		MaxTurns: config.MaxTurns,
		Verbose:  config.Verbose,
		Logger:   config.Logger,
	}, os.Stdin, os.Stdout); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

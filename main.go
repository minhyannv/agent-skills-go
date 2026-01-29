// Package main wires skill discovery, tool execution, and OpenAI chat completions.
package main

import (
	"log"
	"os"
)

// main is the program entry point.
func main() {
	log.SetFlags(0)

	// Parse configuration
	config := ParseConfig()

	// Initialize application
	app, err := NewApp(config)
	if err != nil {
		os.Exit(1)
	}

	// Enter interactive mode
	runInteractiveMode(app)
}

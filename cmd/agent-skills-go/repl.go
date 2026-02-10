package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/minhyannv/agent-skills-go/pkg/agentskills"
)

// replOptions configures REPL behavior.
type replOptions struct {
	Stream   bool
	MaxTurns int
	Verbose  bool
	Logger   agentskills.Logger
}

// runREPL starts an interactive REPL session for the given app.
func runREPL(app *agentskills.App, opts replOptions, in io.Reader, out io.Writer) error {
	if app == nil {
		return fmt.Errorf("app is required")
	}
	if in == nil {
		return fmt.Errorf("input reader is required")
	}
	if out == nil {
		out = io.Discard
	}

	if opts.Verbose && opts.Logger != nil {
		opts.Logger.Debugf("[verbose] repl start: stream=%v max_turns=%d", opts.Stream, opts.MaxTurns)
	}

	messages := []agentskills.Message{}
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
			handled, shouldQuit := handleCommand(input, &messages, out)
			if shouldQuit {
				break
			}
			if handled {
				continue
			}
		}

		messages = append(messages, agentskills.Message{
			Role:    agentskills.RoleUser,
			Content: input,
		})
		result, err := app.Chat(messages, agentskills.ChatOptions{
			Stream:       opts.Stream,
			StreamWriter: out,
			MaxTurns:     opts.MaxTurns,
		})
		if err != nil {
			_, _ = fmt.Fprintf(out, "Error: %v\n\n", err)
			messages = messages[:len(messages)-1]
			continue
		}

		messages = result.Messages
		if !result.Streamed {
			_, _ = fmt.Fprintf(out, "%s\n\n", result.Content)
		} else {
			_, _ = fmt.Fprintln(out)
		}
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
	messages *[]agentskills.Message,
	out io.Writer,
) (bool, bool) {
	cmd := strings.ToLower(input)
	switch cmd {
	case "/help", "/h":
		printHelp(out)
		return true, false
	case "/clear", "/c":
		*messages = nil
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

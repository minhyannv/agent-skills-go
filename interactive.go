// Interactive terminal mode for user interaction.
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/openai/openai-go"
)

// runInteractiveMode runs an interactive chat session.
func runInteractiveMode(app *App) {
	if app.Config.Verbose {
		log.Printf("[verbose] interactive mode start: model=%s stream=%v max_turns=%d", app.Config.OpenAIModel, app.Config.Stream, app.Config.MaxTurns)
	}
	// Initialize conversation history with system message
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(app.SystemPrompt),
	}

	scanner := bufio.NewScanner(os.Stdin)

	printWelcome()

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if app.Config.Verbose {
			log.Printf("[verbose] input received: bytes=%d is_command=%v messages=%d", len(input), strings.HasPrefix(input, "/"), len(messages))
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			if handleCommand(input, &messages, app.SystemPrompt) {
				continue
			}
		}

		// Add user message to history
		messages = append(messages, openai.UserMessage(input))

		// Run chat loop with current history
		updatedMessages, result, err := runInteractiveChatLoop(
			app.Ctx,
			app.Client,
			app.Config.OpenAIModel,
			messages,
			app.Tools,
			app.Config.MaxTurns,
			app.Config.Stream,
			app.Config.Verbose,
		)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Println()
			// Remove the user message on error to keep history consistent
			messages = messages[:len(messages)-1]
			continue
		}

		// Update messages with assistant response
		messages = updatedMessages
		if !result.Streamed {
			fmt.Println(result.Content)
			fmt.Println()
		} else {
			fmt.Println()
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading input: %v", err)
	}
}

// printWelcome prints the welcome message.
func printWelcome() {
	fmt.Println("=== Agent Skills Go - Interactive Mode ===")
	fmt.Println("Type your message and press Enter. Commands:")
	fmt.Println("  /help  - Show this help message")
	fmt.Println("  /clear - Clear conversation history")
	fmt.Println("  /quit  - Exit the program")
	fmt.Println("  /exit  - Exit the program")
	fmt.Println()
}

// handleCommand processes interactive commands.
// Returns true if the command was handled and the loop should continue.
func handleCommand(input string, messages *[]openai.ChatCompletionMessageParamUnion, systemPrompt string) bool {
	cmd := strings.ToLower(input)
	switch cmd {
	case "/help", "/h":
		printHelp()
		return true
	case "/clear", "/c":
		*messages = []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
		}
		fmt.Println("Conversation history cleared.")
		fmt.Println()
		return true
	case "/quit", "/exit", "/q":
		fmt.Println("Goodbye!")
		os.Exit(0)
		return true
	default:
		fmt.Printf("Unknown command: %s. Type /help for available commands.\n", input)
		fmt.Println()
		return true
	}
}

// printHelp prints the help message.
func printHelp() {
	fmt.Println("Commands:")
	fmt.Println("  /help  - Show this help message")
	fmt.Println("  /clear - Clear conversation history")
	fmt.Println("  /quit  - Exit the program")
	fmt.Println("  /exit  - Exit the program")
	fmt.Println()
}

// Chat completion and message handling.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/openai/openai-go"
)

// ChatLoopResult represents the result of a chat loop execution.
type ChatLoopResult struct {
	Content  string
	Streamed bool
}

// runChatOnce sends a single request and optionally streams deltas to stdout.
func runChatOnce(ctx context.Context, client openai.Client, params openai.ChatCompletionNewParams, stream bool, verbose bool) (openai.ChatCompletionMessage, bool, error) {
	if !stream {
		if verbose {
			log.Printf("[verbose] Sending non-streaming chat completion request")
		}
		completion, err := client.Chat.Completions.New(ctx, params)
		if err != nil {
			if verbose {
				log.Printf("[verbose] Chat completion request failed: %v", err)
			}
			return openai.ChatCompletionMessage{}, false, err
		}
		if len(completion.Choices) == 0 {
			if verbose {
				log.Printf("[verbose] Chat completion returned empty choices")
			}
			return openai.ChatCompletionMessage{}, false, errors.New("empty completion choices")
		}
		if verbose {
			log.Printf("[verbose] Chat completion received: %d choice(s), finish_reason=%s", len(completion.Choices), completion.Choices[0].FinishReason)
		}
		return completion.Choices[0].Message, false, nil
	}

	if verbose {
		log.Printf("[verbose] Sending streaming chat completion request")
	}
	streamResp := client.Chat.Completions.NewStreaming(ctx, params)
	defer streamResp.Close()

	acc := openai.ChatCompletionAccumulator{}
	streamed := false
	chunkCount := 0
	for streamResp.Next() {
		chunk := streamResp.Current()
		chunkCount++
		if !acc.AddChunk(chunk) {
			if verbose {
				log.Printf("[verbose] Failed to accumulate stream chunk %d", chunkCount)
			}
			return openai.ChatCompletionMessage{}, streamed, errors.New("failed to accumulate stream")
		}
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			if delta.Content != "" {
				_, _ = io.WriteString(os.Stdout, delta.Content)
				streamed = true
			}
		}
	}
	if err := streamResp.Err(); err != nil {
		if verbose {
			log.Printf("[verbose] Streaming error after %d chunks: %v", chunkCount, err)
		}
		return openai.ChatCompletionMessage{}, streamed, err
	}
	if len(acc.Choices) == 0 {
		if verbose {
			log.Printf("[verbose] Streaming completed with %d chunks but no choices", chunkCount)
		}
		return openai.ChatCompletionMessage{}, streamed, errors.New("empty streamed completion choices")
	}
	if verbose {
		log.Printf("[verbose] Streaming completed: %d chunks, finish_reason=%s", chunkCount, acc.Choices[0].FinishReason)
	}
	return acc.Choices[0].Message, streamed, nil
}

// runInteractiveChatLoop runs a chat loop with existing message history.
// Returns updated messages, result, and error.
func runInteractiveChatLoop(ctx context.Context, client openai.Client, model openai.ChatModel, messages []openai.ChatCompletionMessageParamUnion, tools *Tools, maxTurns int, stream bool, verbose bool) ([]openai.ChatCompletionMessageParamUnion, ChatLoopResult, error) {
	if maxTurns <= 0 {
		maxTurns = 1
	}

	var lastContent string
	streamedAny := false
	currentMessages := messages

	for turn := 0; turn < maxTurns; turn++ {
		if verbose {
			log.Printf("[verbose] Turn %d/%d: sending request with %d messages", turn+1, maxTurns, len(currentMessages))
		}

		message, streamed, err := runChatOnce(ctx, client, openai.ChatCompletionNewParams{
			Model:    model,
			Messages: currentMessages,
			Tools:    tools.Definitions(),
		}, stream, verbose)
		if err != nil {
			if verbose {
				log.Printf("[verbose] Turn %d: chat request failed: %v", turn+1, err)
			}
			return messages, ChatLoopResult{}, err
		}
		if streamed {
			streamedAny = true
		}
		if strings.TrimSpace(message.Content) != "" {
			lastContent = message.Content
			if verbose {
				log.Printf("[verbose] Turn %d: received assistant content (%d bytes)", turn+1, len(message.Content))
			}
		}

		if len(message.ToolCalls) == 0 {
			if lastContent == "" {
				lastContent = message.Content
			}
			if stream && streamed && !strings.HasSuffix(message.Content, "\n") {
				fmt.Fprintln(os.Stdout)
			}
			if verbose {
				log.Printf("[verbose] Chat loop completed after %d turns (no tool calls)", turn+1)
			}
			// Update messages with assistant response
			updatedMessages := append(currentMessages, message.ToParam())
			return updatedMessages, ChatLoopResult{Content: lastContent, Streamed: streamedAny}, nil
		}

		if verbose {
			log.Printf("[verbose] Turn %d: received %d tool call(s)", turn+1, len(message.ToolCalls))
		}

		currentMessages = append(currentMessages, message.ToParam())
		for i, call := range message.ToolCalls {
			if verbose {
				log.Printf("[verbose] Turn %d: executing tool call %d/%d: %s(id=%s)", turn+1, i+1, len(message.ToolCalls), call.Function.Name, call.ID)
				log.Printf("[verbose] Turn %d: tool call %d arguments: %s", turn+1, i+1, call.Function.Arguments)
			}

			output, err := tools.Execute(call)
			if err != nil {
				output = fmt.Sprintf(`{"ok":false,"error":%q}`, err.Error())
				if verbose {
					log.Printf("[verbose] Turn %d: tool call %d failed: %v", turn+1, i+1, err)
				}
			} else {
				if verbose {
					outputLen := len(output)
					if outputLen > 200 {
						log.Printf("[verbose] Turn %d: tool call %d succeeded, output: %s... (%d bytes total)", turn+1, i+1, output[:200], outputLen)
					} else {
						log.Printf("[verbose] Turn %d: tool call %d succeeded, output: %s", turn+1, i+1, output)
					}
				}
			}
			currentMessages = append(currentMessages, openai.ToolMessage(output, call.ID))
		}
	}

	if lastContent == "" {
		if verbose {
			log.Printf("[verbose] Max turns (%d) reached without assistant content", maxTurns)
		}
		return messages, ChatLoopResult{}, errors.New("max turns reached without assistant content")
	}
	if verbose {
		log.Printf("[verbose] Chat loop completed after %d turns with final content", maxTurns)
	}
	// The messages should already be updated with the assistant response
	// from the last turn, so just return currentMessages
	return currentMessages, ChatLoopResult{Content: lastContent, Streamed: streamedAny}, nil
}

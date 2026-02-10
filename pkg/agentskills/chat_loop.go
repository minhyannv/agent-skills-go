package agentskills

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/openai/openai-go"
)

// Role is the role for a chat message.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message is the public, provider-agnostic chat message DTO.
type Message struct {
	Role    Role
	Content string
}

// ChatOptions controls one chat request.
type ChatOptions struct {
	Stream       bool
	StreamWriter io.Writer
	MaxTurns     int
}

// ChatResult describes the final assistant result for one chat loop.
type ChatResult struct {
	Content  string
	Streamed bool
	Messages []Message
}

func (a *App) runChatOnce(
	params openai.ChatCompletionNewParams,
	stream bool,
	streamWriter io.Writer,
) (openai.ChatCompletionMessage, bool, error) {
	if !stream {
		a.debugf("[verbose] chat: sending non-streaming request")
		completion, err := a.client.Chat.Completions.New(a.ctx, params)
		if err != nil {
			return openai.ChatCompletionMessage{}, false, err
		}
		if len(completion.Choices) == 0 {
			return openai.ChatCompletionMessage{}, false, errors.New("empty completion choices")
		}
		return completion.Choices[0].Message, false, nil
	}

	a.debugf("[verbose] chat: sending streaming request")
	if streamWriter == nil {
		streamWriter = io.Discard
	}

	streamResp := a.client.Chat.Completions.NewStreaming(a.ctx, params)
	defer streamResp.Close()

	acc := openai.ChatCompletionAccumulator{}
	streamed := false
	for streamResp.Next() {
		chunk := streamResp.Current()
		if !acc.AddChunk(chunk) {
			return openai.ChatCompletionMessage{}, streamed, errors.New("failed to accumulate stream")
		}
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			if delta.Content != "" {
				_, _ = io.WriteString(streamWriter, delta.Content)
				streamed = true
			}
		}
	}
	if err := streamResp.Err(); err != nil {
		return openai.ChatCompletionMessage{}, streamed, err
	}
	if len(acc.Choices) == 0 {
		return openai.ChatCompletionMessage{}, streamed, errors.New("empty streamed completion choices")
	}
	return acc.Choices[0].Message, streamed, nil
}

func (a *App) runChatLoop(
	messages []openai.ChatCompletionMessageParamUnion,
	maxTurns int,
	stream bool,
	streamWriter io.Writer,
) ([]openai.ChatCompletionMessageParamUnion, ChatResult, error) {
	if maxTurns <= 0 {
		maxTurns = 1
	}

	var lastContent string
	streamedAny := false
	currentMessages := messages

	for turn := 0; turn < maxTurns; turn++ {
		a.debugf("[verbose] chat: turn=%d/%d", turn+1, maxTurns)
		message, streamed, err := a.runChatOnce(openai.ChatCompletionNewParams{
			Model:    openai.ChatModel(a.config.Model),
			Messages: currentMessages,
			Tools:    a.tools.definitions(),
		}, stream, streamWriter)
		if err != nil {
			return messages, ChatResult{}, err
		}
		if streamed {
			streamedAny = true
		}
		if strings.TrimSpace(message.Content) != "" {
			lastContent = message.Content
		}

		if len(message.ToolCalls) == 0 {
			if lastContent == "" {
				lastContent = message.Content
			}
			if stream && streamed && !strings.HasSuffix(message.Content, "\n") {
				_, _ = fmt.Fprintln(writerOrDiscard(streamWriter))
			}
			updatedMessages := append(currentMessages, message.ToParam())
			return updatedMessages, ChatResult{Content: lastContent, Streamed: streamedAny}, nil
		}

		currentMessages = append(currentMessages, message.ToParam())
		a.debugf("[verbose] chat: assistant requested %d tool call(s)", len(message.ToolCalls))
		for _, call := range message.ToolCalls {
			output, err := a.tools.execute(call)
			if err != nil {
				output = fmt.Sprintf(`{"ok":false,"error":%q}`, err.Error())
			}
			currentMessages = append(currentMessages, openai.ToolMessage(output, call.ID))
		}
	}

	if lastContent == "" {
		return messages, ChatResult{}, errors.New("max turns reached without assistant content")
	}
	return currentMessages, ChatResult{Content: lastContent, Streamed: streamedAny}, nil
}

func writerOrDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

func (a *App) debugf(format string, args ...any) {
	debugf(a.verbose, a.logger, format, args...)
}

// Chat runs one chat loop using provider-agnostic messages.
func (a *App) Chat(messages []Message, opts ChatOptions) (ChatResult, error) {
	internalMessages, err := a.toOpenAIMessages(messages)
	if err != nil {
		return ChatResult{}, err
	}

	maxTurns := opts.MaxTurns
	if maxTurns <= 0 {
		maxTurns = a.config.MaxTurns
	}

	_, result, err := a.runChatLoop(
		internalMessages,
		maxTurns,
		opts.Stream,
		opts.StreamWriter,
	)
	if err != nil {
		return ChatResult{}, err
	}

	updated := append([]Message{}, messages...)
	if strings.TrimSpace(result.Content) != "" {
		updated = append(updated, Message{
			Role:    RoleAssistant,
			Content: result.Content,
		})
	}
	result.Messages = updated
	return result, nil
}

func (a *App) toOpenAIMessages(messages []Message) ([]openai.ChatCompletionMessageParamUnion, error) {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages)+1)
	hasSystem := false
	for _, msg := range messages {
		if msg.Role == RoleSystem {
			hasSystem = true
			break
		}
	}
	if !hasSystem {
		out = append(out, openai.SystemMessage(a.systemPrompt))
	}

	for i, msg := range messages {
		switch msg.Role {
		case RoleSystem:
			out = append(out, openai.SystemMessage(msg.Content))
		case RoleUser:
			out = append(out, openai.UserMessage(msg.Content))
		case RoleAssistant:
			out = append(out, openai.AssistantMessage(msg.Content))
		default:
			return nil, fmt.Errorf("invalid message role at index %d: %q", i, msg.Role)
		}
	}
	return out, nil
}

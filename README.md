# Agent Skills Go

Agent Skills Go is a skill-aware AI agent framework in Go.

It provides:
- A reusable core library: `pkg/agentskills`
- A local CLI adapter: `cmd/agent-skills-go`

The core library focuses on programmatic integration. The CLI is just one way to run it.

## Highlights

- Library-first architecture (`New` + `Chat`)
- Skill discovery from local `SKILL.md` files
- Built-in tools: `read_file`, `write_file`, `run_shell`
- Streaming and non-streaming chat support
- Security controls for filesystem and shell execution
- CLI kept separate from core logic

## Project Layout

```text
cmd/agent-skills-go/          # CLI entry and adapters
cmd/agent-skills-go/repl.go   # REPL adapter

pkg/agentskills/              # Core reusable library
```

## Quick Start (CLI)

### Prerequisites

- Go 1.22+
- OpenAI API key
- OpenAI model name

### Install

```bash
git clone https://github.com/minhyannv/agent-skills-go.git
cd agent-skills-go
go mod download
```

### Configure

Set environment variables (or put them in `.env` for CLI use):

```bash
OPENAI_API_KEY=your_api_key_here
OPENAI_MODEL=gpt-4o-mini
OPENAI_BASE_URL=https://api.openai.com/v1  # optional
```

### Run

```bash
go run ./cmd/agent-skills-go
# Optional: enable skills
# go run ./cmd/agent-skills-go -skills_dirs ./skills -skills_dirs ../shared-skills
```

## Use as a Library

Install:  

```bash
go get github.com/minhyannv/agent-skills-go
```

### End-to-End Example

```go 
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/minhyannv/agent-skills-go/pkg/agentskills"
)

func main() {
	cfg := agentskills.DefaultConfig()
	cfg.SkillsDirs = []string{"./skills"} // optional
	cfg.APIKey = "your_api_key_here"
	cfg.Model = "gpt-4o-mini"
	cfg.BaseURL = "https://api.openai.com/v1"
	cfg.Verbose = true
	cfg.Logger = agentskills.NewWriterLogger(os.Stderr)

	app, err := agentskills.New(context.Background(), cfg)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "init failed: %v\n", err)
		os.Exit(1)
	}

	messages := []agentskills.Message{
		{Role: agentskills.RoleUser, Content: "Summarize this repository."},
	}

	result, err := app.Chat(messages, agentskills.ChatOptions{
		Stream: false,
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "chat failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(result.Content)
	messages = result.Messages // carry forward conversation history
	_ = messages
}
```

## Skills

Skills are discovered from directories in `Config.SkillsDirs`.
Each skill must provide a `SKILL.md` with YAML front matter.
If `SkillsDirs` is empty, the app still works and chats normally without loading any skills.

Example:

```yaml
---
name: pdf
description: PDF processing and manipulation
---
```

## Built-in Tools

### `read_file`

Reads file content with optional byte limit.

Arguments:
- `path` (required)
- `max_bytes` (optional)

### `write_file`

Writes full content to a file.

Arguments:
- `path` (required)
- `content` (required)
- `overwrite` (optional, default false behavior in caller)

### `run_shell`

Runs a command directly (no shell expansion).

Arguments:
- `command` (required)
- `working_dir` (optional)
- `timeout_seconds` (optional)

## Security Model

- Path traversal protection
- Allowed directory restriction (default: current working directory)
- Shell hardening:
  - blocks dangerous executables
  - blocks shell control syntax/operators
  - blocks nested shell interpreters
- Subprocess environment is sanitized

## CLI Configuration

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-skills_dirs` | Skill directory; repeat flag for multiple paths (comma-separated values are not supported) | empty (no skills loaded) |
| `-max_turns` | Max tool-call turns per message | `10` |
| `-stream` | Stream assistant output | `false` |
| `-verbose` | Verbose logging | `false` |
| `-allowed_dir` | Base directory for file operations (`""` disables restriction) | current working directory |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `OPENAI_API_KEY` | Required API key |
| `OPENAI_MODEL` | Required model name |
| `OPENAI_BASE_URL` | Optional base URL |

## Development

Run tests:

```bash
go test ./...
```

Run vet:

```bash
go vet ./...
```

Build CLI:

```bash
go build -o agent-skills-go ./cmd/agent-skills-go
```

## Contributing

See `CONTRIBUTING.md`.

## Security

See `SECURITY.md`.

## License

MIT. See `LICENSE`.

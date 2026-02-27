# Agent Skills Go

Agent Skills Go is a skill-aware AI agent framework in Go.

It provides:
- A reusable core library split by responsibility: `pkg/agent`, `pkg/config`, `pkg/logger`, `pkg/prompt`, `pkg/skills`, `pkg/tools`
- A local CLI adapter: `cmd/agent-skills-go`

The core library focuses on programmatic integration. The CLI is just one way to run it.

## Highlights

- Library-first architecture (`New` + `Run`)
- Skill discovery from local `SKILL.md` files
- Built-in tools: `read_file`, `write_file`, `run_shell`
- Non-streaming agent loop with tool-calling
- Logger dependency injection via `agent.WithLogger(...)`
- Security controls for filesystem and shell execution
- Single-file CLI implementation for easier maintenance

## Project Layout

```text
cmd/agent-skills-go/main.go   # Single-file CLI (flags + REPL + entrypoint)

pkg/agent/                    # AgentLoop orchestration + agent loop
pkg/config/                   # Runtime configuration model
pkg/logger/                   # Logging interface + implementations
pkg/prompt/                   # System prompt composition
pkg/skills/                   # Skill discovery + metadata parsing
pkg/tools/                    # Built-in tools + security execution
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

By default, the CLI auto-loads these built-in skills when present:
- `skills/.system/skill-creator`
- `skills/.system/skill-installer`

When `skill-installer` installs a new skill, it writes to:
- `$CODEX_HOME/skills/<skill-name>`
- defaults to `~/.codex/skills/<skill-name>` when `CODEX_HOME` is unset

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

	"github.com/minhyannv/agent-skills-go/pkg/agent"
	configpkg "github.com/minhyannv/agent-skills-go/pkg/config"
	loggerpkg "github.com/minhyannv/agent-skills-go/pkg/logger"
)

func main() {
	cfg := configpkg.DefaultConfig()
	cfg.SkillsDirs = []string{"./skills"} // optional
	cfg.APIKey = "your_api_key_here"
	cfg.Model = "gpt-4o-mini"
	cfg.BaseURL = "https://api.openai.com/v1"
	cfg.Verbose = true

	app, err := agent.New(
		context.Background(),
		cfg,
		agent.WithLogger(loggerpkg.NewWriterLogger(os.Stderr)),
	)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "init failed: %v\n", err)
		os.Exit(1)
	}

	finalMessage, err := app.Run("Summarize this repository.")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "chat failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(finalMessage.Content)
}
```

Notes:
- `Run` accepts a single user input string and returns one final assistant message.
- Conversation history is stored inside `AgentLoop`; call `app.Reset()` to clear it.

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
| `-max_turns` | Max internal tool-call iterations per user input | `10` |
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

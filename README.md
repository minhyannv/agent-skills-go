# Agent Skills Go

Agent Skills Go is a Go-based interactive agent that discovers local skills and uses OpenAI chat completions to call
tools. It reads skill documentation from `SKILL.md`, runs approved scripts, and enforces path and command safety checks.

## Features

- Interactive terminal chat loop with tool calling
- Skill discovery via `SKILL.md` front matter
- Built-in tools: `read_file`, `write_file`, `run_shell`
- Command execution via `run_shell`
- Security controls: path validation, allowed directories, dangerous command filtering
- Configurable via env vars and CLI flags
- Optional streaming and verbose logging

## Quick Start

### Prerequisites

- Go 1.22+
- OpenAI API key
- A model name in `OPENAI_MODEL`

### Install

```bash
git clone https://github.com/minhyannv/agent-skills-go.git
cd agent-skills-go
go mod download
```

### Configure

Create a `.env` file or set environment variables:

```bash
OPENAI_API_KEY=your_api_key_here
OPENAI_MODEL=gpt-4o-mini
OPENAI_BASE_URL=https://api.openai.com/v1  # optional
```

### Run

```bash
go run . -skills_dirs ./skills,../shared-skills
```

You will enter interactive mode:

```
=== Agent Skills Go - Interactive Mode ===
Type your message and press Enter. Commands:
  /help  - Show this help message
  /clear - Clear conversation history
  /quit  - Exit the program
  /exit  - Exit the program

> 
```

## Configuration

### Command-line flags

| Flag           | Description                                                 | Default    |
|----------------|-------------------------------------------------------------|------------|
| `-skills_dirs` | Comma-separated list of directories containing skills       | `./skills` |
| `-max_turns`   | Max tool-call turns per user message                        | `10`       |
| `-stream`      | Stream assistant output                                     | `false`    |
| `-verbose`     | Verbose logging                                             | `false`    |
| `-allowed_dir` | Base directory for file operations (empty = no restriction) | ``         |

### Environment variables

| Variable          | Description                             |
|-------------------|-----------------------------------------|
| `OPENAI_API_KEY`  | OpenAI API key (required)               |
| `OPENAI_MODEL`    | Model name (required)                   |
| `OPENAI_BASE_URL` | Override OpenAI API base URL (optional) |

## Skills

Skills are discovered by walking the skills directories and parsing `SKILL.md` files. Each file must include YAML front
matter with at least a `name` field. Missing or invalid front matter will fail startup.

Example structure:

```
skills/
  pdf/
    SKILL.md
    scripts/
  docx/
    SKILL.md
```

Example `SKILL.md` header:

```yaml
---
name: pdf
description: PDF processing and manipulation
---
```

At startup, the system prompt includes a list of available skills and the full `SKILL.md` file path for each skill. The
assistant is instructed to open `SKILL.md` with `read_file` before using a skill.

## Built-in Tools

### `read_file`

Read file contents with optional `max_bytes` (default limit is 1MB).

Arguments:

- `path` (string, required)
- `max_bytes` (int, optional)

### `write_file`

Write content to a file on disk.

Arguments:

- `path` (string, required)
- `content` (string, required)
- `overwrite` (bool, optional) — when false, writing to an existing file returns an error

### `run_shell`

Run a shell command using `bash -lc`. Dangerous commands are blocked.

Arguments:

- `command` (string)
- `working_dir` (string, optional)
- `timeout_seconds` (int, optional)

Notes:

- Default timeout is 60 seconds when `timeout_seconds` is not set or <= 0.
- If `-allowed_dir` is set, `working_dir` must be inside an allowed directory.

## Security Model

- **Path validation** blocks traversal attempts (e.g., `../`).
- **Allowed directories**: if `-allowed_dir` is set, all file and working directory operations must stay within that
  directory. The skills directories are also allowed so `SKILL.md` and skill scripts can be read or executed.
- **Dangerous command filtering** blocks destructive commands like `rm`, `dd`, and `mkfs`.

## Architecture

Key packages:

- `main.go`: entrypoint
- `config.go`: config loading and flags
- `app.go`: initialization and tool wiring
- `prompt.go`: system prompt construction
- `skills.go`: skill discovery and parsing
- `tools_*.go`: tool implementations
- `security.go`: safety checks

## Development

Run tests:

```bash
go test ./...
```

Build:

```bash
go build -o agent-skills-go .
```

## Contributing

See `CONTRIBUTING.md`.

## Security

See `SECURITY.md`.

## License

MIT. See `LICENSE`.

## Acknowledgments

- Built with the OpenAI Go SDK

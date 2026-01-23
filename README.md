# Agent Skills Go

A powerful, interactive AI agent framework built with Go that leverages OpenAI's chat completions API to execute skills through tool calling. The agent can read skill documentation, execute scripts, and manipulate files in a secure, controlled environment.

## Features

- 🤖 **Interactive Terminal Mode**: Continuous conversation with the AI agent
- 📚 **Skill-Based Architecture**: Load and execute skills from structured directories
- 🛠️ **Built-in Tools**: File operations, shell commands, Python/Go script execution
- 🔒 **Security First**: Path validation, directory restrictions, and dangerous command filtering
- ⚙️ **Flexible Configuration**: Environment variables and command-line flags
- 📝 **Verbose Logging**: Detailed tool execution logs for debugging

## Quick Start

### Prerequisites

- Go 1.22 or later
- OpenAI API key

### Installation

```bash
git clone https://github.com/minhyannv/agent-skills-go.git
cd agent-skills-go
go mod download
```

### Configuration

Create a `.env` file (optional) or set environment variables:

```bash
OPENAI_API_KEY=your_api_key_here
OPENAI_BASE_URL=https://api.openai.com/v1  # Optional, defaults to OpenAI
OPENAI_MODEL=gpt-4o-mini                     # Optional, defaults to gpt-4o-mini
```

### Run

```bash
go run . -skills_dir examples/skills
```

The application will start in interactive mode:

```
=== Agent Skills Go - Interactive Mode ===
Type your message and press Enter. Commands:
  /help  - Show this help message
  /clear - Clear conversation history
  /quit  - Exit the program
  /exit  - Exit the program

> 
```

## Command-Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-skills_dir` | Directory containing skills | `examples/skills` |
| `-model` | OpenAI model to use | `gpt-4o-mini` (or `OPENAI_MODEL`) |
| `-max_turns` | Maximum tool-call turns | `20` |
| `-stream` | Stream assistant output | `false` |
| `-verbose` | Enable verbose tool-call logging | `false` |
| `-allowed_dir` | Restrict file operations to this directory (empty = no restriction) | `` |

## Interactive Commands

While in interactive mode, you can use these special commands:

- `/help` or `/h` - Show help information
- `/clear` or `/c` - Clear conversation history
- `/quit` or `/exit` or `/q` - Exit the program

## Built-in Tools

The agent has access to the following tools:

### File Operations
- **`read_file`**: Read file contents (with size limits)
- **`write_file`**: Write content to files

### Script Execution
- **`run_shell`**: Execute shell commands (with security filtering)
- **`run_python`**: Execute Python scripts from files
- **`run_go`**: Execute Go scripts from files

All script execution tools support:
- Command-line arguments
- Working directory specification
- Timeout configuration

## Security Features

The framework includes multiple security mechanisms:

1. **Path Validation**: Prevents path traversal attacks (e.g., `../../../etc/passwd`)
2. **Directory Restrictions**: Limit file operations to a specific directory via `-allowed_dir`
3. **Command Filtering**: Blocks dangerous commands (e.g., `rm -rf`, `dd`, etc.)
4. **Working Directory Validation**: Ensures working directories are within allowed scope

### Security Best Practices

- ⚠️ **Always use `-allowed_dir` in production** to restrict file operations
- 🔍 Regularly review and update the dangerous command list
- 🔐 Run the application with minimal required permissions
- 🚫 Never expose API keys in version control

## Skill Directory Structure

Skills are organized in directories, each containing a `SKILL.md` file:

```
skills/
  pdf/
    SKILL.md          # Skill documentation
    scripts/          # Related scripts
  docx/
    SKILL.md
    scripts/
  xlsx/
    SKILL.md
    scripts/
```

### Skill Documentation Format

Each `SKILL.md` file should include YAML front matter:

```yaml
---
name: pdf
description: PDF processing and manipulation
---
```

The agent will automatically discover and load skills from the specified directory, building a system prompt that includes available skills and their locations.

## Architecture

The project follows a modular architecture:

- **`main.go`**: Application entry point
- **`config.go`**: Configuration management (flags and environment variables)
- **`app.go`**: Application initialization and setup
- **`chat.go`**: Core chat completion logic
- **`interactive.go`**: Interactive terminal mode
- **`skills.go`**: Skill discovery and loading
- **`prompt.go`**: System prompt generation
- **`tool.go`**: Tool interface and management
- **`tools_*.go`**: Individual tool implementations
- **`security.go`**: Security validation functions
- **`command.go`**: Command execution utilities

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o agent-skills-go .
```

### Code Structure

The codebase is organized with clear separation of concerns:

- **Configuration**: Centralized in `config.go`
- **Tools**: Each tool is a separate object implementing the `Tool` interface
- **Skills**: Discovered and loaded dynamically from directories
- **Security**: Validation functions in `security.go`

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines and workflow.

## Security

If you discover a security vulnerability, please refer to [SECURITY.md](SECURITY.md) for reporting procedures.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [OpenAI Go SDK](https://github.com/openai/openai-go)
- Inspired by agent frameworks that combine LLMs with tool execution

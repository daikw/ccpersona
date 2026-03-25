# ccpersona - Claude Code Persona System

A system that automatically applies different "personas" to Claude Code sessions based on project configuration. It allows you to maintain consistent interaction styles, expertise levels, and behavioral patterns across different projects.

## Features

- 🎭 **Per-project persona configuration** - Set optimal personas for each project
- 🔄 **Automatic application** - Personas are applied automatically when you start working
- 🎯 **Consistent interactions** - Maintain unified response styles throughout projects
- 🤖 **Multi-platform support** - Works with Claude Code, Cursor, and OpenAI Codex

## Installation

### Quick Install (Recommended)

```bash
curl -sSL https://raw.githubusercontent.com/daikw/ccpersona/main/install.sh | sh
```

This works on Linux, macOS, and Windows (via WSL/Git Bash).

### Homebrew (macOS/Linux)

```bash
brew tap daikw/tap
brew install ccpersona
```

### Build from Source

```bash
git clone https://github.com/daikw/ccpersona.git
cd ccpersona
make build
make install
```

## Quick Start

1. **Initialize in your project**

```bash
cd your-project
ccpersona init
```

2. **Set a persona**

```bash
# List available personas
ccpersona list

# Set active persona
ccpersona set zundamon
```

3. **Configure hooks for your AI coding assistant**

#### For Claude Code

Add the following to your Claude Code settings file (e.g., `~/.claude/settings.json`):

```json
{
  "hooks": {
    "SessionStart": [{"hooks": [{"type": "command", "command": "ccpersona hook"}]}],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ccpersona voice"
          }
        ]
      }
    ],
    "Notification": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ccpersona notify"
          }
        ]
      }
    ]
  }
}
```

#### For OpenAI Codex

Add the following to your Codex config file (`~/.codex/config.toml`):

```toml
# Notification hook that auto-detects platform
notify = ["ccpersona", "notify"]
```

#### For Cursor

Run `ccpersona init` and select "Cursor" when prompted. This will create `.cursor/hooks.json` automatically:

```json
{
  "version": 1,
  "hooks": {
    "sessionStart": [
      { "command": "ccpersona hook" }
    ],
    "afterAgentResponse": [
      { "command": "ccpersona notify --voice" }
    ]
  }
}
```

## Usage

### Basic Commands

```bash
# Initialize persona configuration (interactive - select from available personas)
ccpersona init

# Show current persona (or specify a name to show details)
ccpersona show              # Show current active persona and its content
ccpersona show <name>       # Show specific persona details

# Edit a persona (creates if not exists)
ccpersona edit <persona-name>

# Edit configuration
ccpersona config        # Edit project config
ccpersona config -g     # Edit global config

# Check status (auto-diagnoses on errors)
ccpersona status            # Quick status check
ccpersona status --diagnose # Force detailed diagnostics

# Execute as Claude Code hooks
ccpersona hook                    # session-start hook
ccpersona voice                   # Stop hook (voice synthesis)
ccpersona notify                  # Notification hook (works with both Claude Code and Codex)

# Voice synthesis (expects JSON hook event from stdin by default)
ccpersona voice                   # Read Stop hook JSON event from stdin
echo "こんにちは、世界！" | ccpersona voice --plain  # Read plain text

# Voice synthesis from transcript
ccpersona voice --transcript  # Read latest assistant message from transcript

# Notification handling
ccpersona notify --voice --desktop  # Show desktop notification and speak
```

## Documentation

- [Configuration](docs/configuration.md) - Persona definition, project config, voice settings
- [Advanced Usage](docs/advanced-usage.md) - Multi-device setup, remote voice synthesis
- [Persona Creation Best Practices](docs/persona-best-practices.md) - How to write effective personas
- [Troubleshooting Guide](docs/troubleshooting.md) - Diagnose and fix common issues


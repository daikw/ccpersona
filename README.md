# ccpersona - Claude Code Persona System

A system that automatically applies different "personas" to Claude Code sessions based on project configuration. It allows you to maintain consistent interaction styles, expertise levels, and behavioral patterns across different projects.

## Features

- 🎭 **Per-project persona configuration** - Set optimal personas for each project
- 🔄 **Automatic application** - Personas are applied automatically when you start working
- 📝 **Customizable** - Create and edit custom personas easily
- 🎯 **Consistent interactions** - Maintain unified response styles throughout projects
- 🔊 **Voice synthesis** - Optional text-to-speech for assistant messages
- 🤖 **Multi-platform support** - Works with Claude Code, Cursor, and OpenAI Codex

## Installation

### Quick Install (Recommended)

```bash
curl -sSL https://raw.githubusercontent.com/daikw/ccpersona/main/install.sh | sh
```

This works on Linux (including Jetson/ARM64), macOS, and Windows (via WSL/Git Bash).

### Homebrew (macOS/Linux)

```bash
brew tap daikw/tap
brew install ccpersona
```

### Direct Download

```bash
# Linux ARM64 (Jetson, Raspberry Pi, etc.)
curl -Lo ccpersona.tar.gz https://github.com/daikw/ccpersona/releases/latest/download/ccpersona_Linux_arm64.tar.gz
tar -xzf ccpersona.tar.gz && sudo mv ccpersona /usr/local/bin/

# Linux x86_64
curl -Lo ccpersona.tar.gz https://github.com/daikw/ccpersona/releases/latest/download/ccpersona_Linux_x86_64.tar.gz
tar -xzf ccpersona.tar.gz && sudo mv ccpersona /usr/local/bin/

# macOS ARM64 (Apple Silicon)
curl -Lo ccpersona.tar.gz https://github.com/daikw/ccpersona/releases/latest/download/ccpersona_Darwin_arm64.tar.gz
tar -xzf ccpersona.tar.gz && sudo mv ccpersona /usr/local/bin/
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
ccpersona config init
```

2. **Set a persona**

```bash
# List available personas
ccpersona persona list

# Set active persona
ccpersona config set-persona zundamon
```

3. **Configure hooks for your AI coding assistant**

#### For Claude Code

Add the following to your Claude Code settings file (e.g., `~/.claude/settings.json`):

```json
{
  "hooks": {
    "SessionStart": [{"hooks": [{"type": "command", "command": "ccpersona runtime hook"}]}],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ccpersona runtime voice"
          }
        ]
      }
    ],
    "Notification": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ccpersona runtime notify"
          }
        ]
      }
    ]
  }
}
```

#### For OpenAI Codex

For Codex `hooks.json` lifecycle hooks, pass the platform explicitly because
Claude Code and Codex share the same `SessionStart` payload shape:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ccpersona runtime hook --platform codex"
          }
        ]
      }
    ]
  }
}
```

Add the following to your Codex config file (`~/.codex/config.toml`):

```toml
# Notification hook that auto-detects platform
notify = ["ccpersona", "runtime", "notify"]
```

#### For Cursor

Create `.cursor/hooks.json` with runtime commands:

```json
{
  "version": 1,
  "hooks": {
    "sessionStart": [
      { "command": "ccpersona runtime hook" }
    ],
    "afterAgentResponse": [
      { "command": "ccpersona runtime notify --voice" }
    ]
  }
}
```

- **sessionStart**: Applies persona when conversation starts
- **afterAgentResponse**: Synthesizes voice from AI response (provides `text` field directly)

## Usage

### Basic Commands

```bash
# Initialize ccpersona configuration
ccpersona config init

# Show/edit ccpersona config
ccpersona config show
ccpersona config edit
ccpersona config edit -g
ccpersona config set-persona <name>
ccpersona config migrate

# Manage persona markdown files
ccpersona persona list
ccpersona persona show <name>
ccpersona persona edit <name>

# Check status (auto-diagnoses on errors)
ccpersona config status            # Quick status check
ccpersona config status --diagnose # Force detailed diagnostics

# Execute as AI coding-agent runtime hooks
ccpersona runtime hook                    # session-start hook
ccpersona runtime voice                   # Stop hook (voice synthesis)
ccpersona runtime notify                  # Notification hook (works with Claude Code, Codex, and Cursor)
ccpersona runtime mcp                     # MCP server
ccpersona runtime engine status           # TTS engine status

# Voice synthesis (expects JSON hook event from stdin by default)
ccpersona runtime voice                   # Read Stop hook JSON event from stdin
echo "こんにちは、世界！" | ccpersona runtime voice --plain  # Read plain text

# Voice synthesis from transcript
ccpersona runtime voice --transcript  # Read latest assistant message from transcript

# Notification handling
ccpersona runtime notify --voice --desktop  # Show desktop notification and speak
```

Top-level runtime commands such as `ccpersona hook`, `ccpersona voice`,
`ccpersona notify`, `ccpersona mcp`, and `ccpersona engine` remain executable for
existing hook integrations, but they are hidden from help. Use
`ccpersona runtime ...` for new configurations.

### Creating Personas

To create a new persona:

```bash
ccpersona persona edit my-persona   # Creates if not exists, then opens editor
```

Your editor will open for you to define the persona.

## Persona Definition Structure

Personas are defined as Markdown files with the following sections:

```markdown
# Persona: Name

## Communication Style
Define speaking patterns and tone

## Thinking Approach
Define problem-solving methodology

## Values
Define prioritized values

## Expertise
Define technical specialties and strengths

## Interaction Style
Define how to respond to questions and explain concepts

## Emotional Expression (Optional)
Define patterns for expressing emotions
```

### Sample Personas

- **default** - Standard, polite technical professional
- **zundamon** - Cheerful and energetic character
- **strict_engineer** - Strict, efficiency-focused engineer

## Project Configuration

Use `.agents/ccpersona.json` for persona, voice/provider settings, and user-defined engines:

```json
{
  "name": "fable",
  "voice": {
    "provider": "openai",
    "base_url": "http://127.0.0.1:8088/v1",
    "model": "irodori-tts",
    "voice": "none",
    "format": "wav",
    "timeout_seconds": 300,
    "volume": 1.0,
    "speed": 1.0
  },
  "engines": {
    "irodori": {
      "base_url": "http://127.0.0.1:8088",
      "health": "openai"
    }
  },
  "custom_instructions": "Additional project-specific instructions"
}
```

The lookup order is project first, then global:

1. `<project>/.agents/ccpersona.json`
2. `~/.agents/ccpersona.json`

Legacy files such as `.claude/persona.json`, `.agents/persona.json`, `.codex/persona.json`, `.cursor/persona.json`, and `.claude/config.json` are ignored at runtime. If any are detected and no unified config exists, ccpersona prints a stderr warning and continues with defaults. Run `ccpersona config migrate` to create the unified file.

### Voice Configuration

The voice command expects JSON hook event data from stdin by default (as sent by Claude Code's Stop hook). For plain text input, use the `--plain` flag.

Input modes:
- Default: Stop hook JSON event with `transcript_path` field
- `--plain`: Plain text from stdin
- `--transcript`: Read from latest transcript file

The voice synthesis feature supports:
- **VOICEVOX** - Local voice engine (default port: 50021)
- **AivisSpeech** - Alternative voice engine (default port: 10101)
- **OpenAI / OpenAI-compatible** - Cloud TTS or local servers (Irodori-TTS-Server, kani-tts, etc.)
- **ElevenLabs**, **Amazon Polly**, **GCP** - Cloud TTS providers

Reading modes:
- `short` - Read only the first line (default)
- `full` - Read entire message (use `--chars` to limit characters)

Legacy mode names (`first_line`, `full_text`, etc.) are still supported for backward compatibility.

### Voice Provider Configuration

Voice provider settings live under the `voice` key in `.agents/ccpersona.json`.

Example with OpenAI cloud provider:

```json
{
  "name": "default",
  "voice": {
    "provider": "openai",
    "api_key": "${OPENAI_API_KEY}",
    "model": "tts-1",
    "voice": "nova",
    "speed": 1.0,
    "format": "mp3"
  }
}
```

### OpenAI-Compatible Local TTS Servers

Setting `base_url` in the `voice` block redirects OpenAI-provider requests to a local OpenAI-compatible TTS server (e.g. [Irodori-TTS-Server](https://github.com/daikw/irodori-tts-server) on port 8088, kani-tts on port 8000). When `base_url` is present, `api_key` is optional because local servers typically require no authentication.

Use `timeout_seconds` to extend the HTTP timeout for GPU inference, which can be slow on the first request (default: 30 seconds).

```json
{
  "name": "default",
  "voice": {
    "provider": "openai",
    "base_url": "http://127.0.0.1:8088",
    "model": "irodori-tts",
    "voice": "none",
    "timeout_seconds": 120
  }
}
```

Then synthesize with:

```bash
echo "こんにちは！" | ccpersona runtime voice --plain --provider openai
```

### Engine Registry

The `engines` key in `.agents/ccpersona.json` lets you declare user-defined TTS engines that the `runtime engine` subcommand can manage alongside the built-in VOICEVOX and AivisSpeech engines.

| Field | Type | Description |
|-------|------|-------------|
| `base_url` | string | Base URL for health checks (e.g. `http://127.0.0.1:8088`) |
| `health` | string | Health check type: `"openai"` (GET `/v1/models`) or `"voicevox"` (GET `/version`). Defaults to `"openai"` |
| `command` | string | Executable to launch. Omit to treat the engine as externally managed (status-only) |
| `args` | array | Arguments passed to `command` |
| `dir` | string | Working directory (`~` is expanded) |
| `env` | object | Extra environment variables for the process |

Engine names must not collide with the built-in names (`voicevox`, `aivisspeech`); the registry returns an error if they do.

`runtime engine status` lists all engines (built-in + user-defined) with their health and service state. Engines without `command` are shown as `external (not managed by ccpersona)` and cannot be installed/started/stopped.

Example: declaring an Irodori-TTS engine and using it end-to-end:

```json
{
  "name": "default",
  "voice": {
    "provider": "openai",
    "base_url": "http://127.0.0.1:8088",
    "model": "irodori-tts",
    "voice": "none",
    "timeout_seconds": 120
  },
  "engines": {
    "irodori": {
      "base_url": "http://127.0.0.1:8088",
      "health": "openai",
      "command": "/usr/local/bin/irodori-tts-server",
      "args": ["--port", "8088"],
      "dir": "~/irodori-tts"
    }
  }
}
```

Check the engine is running, then synthesize:

```bash
ccpersona runtime engine status irodori    # shows health + service state
ccpersona runtime voice --plain --provider openai  # routes through base_url
```

## Advanced Usage

### Multi-Device Setup with Remote Voice Synthesis

If you work on multiple devices (e.g., Mac + Jetson terminals), you can run a single voice synthesis engine on your main machine and forward the connection to other devices.

**Architecture:**
```
┌─────────────────┐     ssh -R 10101:localhost:10101
│   Mac (Server)  │◄────────────────────────────────┐
│  AivisSpeech    │                                 │
│  (port 10101)   │                                 │
└─────────────────┘                                 │
                                                    │
┌─────────────────┐  ┌─────────────────┐  ┌────────┴────────┐
│   Jetson #1     │  │   Jetson #2     │  │   Jetson #3     │
│  Speaker: A     │  │  Speaker: B     │  │  Speaker: C     │
│  (project-foo)  │  │  (project-bar)  │  │  (project-baz)  │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

**Setup:**

1. **On the server (Mac):** Start AivisSpeech or VOICEVOX

2. **On each client (Jetson):** Connect with port forwarding
   ```bash
   ssh -R 10101:localhost:10101 user@server
   ```

3. **Configure different speaker IDs per device:**
   ```json
   // Jetson #1: .agents/ccpersona.json
   {
     "name": "default",
     "voice": {
       "provider": "aivisspeech",
       "speaker": 888753760
     }
   }

   // Jetson #2: .agents/ccpersona.json
   {
     "name": "default",
     "voice": {
       "provider": "aivisspeech",
       "speaker": 1234567890
     }
   }
   ```

Now each device produces a distinct voice, making it easy to identify which session is speaking.

## Documentation

- [Persona Creation Best Practices](docs/persona-best-practices.md) - How to write effective personas
- [Troubleshooting Guide](docs/troubleshooting.md) - Diagnose and fix common issues

## File Locations

- Global personas: `~/.agents/personas/`
- Unified project configuration: `<project>/.agents/ccpersona.json`
- Unified global configuration: `~/.agents/ccpersona.json`
- Session tracking: `/tmp/ccpersona-sessions/`

For migration compatibility, existing persona files under `~/.claude/personas/`
are still read as a fallback. New persona files are created under
`~/.agents/personas/`.

## Development

### Requirements

- Go 1.25 or later
- Make

### Build

```bash
# Build for current platform
make build

# Run tests
make test

# Build for all platforms
make build-all
```

### Release

This project uses [GoReleaser](https://goreleaser.com/) for automated releases.

```bash
# Create a snapshot build locally
make snapshot

# Test the release process without publishing
make release-test

# Create a new version tag
make tag
# Then push the tag to trigger the release
git push origin --tags
```

## Technical Details

### How Hooks Work

#### Claude Code Integration

ccpersona integrates with Claude Code through the SessionStart hook (recommended):

1. Configure Claude Code to run `ccpersona runtime hook` on SessionStart (see Quick Start above)
2. At the start of each session, ccpersona checks `.agents/ccpersona.json` in the current project, then `~/.agents/ccpersona.json`
3. If found, the persona instructions are output to stdout and Claude Code applies them
4. Legacy config files are ignored; run `ccpersona config migrate` before relying on old settings

#### OpenAI Codex Integration

ccpersona integrates with OpenAI Codex through the notify hook:

1. Configure Codex to run `ccpersona runtime notify` on agent-turn-complete events
2. When an agent turn completes, ccpersona receives the event in JSON format
3. The unified hook interface automatically detects the platform
4. Appropriate actions (voice synthesis, notifications) are performed based on the event type

#### Unified Hook Interface

The `runtime notify` command provides a unified interface that automatically detects and handles events from Claude Code, OpenAI Codex, and Cursor:

- **Auto-detection**: Analyzes the JSON structure to determine the source platform
  - Codex: `"type": "agent-turn-complete"` field
  - Cursor: `"conversation_id"` field
  - Claude Code: `"session_id"` + `"hook_event_name"` fields
- **Platform hints**: Use `ccpersona runtime hook --platform codex` or `CCPERSONA_PLATFORM=codex`
  for Codex lifecycle hooks whose payload shape overlaps Claude Code
- **Codex events**: Handles `agent-turn-complete` events with turn completion notifications
- **Cursor events**: Handles `afterAgentResponse` for voice synthesis (provides AI response directly)
- **Claude Code events**: Routes `UserPromptSubmit`, `Stop`, and `Notification` events to appropriate handlers
- **Persona support**: Applies personas for all platforms using platform-specific configuration
- **Voice synthesis**: Works with all platforms using the configured voice engine

This design provides:
- Simple setup (works immediately after brew install)
- Cross-platform compatibility (Windows/Mac/Linux)
- Multi-platform AI assistant support (Claude Code, OpenAI Codex, and Cursor)
- Robust error handling (silent failures to avoid disrupting the AI assistant)
- Idempotent persona application (SessionStart fires once per session; re-running is always safe)
- Advanced customization options

### Security Notes

- Hooks execute with your user permissions
- Only install personas from trusted sources
- Review persona content before applying

## License

MIT License

## Contributing

Issues and Pull Requests are welcome!

## Acknowledgments

- [urfave/cli](https://github.com/urfave/cli) - CLI framework
- [zerolog](https://github.com/rs/zerolog) - Structured logging
- [VOICEVOX](https://voicevox.hiroshiba.jp/) - Voice synthesis engine
- [AivisSpeech](https://aivis-project.com/) - Alternative voice engine

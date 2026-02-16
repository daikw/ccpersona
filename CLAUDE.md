# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Essential Commands

```bash
# Build and development
export GOPATH=$HOME/go && make build    # Build ccpersona binary (GOPATH export required to avoid tilde expansion error)
export GOPATH=$HOME/go && make test     # Run all tests with coverage
make fmt                                 # Format code
make vet                                 # Run go vet
make check                              # Run fmt, vet, and tests
make build-all                          # Build for all platforms (darwin/linux/windows)

# Test specific packages
go test -v ./internal/persona/...       # Test persona package
go test -v -run TestHandleSessionStart  # Run specific test

# Test voice synthesis
echo "テストなのだ！" | ./ccpersona voice --plain
```

## Development

- `MUST` use `mise` as a tool for tool's version management
- `MUST` check lint (golangci-lint) before commit
- `MUST` check test coverage before commit

## Architecture Overview

ccpersona is a persona management system that automatically applies different "personalities" to AI coding assistant sessions (Claude Code and OpenAI Codex) based on project configuration. The system is designed as a single Go binary that replaces shell script dependencies and provides unified hook handling for multiple platforms.

### Core Components

1. **Persona System** (`internal/persona/`)
   - Personas are markdown files stored in `~/.claude/personas/`
   - Project configuration in `.claude/persona.json`
   - Session tracking prevents duplicate persona applications
   - Manager handles persona CRUD operations and AI assistant integration

2. **Hook System** (`internal/hook/`)
   - **types.go**: Defines event types for Claude Code, Codex, and Cursor
     - Claude Code events: UserPromptSubmit, Stop, Notification, PreToolUse, PostToolUse, PreCompact
     - Codex events: CodexNotifyEvent (agent-turn-complete)
     - Cursor events: sessionStart, afterAgentResponse, stop (camelCase naming)
   - **unified.go**: Unified hook interface with auto-detection
     - DetectAndParse(): Automatically detects Claude Code, Codex, or Cursor events from JSON
     - UnifiedHookEvent: Normalized event structure for all platforms
     - Platform-specific handlers route events to appropriate logic
   - **Platform Detection**:
     - Codex: `"type": "agent-turn-complete"` field
     - Cursor: `"conversation_id"` field (uses camelCase event names)
     - Claude Code: `"session_id"` + `"hook_event_name"` fields (uses PascalCase event names)

3. **Voice Synthesis** (`internal/voice/`)
   - Default: reads Stop hook JSON event from stdin (expects JSON with transcript_path)
   - With --plain flag: reads plain text from stdin for voice synthesis
   - With --transcript flag: reads from `~/.claude/projects/*.jsonl`
   - As Stop hook: automatically uses transcript path from hook event
   - Supports VOICEVOX (port 50021) and AivisSpeech (port 10101) engines
   - Reading modes: short (first line) or full (entire text with optional --chars limit)
   - Cross-platform audio playback (afplay/aplay/paplay/ffplay)

4. **CLI Framework**
   - Uses urfave/cli v3 (note: v3 has different API from v2)
   - Single entry point in `cmd/main.go`
   - All commands return nil on success to avoid disrupting hooks
   - **notify**: Unified command that auto-detects and handles events from Claude Code, Codex, and Cursor

### Hook Integration

#### Claude Code Integration

The system integrates with Claude Code via hooks. **SessionStart is the recommended hook** as it's triggered once per session:

**Recommended configuration (SessionStart):**
```json
{
  "hooks": {
    "session-start": ["ccpersona hook"]
  }
}
```

**Legacy configuration (UserPromptSubmit) - still supported:**
```json
{
  "hooks": {
    "user-prompt-submit": ["ccpersona hook"]
  }
}
```

The hook process:
1. User configures Claude Code with the hook command
2. On session start (or prompt submit for legacy), ccpersona checks for `.claude/persona.json`
3. If found, applies the specified persona by outputting formatted instructions
4. Session tracking prevents re-application during the same session

#### OpenAI Codex Integration

The system integrates with OpenAI Codex via the notify hook:
1. User configures Codex with `notify = ["ccpersona", "notify"]` in `~/.codex/config.toml`
2. On agent-turn-complete events, ccpersona receives JSON with turn details
3. The unified hook interface (DetectAndParse) automatically detects Codex events
4. Appropriate actions are performed (notifications, voice synthesis)

#### Cursor Integration

The system integrates with Cursor via hooks configured in `~/.cursor/hooks.json`:

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

- **sessionStart**: Applies persona at conversation start
- **afterAgentResponse**: Synthesizes voice from the AI response (recommended for voice)
  - This hook provides the `text` field with the AI's response directly
  - Unlike Claude Code's `stop` event, Cursor's `stop` doesn't include the response text

#### Unified Hook Interface

The `notify` command provides a single interface for all platforms:
- **Auto-detection**: Parses stdin JSON and identifies the platform by structure
  - Codex events have `"type": "agent-turn-complete"` field
  - Cursor events have `"conversation_id"` field
  - Claude Code events have `"session_id"` + `"hook_event_name"` fields
- **Routing**: Routes events to platform-specific handlers
- **Shared functionality**: All platforms use the same persona and voice configuration

### Platform-Specific Configuration

ccpersona supports platform-specific configuration files, allowing different personas for different AI assistants.
Each platform uses its own standard configuration directory.

#### Global Configuration Directories

| Platform    | Global Config Path         |
|-------------|---------------------------|
| Claude Code | `~/.claude/persona.json`  |
| Codex       | `~/.codex/persona.json`   |
| Cursor      | `~/.cursor/persona.json`  |

#### Configuration Fallback Hierarchy

**Claude Code:**
1. `.claude/persona.json` (project)
2. `~/.claude/persona.json` (global)

**Codex:**
1. `.claude/codex/persona.json` (project, platform-specific)
2. `.claude/persona.json` (project, common)
3. `~/.codex/persona.json` (global)

**Cursor:**
1. `.claude/cursor/persona.json` (project, platform-specific)
2. `.claude/persona.json` (project, common)
3. `~/.cursor/persona.json` (global)

#### Example Directory Structure

```
# Global configs (each platform uses its own directory)
~/.claude/persona.json        # Claude Code
~/.codex/persona.json         # Codex
~/.cursor/persona.json        # Cursor

# Project configs (all in .claude/)
./your-project/.claude/
├── persona.json              # Common (all platforms)
├── codex/
│   └── persona.json          # Codex specific
└── cursor/
    └── persona.json          # Cursor specific
```

### Key Design Decisions

- **No shell scripts**: All functionality implemented in Go for cross-platform compatibility
- **Multi-platform support**: Single codebase works with both Claude Code and OpenAI Codex
- **Silent failures in hooks**: Errors are logged but don't fail to avoid disrupting AI assistants
- **Session persistence**: Session markers stored in `/tmp/ccpersona-sessions/` with 24-hour cleanup
- **Persona format**: Markdown with specific sections (口調, 考え方, 価値観, etc.)
- **Unified hook interface**: Auto-detection of platform from JSON structure eliminates need for separate configurations

## Important Implementation Details

- When modifying hook behavior, test with actual Claude Code to ensure compatibility
- Voice synthesis requires external engines running locally
- The `ccpersona` binary must be in PATH for Claude Code hooks to work
- Personas can include voice configuration for automatic synthesis
- GOPATH tilde expansion issue requires explicit export in make commands

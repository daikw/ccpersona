# ccpersona AI and Maintainer Reference

This file contains the detailed operational notes that are useful for AI agents,
maintainers, and advanced users. Keep the public [README.md](README.md) short and
human-oriented.

## Repository Commands

```bash
make build
make test
make fmt
make vet
make check
make build-all
golangci-lint run
```

Useful targeted checks:

```bash
go test ./internal/persona ./internal/voice ./cmd
HOME=$(mktemp -d) go test $(go list ./... | grep -v '/internal/voice/provider$')
```

## Command Shape

Visible root commands are intentionally small:

```text
ccpersona config
ccpersona persona
ccpersona runtime
```

Runtime commands are grouped under `ccpersona runtime`:

```text
ccpersona runtime hook
ccpersona runtime voice
ccpersona runtime notify
ccpersona runtime mcp
ccpersona runtime engine
```

Legacy top-level runtime commands remain executable but hidden from help:

```text
ccpersona hook
ccpersona voice
ccpersona notify
ccpersona mcp
ccpersona engine
```

Do not add new visible root commands unless the command is a primary user
workflow. Prefer `config`, `persona`, or `runtime` subcommands.

## Configuration Model

ccpersona uses one unified config file for all supported coding agents:

1. `<project>/.agents/ccpersona.json`
2. `~/.agents/ccpersona.json`

Example:

```json
{
  "name": "fable",
  "voice": {
    "provider": "openai",
    "base_url": "http://127.0.0.1:8088/v1",
    "model": "irodori-tts",
    "voice": "none",
    "format": "wav",
    "timeout_seconds": 300
  },
  "engines": {
    "irodori": {
      "base_url": "http://127.0.0.1:8088",
      "health": "openai"
    }
  }
}
```

Legacy files such as `.claude/persona.json`, `.agents/persona.json`,
`.codex/persona.json`, `.cursor/persona.json`, and `.claude/config.json` are
ignored at runtime. If any are detected and no unified config exists, ccpersona
prints a stderr warning and continues with defaults. Use
`ccpersona config migrate` to create the unified file.

## File Locations

```text
~/.agents/ccpersona/personas/   global persona markdown files
~/.agents/ccpersona/mute        global voice mute marker
~/.agents/ccpersona.json        global config
<project>/.agents/ccpersona.json project config
/tmp/ccpersona-sessions/        session tracking
```

Migration fallbacks:

```text
~/.agents/personas/
~/.claude/personas/
~/.claude/ccpersona/mute
```

New persona files and new mute markers should be written only to the canonical
`~/.agents/ccpersona/` paths.

## Hook Integration

### Claude Code

Claude Code integration is primarily through `SessionStart`:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ccpersona runtime hook"
          }
        ]
      }
    ]
  }
}
```

`Stop` can run voice synthesis:

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ccpersona runtime voice"
          }
        ]
      }
    ]
  }
}
```

Legacy `UserPromptSubmit` integration is still supported but SessionStart is
preferred because persona application is idempotent and session-scoped.

### OpenAI Codex

Codex lifecycle hooks can share a payload shape with Claude Code. Use an explicit
platform hint:

```bash
ccpersona runtime hook --platform codex
```

Codex notifications use:

```toml
notify = ["ccpersona", "runtime", "notify"]
```

### Cursor

Cursor hooks use camelCase event names:

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

`afterAgentResponse` is the recommended Cursor voice hook because it provides the
assistant response text directly.

## Unified Hook Detection

`ccpersona runtime notify` reads JSON from stdin and normalizes events across
Claude Code, Codex, and Cursor.

Detection hints:

- Codex notify: `type == "agent-turn-complete"`
- Codex lifecycle with explicit hint: `--platform codex` or `CCPERSONA_PLATFORM=codex`
- Cursor: `conversation_id`
- Claude Code: `session_id` plus `hook_event_name`

The normalized event carries the platform, event type, session ID, transcript
path, and optional assistant response text.

Hook commands should fail soft. Runtime paths print useful diagnostics to stderr
but avoid disrupting the caller when possible.

## Voice Configuration

The voice command expects Claude Code Stop hook JSON on stdin by default. Other
input modes:

```bash
echo "こんにちは" | ccpersona runtime voice --plain
ccpersona runtime voice --transcript
```

Supported providers:

- `voicevox`
- `aivisspeech`
- `openai`
- `elevenlabs`
- `polly`
- `gcp`

Reading modes:

- `short`: read only the first line
- `full`: read the whole message, optionally limited with `--chars`

Legacy mode names such as `first_line` and `full_text` are still accepted.

### OpenAI-Compatible Local TTS

The OpenAI provider can target a local OpenAI-compatible TTS server by setting
`base_url`. When `base_url` is present and does not point at the official OpenAI
host, `api_key` is optional.

```json
{
  "name": "default",
  "voice": {
    "provider": "openai",
    "base_url": "http://127.0.0.1:8088/v1",
    "model": "irodori-tts",
    "voice": "none",
    "timeout_seconds": 120
  }
}
```

`timeout_seconds` is important for local GPU inference where the first request
can be slow.

## Engine Registry

The `engines` key declares user-defined TTS engines that
`ccpersona runtime engine` can manage alongside built-in VOICEVOX and
AivisSpeech engines.

| Field | Type | Description |
| --- | --- | --- |
| `base_url` | string | Base URL for health checks |
| `health` | string | `openai` for GET `/v1/models`, `voicevox` for GET `/version` |
| `command` | string | Executable to launch |
| `args` | array | Arguments passed to `command` |
| `dir` | string | Working directory, with `~` expansion |
| `env` | object | Extra environment variables |

Engine names must not collide with built-ins (`voicevox`, `aivisspeech`).

Example:

```json
{
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

Engines without `command` are treated as externally managed: status and health
checks work, but install/start/stop do not.

## Architecture Notes

Core packages:

- `internal/persona`: config loading, persona file management, session context
- `internal/hook`: Claude Code, Codex, and Cursor hook parsing and normalization
- `internal/voice`: voice config resolution, transcript reading, provider layer
- `internal/voice/provider`: cloud and OpenAI-compatible provider implementations
- `internal/engine`: built-in and user-defined TTS engine registry
- `internal/mcp`: stdio MCP server
- `cmd`: CLI wiring with urfave/cli v3

Design constraints:

- Keep hook execution robust and quiet.
- Keep visible CLI surface small.
- Prefer `.agents` paths for cross-agent state.
- Preserve migration fallbacks for older user installations.
- Avoid shell-script runtime dependencies in hook paths.

## Multi-Device Voice Setup

For Mac plus remote Linux/Jetson workflows, run a TTS engine on one machine and
forward the port:

```bash
ssh -R 10101:localhost:10101 user@server
```

Each client can set a different speaker ID in its project
`.agents/ccpersona.json`:

```json
{
  "name": "default",
  "voice": {
    "provider": "aivisspeech",
    "speaker": 888753760
  }
}
```

## Release

This project uses GoReleaser.

```bash
make snapshot
make release-test
make tag
git push origin --tags
```

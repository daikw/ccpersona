# ccpersona

ccpersona applies project-specific personas to AI coding assistants and can
optionally speak assistant messages through local or cloud TTS providers.

It supports Claude Code, OpenAI Codex, and Cursor from one configuration file:

```text
<project>/.agents/ccpersona.json
~/.agents/ccpersona.json
```

For implementation details, hook payload behavior, and maintainer notes, see
[README.ai.md](README.ai.md).

## Features

- Per-project persona selection
- Shared config across Claude Code, Codex, Cursor, and MCP
- Optional voice synthesis for assistant messages
- Local TTS engines such as VOICEVOX, AivisSpeech, and OpenAI-compatible servers
- Cloud TTS providers such as OpenAI, ElevenLabs, Amazon Polly, and GCP
- Hidden compatibility commands for older hook configurations

## Install

### Quick Install

```bash
curl -sSL https://raw.githubusercontent.com/daikw/ccpersona/main/install.sh | sh
```

This works on Linux, macOS, and Windows through WSL or Git Bash.

### Homebrew

```bash
brew tap daikw/tap
brew install ccpersona
```

### From Source

```bash
git clone https://github.com/daikw/ccpersona.git
cd ccpersona
make build
make install
```

## Quick Start

Create project config:

```bash
cd your-project
ccpersona config init
```

Choose a persona:

```bash
ccpersona persona list
ccpersona config set-persona fable
```

Configure your assistant to run ccpersona hooks:

| Assistant | Primary command |
| --- | --- |
| Claude Code | `ccpersona runtime hook` on `SessionStart` |
| OpenAI Codex | `ccpersona runtime hook --platform codex` for lifecycle hooks |
| Cursor | `ccpersona runtime hook` on `sessionStart` |

For copy-paste hook snippets, see [README.ai.md](README.ai.md#hook-integration).

## Configuration

Minimal project config:

```json
{
  "name": "fable"
}
```

Config with OpenAI-compatible local TTS:

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
  }
}
```

Lookup order:

1. `<project>/.agents/ccpersona.json`
2. `~/.agents/ccpersona.json`

Global persona files live under:

```text
~/.agents/ccpersona/personas/
```

Existing persona files under `~/.agents/personas/` or `~/.claude/personas/`
are still read as migration fallbacks. New persona files are created under
`~/.agents/ccpersona/personas/`.

## Commands

```bash
ccpersona config init
ccpersona config show
ccpersona config edit
ccpersona config set-persona <name>
ccpersona config status
ccpersona config migrate

ccpersona persona list
ccpersona persona show <name>
ccpersona persona edit <name>

ccpersona runtime hook
ccpersona runtime voice
ccpersona runtime notify
ccpersona runtime mcp
ccpersona runtime engine status
```

Top-level runtime commands such as `ccpersona hook`, `ccpersona voice`,
`ccpersona notify`, `ccpersona mcp`, and `ccpersona engine` remain executable for
existing hook integrations, but they are hidden from help. Use
`ccpersona runtime ...` for new configurations.

## Documentation

- [README.ai.md](README.ai.md) - AI agent and maintainer reference
- [Persona Creation Best Practices](docs/persona-best-practices.md)
- [Troubleshooting Guide](docs/troubleshooting.md)
- [Hook Migration Notes](docs/hook_migration.md)

## Development

```bash
make build
make test
make check
make build-all
```

This project uses GoReleaser for releases.

```bash
make snapshot
make release-test
make tag
git push origin --tags
```

## Security Notes

- Hooks execute with your user permissions.
- Only install personas from trusted sources.
- Review persona content before applying it.

## License

MIT License

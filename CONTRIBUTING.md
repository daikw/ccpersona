# Contributing

Issues and Pull Requests are welcome!

## Development

### Requirements

- Go 1.21 or later
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

ccpersona integrates with Claude Code through the SessionStart hook:

1. Configure Claude Code to run `ccpersona hook` on session start
2. When a session starts, ccpersona checks for `.claude/persona.json` in the current directory
3. If found, the persona instructions are output to stdout
4. Claude Code receives these instructions and adjusts its behavior accordingly

#### OpenAI Codex Integration

ccpersona integrates with OpenAI Codex through the notify hook:

1. Configure Codex to run `ccpersona notify` on agent-turn-complete events
2. When an agent turn completes, ccpersona receives the event in JSON format
3. The unified hook interface automatically detects the platform
4. Appropriate actions (voice synthesis, notifications) are performed based on the event type

#### Unified Hook Interface

The `notify` command provides a unified interface that automatically detects and handles events from Claude Code, OpenAI Codex, and Cursor:

- **Auto-detection**: Analyzes the JSON structure to determine the source platform
  - Codex: `"type": "agent-turn-complete"` field
  - Cursor: `"conversation_id"` field
  - Claude Code: `"session_id"` + `"hook_event_name"` fields
- **Codex events**: Handles `agent-turn-complete` events with turn completion notifications
- **Cursor events**: Handles `afterAgentResponse` for voice synthesis (provides AI response directly)
- **Claude Code events**: Routes `SessionStart`, `Stop`, and `Notification` events to appropriate handlers
- **Persona support**: Applies personas for all platforms using platform-specific configuration
- **Voice synthesis**: Works with all platforms using the configured voice engine

### Security Notes

- Hooks execute with your user permissions
- Only install personas from trusted sources
- Review persona content before applying

## Acknowledgments

- [urfave/cli](https://github.com/urfave/cli) - CLI framework
- [zerolog](https://github.com/rs/zerolog) - Structured logging
- [VOICEVOX](https://voicevox.hiroshiba.jp/) - Voice synthesis engine
- [AivisSpeech](https://aivis-project.com/) - Alternative voice engine

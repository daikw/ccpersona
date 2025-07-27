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
```

## Architecture Overview

ccpersona is a Claude Code persona management system that automatically applies different "personalities" to Claude Code sessions based on project configuration. The system is designed as a single Go binary that replaces shell script dependencies.

### Core Components

1. **Persona System** (`internal/persona/`)
   - Personas are markdown files stored in `~/.claude/personas/`
   - Project configuration in `.claude/persona.json`
   - Session tracking prevents duplicate persona applications
   - Manager handles persona CRUD operations and Claude Code integration

2. **Voice Synthesis** (`internal/voice/`)
   - Reads Claude Code transcripts from `~/.claude/projects/*.jsonl`
   - Supports VOICEVOX (port 50021) and AivisSpeech (port 10101) engines
   - Multiple reading modes: first_line, full_text, char_limit, etc.
   - Cross-platform audio playback (afplay/aplay/paplay/ffplay)

3. **CLI Framework**
   - Uses urfave/cli v3 (note: v3 has different API from v2)
   - Single entry point in `cmd/main.go`
   - All commands return nil on success to avoid disrupting Claude Code hooks

### Hook Integration

The system integrates with Claude Code via the UserPromptSubmit hook:
1. User configures Claude Code with `"user-prompt-submit": "ccpersona hook"`
2. On session start, ccpersona checks for `.claude/persona.json` in the current directory
3. If found, applies the specified persona by outputting formatted instructions
4. Session tracking prevents re-application during the same session

### Key Design Decisions

- **No shell scripts**: All functionality implemented in Go for cross-platform compatibility
- **Silent failures in hooks**: Errors are logged but don't fail to avoid disrupting Claude Code
- **Session persistence**: Session markers stored in `/tmp/ccpersona-sessions/` with 24-hour cleanup
- **Persona format**: Markdown with specific sections (口調, 考え方, 価値観, etc.)

## Important Implementation Details

- When modifying hook behavior, test with actual Claude Code to ensure compatibility
- Voice synthesis requires external engines running locally
- The `ccpersona` binary must be in PATH for Claude Code hooks to work
- Personas can include voice configuration for automatic synthesis
- GOPATH tilde expansion issue requires explicit export in make commands

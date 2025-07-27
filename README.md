# ccpersona - Claude Code Persona System

A system that automatically applies different "personas" to Claude Code sessions based on project configuration. It allows you to maintain consistent interaction styles, expertise levels, and behavioral patterns across different projects.

## Features

- 🎭 **Per-project persona configuration** - Set optimal personas for each project
- 🔄 **Automatic application** - Personas are applied automatically when you start working
- 📝 **Customizable** - Create and edit custom personas easily
- 🎯 **Consistent interactions** - Maintain unified response styles throughout projects
- 🔊 **Voice synthesis** - Optional text-to-speech for assistant messages

## Installation

### Homebrew (Recommended)

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

### Download Binary

Download the latest binary from the [Releases](https://github.com/daikw/ccpersona/releases) page.

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

3. **Configure Claude Code hook**

Add the following to your Claude Code settings file (e.g., `~/.claude/settings.json`):

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "ccpersona hook"
          }
        ]
      }
    ]
  }
}
```

Now the persona will be applied automatically when you submit prompts in Claude Code.

## Usage

### Basic Commands

```bash
# Initialize persona configuration in current project
ccpersona init

# List available personas
ccpersona list

# Show current active persona
ccpersona current

# Set active persona
ccpersona set <persona-name>

# Show persona details
ccpersona show <persona-name>

# Create a new persona
ccpersona create <persona-name>

# Edit an existing persona
ccpersona edit <persona-name>

# Edit configuration
ccpersona config        # Edit project config
ccpersona config -g     # Edit global config

# Execute as Claude Code hook
ccpersona hook

# Voice synthesis (read latest assistant message)
ccpersona voice

# Voice synthesis with options
ccpersona voice --mode full_text --engine voicevox
```

### Creating Personas

To create a new persona:

```bash
ccpersona create my-persona
ccpersona edit my-persona
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

Manage settings in `.claude/persona.json` for each project:

```json
{
  "name": "zundamon",
  "voice": {
    "engine": "voicevox",
    "speaker_id": 3
  },
  "override_global": true,
  "custom_instructions": "Additional project-specific instructions"
}
```

### Voice Configuration

The voice synthesis feature supports:
- **VOICEVOX** - Local voice engine (default port: 50021)
- **AivisSpeech** - Alternative voice engine (default port: 10101)

Reading modes:
- `first_line` - Read only the first line
- `line_limit` - Read up to N lines
- `after_first` - Skip first line, read the rest
- `full_text` - Read entire message
- `char_limit` - Read up to N characters

## File Locations

- Global personas: `~/.claude/personas/`
- Project configuration: `<project>/.claude/persona.json`
- Session tracking: `/tmp/ccpersona-sessions/`

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

## Technical Details

### How Hooks Work

ccpersona integrates with Claude Code through the UserPromptSubmit hook:

1. Configure Claude Code to run `ccpersona hook` on each prompt submission
2. When you submit a prompt, ccpersona checks for `.claude/persona.json` in the current directory
3. If found and it's a new session, the persona instructions are output
4. Claude Code receives these instructions and adjusts its behavior accordingly

This design provides:
- Simple setup (works immediately after brew install)
- Cross-platform compatibility (Windows/Mac/Linux)
- Robust error handling (silent failures to avoid disrupting Claude Code)
- Session tracking (prevents duplicate persona applications)
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
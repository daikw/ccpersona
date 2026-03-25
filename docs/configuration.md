# Configuration

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

### Creating Personas

```bash
ccpersona edit my-persona   # Creates if not exists, then opens editor
```

Your editor will open for you to define the persona.

## Project Configuration

Manage settings in `.claude/persona.json` for each project:

```json
{
  "name": "zundamon",
  "voice": {
    "provider": "voicevox",
    "speaker": 3,
    "volume": 1.0,
    "speed": 1.0
  },
  "override_global": true,
  "custom_instructions": "Additional project-specific instructions"
}
```

## Voice Configuration

The voice command expects JSON hook event data from stdin by default (as sent by Claude Code's Stop hook). For plain text input, use the `--plain` flag.

Input modes:
- Default: Stop hook JSON event with `transcript_path` field
- `--plain`: Plain text from stdin
- `--transcript`: Read from latest transcript file

The voice synthesis feature supports:
- **VOICEVOX** - Local voice engine (default port: 50021)
- **AivisSpeech** - Alternative voice engine (default port: 10101)

Reading modes:
- `short` - Read only the first line (default)
- `full` - Read entire message (use `--chars` to limit characters)

Legacy mode names (`first_line`, `full_text`, etc.) are still supported for backward compatibility.

## File Locations

- Global personas: `~/.claude/personas/`
- Project configuration: `<project>/.claude/persona.json`
- Session tracking: `/tmp/ccpersona-sessions/`

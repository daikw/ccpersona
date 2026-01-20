package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

var (
	version  = "dev"
	revision = "none"
)

func main() {
	// Setup logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Simple CLI setup - no special hook detection needed

	app := &cli.Command{
		Name:  "ccpersona",
		Usage: "Claude Code Persona System - manage personas for Claude Code sessions",
		Description: `ccpersona helps you manage different personas for Claude Code.
It allows you to switch between different personality settings, voice configurations,
and behavioral patterns for your AI assistant.`,
		Version: fmt.Sprintf("%s (rev: %s)", version, revision),
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"V"},
				Usage:   "Enable verbose logging",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "init",
				Usage:   "Initialize persona configuration in current project",
				Action:  handleInit,
				Aliases: []string{"i"},
			},
			{
				Name:    "list",
				Usage:   "List available personas",
				Action:  handleList,
				Aliases: []string{"ls", "l"},
			},
			{
				Name:    "current",
				Usage:   "Show current active persona",
				Action:  handleCurrent,
				Aliases: []string{"c"},
			},
			{
				Name:      "set",
				Usage:     "Set the active persona for this project",
				Action:    handleSet,
				Aliases:   []string{"s"},
				ArgsUsage: "<persona>",
			},
			{
				Name:      "show",
				Usage:     "Show details of a specific persona",
				Action:    handleShow,
				ArgsUsage: "<persona>",
			},
			{
				Name:      "create",
				Usage:     "Create a new persona",
				Action:    handleCreate,
				ArgsUsage: "<name>",
			},
			{
				Name:      "edit",
				Usage:     "Edit an existing persona",
				Action:    handleEdit,
				ArgsUsage: "<persona>",
			},
			{
				Name:   "config",
				Usage:  "Manage global configuration",
				Action: handleConfig,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "global",
						Aliases: []string{"g"},
						Usage:   "Edit global settings",
					},
				},
			},
			{
				Name:   "setup",
				Usage:  "Interactive setup wizard for ccpersona",
				Action: handleSetup,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "skip-hooks",
						Usage: "Skip Claude Code hooks configuration",
						Value: false,
					},
				},
			},
			{
				Name:   "status",
				Usage:  "Show current ccpersona status",
				Action: handleStatus,
			},
			{
				Name:   "doctor",
				Usage:  "Diagnose ccpersona configuration and connectivity",
				Action: handleDoctor,
			},
			{
				Name:    "hook",
				Aliases: []string{"user_prompt_submit_hook"},
				Usage:   "Execute as Claude Code UserPromptSubmit hook",
				Action:  handleHook,
			},
			{
				Name:    "voice",
				Aliases: []string{"stop_hook"},
				Usage:   "Synthesize voice from text (stdin by default, or from transcript)",
				Action:  handleVoice,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "transcript",
						Usage: "Read from Claude Code transcript instead of stdin",
						Value: false,
					},
					&cli.BoolFlag{
						Name:  "plain",
						Usage: "Read plain text from stdin instead of JSON hook event",
						Value: false,
					},
					&cli.StringFlag{
						Name:  "mode",
						Usage: "Reading mode: short (first line) or full (entire text). Legacy: first_line, full_text also supported",
						Value: "short",
					},
					&cli.StringFlag{
						Name:  "engine",
						Usage: "Voice engine priority: voicevox, aivisspeech (legacy local engines)",
						Value: "aivisspeech",
					},
					&cli.IntFlag{
						Name:  "speaker",
						Usage: "Speaker ID for voice engine (e.g., 3 for VOICEVOX ずんだもん, 888753760 for AivisSpeech)",
						Value: 0,
					},
					&cli.FloatFlag{
						Name:  "volume",
						Usage: "Volume scale for voice synthesis (0.0-2.0, default 1.0)",
						Value: 1.0,
					},
					&cli.IntFlag{
						Name:  "chars",
						Usage: "Max characters limit for 'full' mode (0 = unlimited)",
						Value: 0,
					},
					&cli.BoolFlag{
						Name:  "uuid",
						Usage: "Use UUID mode for complete message extraction",
						Value: false,
					},
					// Cloud provider flags
					&cli.StringFlag{
						Name:  "provider",
						Usage: "TTS provider: openai, elevenlabs, polly, voicevox, aivisspeech",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "api-key",
						Usage: "API key for cloud providers (or use environment variables)",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "voice",
						Usage: "Voice ID (e.g., alloy, echo, fable for OpenAI)",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "model",
						Usage: "Model to use (e.g., tts-1, tts-1-hd for OpenAI)",
						Value: "tts-1",
					},
					&cli.StringFlag{
						Name:  "format",
						Usage: "Audio format: mp3, wav, ogg, flac, aac",
						Value: "mp3",
					},
					&cli.StringFlag{
						Name:  "speed",
						Usage: "Speech speed (0.25-4.0)",
						Value: "1.0",
					},
					&cli.StringFlag{
						Name:  "output",
						Usage: "Output file path (default: temp file)",
						Value: "",
					},
					&cli.BoolFlag{
						Name:  "stdout",
						Usage: "Stream audio to stdout instead of playing",
						Value: false,
					},
					&cli.BoolFlag{
						Name:  "list-voices",
						Usage: "List available voices for the specified provider",
						Value: false,
					},
					// ElevenLabs-specific flags
					&cli.StringFlag{
						Name:  "stability",
						Usage: "Voice stability (0.0-1.0) for ElevenLabs",
						Value: "0.5",
					},
					&cli.StringFlag{
						Name:  "similarity-boost",
						Usage: "Similarity boost (0.0-1.0) for ElevenLabs",
						Value: "0.5",
					},
					&cli.StringFlag{
						Name:  "style",
						Usage: "Style setting (0.0-1.0) for ElevenLabs",
						Value: "0.0",
					},
					&cli.BoolFlag{
						Name:  "use-speaker-boost",
						Usage: "Use speaker boost for ElevenLabs",
						Value: true,
					},
					// Amazon Polly-specific flags
					&cli.StringFlag{
						Name:  "region",
						Usage: "AWS region for Polly (e.g., us-east-1, eu-west-1)",
						Value: "us-east-1",
					},
					&cli.StringFlag{
						Name:  "polly-engine",
						Usage: "Polly engine: neural, standard, long-form, generative",
						Value: "neural",
					},
					&cli.StringFlag{
						Name:  "sample-rate",
						Usage: "Audio sample rate: 8000, 16000, 22050, 24000",
						Value: "22050",
					},
					// Config file options
					&cli.StringFlag{
						Name:  "config",
						Usage: "Path to voice config file (default: .claude/voice.json or ~/.claude/voice.json)",
						Value: "",
					},
					&cli.BoolFlag{
						Name:  "no-config",
						Usage: "Ignore all config files",
						Value: false,
					},
				},
				Commands: []*cli.Command{
					{
						Name:  "config",
						Usage: "Manage voice configuration",
						Commands: []*cli.Command{
							{
								Name:   "show",
								Usage:  "Show current configuration (secrets masked)",
								Action: handleVoiceConfigShow,
							},
							{
								Name:   "validate",
								Usage:  "Validate configuration file",
								Action: handleVoiceConfigValidate,
							},
							{
								Name:   "init",
								Usage:  "Generate example configuration file",
								Action: handleVoiceConfigInit,
								Flags: []cli.Flag{
									&cli.BoolFlag{
										Name:  "global",
										Usage: "Create global config (~/.claude/voice.json)",
										Value: false,
									},
								},
							},
						},
					},
				},
			},
			{
				Name:    "notify",
				Aliases: []string{"notification_hook"},
				Usage:   "Handle Claude Code notifications and alert the user",
				Action:  handleNotify,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "voice",
						Usage: "Use voice synthesis for notifications",
						Value: true,
					},
					&cli.BoolFlag{
						Name:  "desktop",
						Usage: "Show desktop notifications",
						Value: true,
					},
				},
			},
			{
				Name:    "codex-notify",
				Aliases: []string{"codex_hook"},
				Usage:   "Execute as Codex notify hook (supports both Claude Code and Codex)",
				Action:  handleCodexNotify,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "voice",
						Usage: "Use voice synthesis for notifications",
						Value: true,
					},
					&cli.BoolFlag{
						Name:  "desktop",
						Usage: "Show desktop notifications",
						Value: true,
					},
				},
			},
		},
		Before: func(ctx context.Context, c *cli.Command) error {
			if c.Bool("verbose") {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			} else {
				zerolog.SetGlobalLevel(zerolog.InfoLevel)
			}
			return nil
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal().Err(err).Msg("Failed to run application")
	}
}

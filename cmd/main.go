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
				Usage:   "Initialize persona configuration in current project (interactive)",
				Action:  handleInit,
				Aliases: []string{"i"},
			},
			{
				Name:      "show",
				Usage:     "Show persona details (without args: show current persona)",
				Action:    handleShow,
				ArgsUsage: "[persona]",
			},
			{
				Name:      "create",
				Usage:     "Create a new persona (deprecated: use edit)",
				Action:    handleCreate,
				ArgsUsage: "<name>",
			},
			{
				Name:      "edit",
				Usage:     "Edit a persona (creates if not exists)",
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
				Usage:  "Interactive setup wizard (deprecated: use status --diagnose)",
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
				Usage:  "Show current ccpersona status (auto-diagnoses on errors)",
				Action: handleStatus,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "diagnose",
						Usage: "Force detailed diagnostics",
						Value: false,
					},
				},
			},
			{
				Name:   "doctor",
				Usage:  "Diagnose ccpersona configuration (deprecated: use status --diagnose)",
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
					// Input mode flags
					&cli.BoolFlag{
						Name:  "force",
						Usage: "Bypass the global mute gate for this invocation",
						Value: false,
					},
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
						Usage: "Reading mode: short (first line) or full (entire text)",
						Value: "short",
					},
					// Provider selection
					&cli.StringFlag{
						Name:  "provider",
						Usage: "TTS provider: voicevox, aivisspeech, openai, elevenlabs, polly, gcp",
						Value: "aivisspeech",
					},
					// Voice selection
					&cli.IntFlag{
						Name:  "speaker",
						Usage: "Speaker ID for local engines (VOICEVOX/AivisSpeech)",
						Value: 0,
					},
					&cli.StringFlag{
						Name:  "voice",
						Usage: "Voice ID for cloud providers (e.g., alloy for OpenAI)",
						Value: "",
					},
					// Output
					&cli.StringFlag{
						Name:  "output",
						Usage: "Output file path, or '-' for stdout (default: play audio)",
						Value: "",
					},
					&cli.BoolFlag{
						Name:  "list-voices",
						Usage: "List available voices for the specified provider",
						Value: false,
					},
					// Config
					&cli.StringFlag{
						Name:  "config",
						Usage: "Path to voice config file (default: .claude/voice.json)",
						Value: "",
					},
				},
				Commands: []*cli.Command{
					{
						Name:   "mute",
						Usage:  "Globally disable voice synthesis across all paths",
						Action: handleVoiceMute,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "reason",
								Usage: "Optional note stored alongside the mute marker",
								Value: "",
							},
						},
					},
					{
						Name:   "unmute",
						Usage:  "Lift the global voice synthesis mute",
						Action: handleVoiceUnmute,
					},
					{
						Name:   "status",
						Usage:  "Show the current global mute state",
						Action: handleVoiceStatus,
					},
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
				Name:   "mcp",
				Usage:  "Start as a stdio MCP server",
				Action: handleMCP,
			},
			{
				Name:    "notify",
				Aliases: []string{"notification_hook"},
				Usage:   "Handle notifications (auto-detects Claude Code, Codex, or Cursor)",
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
				Name:  "engine",
				Usage: "Manage TTS engine background services (VOICEVOX / AivisSpeech)",
				Commands: []*cli.Command{
					{
						Name:      "install",
						Usage:     "Install engine as a background service",
						ArgsUsage: "[voicevox|aivisspeech|all]",
						Action:    handleEngineInstall,
					},
					{
						Name:      "uninstall",
						Usage:     "Uninstall engine background service",
						ArgsUsage: "[voicevox|aivisspeech|all]",
						Action:    handleEngineUninstall,
					},
					{
						Name:      "start",
						Usage:     "Start engine background service",
						ArgsUsage: "[voicevox|aivisspeech|all]",
						Action:    handleEngineStart,
					},
					{
						Name:      "stop",
						Usage:     "Stop engine background service",
						ArgsUsage: "[voicevox|aivisspeech|all]",
						Action:    handleEngineStop,
					},
					{
						Name:   "status",
						Usage:  "Show engine service status",
						Action: handleEngineStatus,
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

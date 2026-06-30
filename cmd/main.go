package main

import (
	"context"
	"fmt"
	"os"

	"github.com/daikw/ccpersona/internal/cliui"
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

	app := newApp()
	if err := app.Run(context.Background(), os.Args); err != nil {
		// Command handlers own their user-facing messages; report the error
		// once in plain CLI form instead of a zerolog FTL record.
		fmt.Fprintf(os.Stderr, "%s %v\n", cliui.Failure("error:"), err)
		os.Exit(1)
	}
}

func newApp() *cli.Command {
	return &cli.Command{
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
				Name:    "list",
				Usage:   "List available personas (active persona marked with *)",
				Action:  handleList,
				Aliases: []string{"ls"},
			},
			{
				Name:      "set",
				Usage:     "Set the active persona for the current project (or --global)",
				Action:    handleSet,
				ArgsUsage: "<name>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "global",
						Aliases: []string{"g"},
						Usage:   "Write to the global ccpersona config (~/.agents/ccpersona.json)",
					},
				},
			},
			{
				Name:      "edit",
				Usage:     "Edit a persona (creates if not exists)",
				Action:    handleEdit,
				ArgsUsage: "<persona>",
			},
			{
				Name:   "config",
				Usage:  "Edit ccpersona configuration (or use config subcommands)",
				Action: handleConfig,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "global",
						Aliases: []string{"g"},
						Usage:   "Edit global settings",
					},
				},
				Commands: []*cli.Command{
					{
						Name:   "init",
						Usage:  "Create a ccpersona config file",
						Action: handleVoiceConfigInit,
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:    "global",
								Aliases: []string{"g"},
								Usage:   "Create global config (~/.agents/ccpersona.json)",
								Value:   false,
							},
						},
					},
					{
						Name:   "edit",
						Usage:  "Edit project or global ccpersona config",
						Action: handleConfig,
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:    "global",
								Aliases: []string{"g"},
								Usage:   "Edit global config (~/.agents/ccpersona.json)",
							},
						},
					},
					{
						Name:   "show",
						Usage:  "Show current ccpersona config with secrets masked",
						Action: handleVoiceConfigShow,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "config",
								Usage: "Path to ccpersona config file",
								Value: "",
							},
						},
					},
					{
						Name:   "status",
						Usage:  "Show current ccpersona status",
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
						Name:      "set-persona",
						Usage:     "Set the active persona in ccpersona config",
						Action:    handleSet,
						ArgsUsage: "<name>",
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:    "global",
								Aliases: []string{"g"},
								Usage:   "Write to global config (~/.agents/ccpersona.json)",
							},
						},
					},
					{
						Name:   "migrate",
						Usage:  "Migrate legacy persona/voice config files to .agents/ccpersona.json",
						Action: handleConfigMigrate,
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:    "global",
								Aliases: []string{"g"},
								Usage:   "Migrate global config to ~/.agents/ccpersona.json",
							},
							&cli.BoolFlag{
								Name:  "force",
								Usage: "Overwrite existing .agents/ccpersona.json",
								Value: false,
							},
						},
					},
				},
			},
			{
				Name:  "persona",
				Usage: "Manage persona markdown files",
				Commands: []*cli.Command{
					{
						Name:    "list",
						Usage:   "List persona markdown files",
						Action:  handleList,
						Aliases: []string{"ls"},
					},
					{
						Name:      "show",
						Usage:     "Show a persona markdown file",
						Action:    handlePersonaShow,
						ArgsUsage: "<name>",
					},
					{
						Name:      "edit",
						Usage:     "Edit a persona markdown file (creates if missing)",
						Action:    handleEdit,
						ArgsUsage: "<name>",
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
				Name:        "hook",
				Aliases:     []string{"user_prompt_submit_hook"},
				Usage:       "Execute as Claude Code hook (SessionStart recommended)",
				Description: "The 'user_prompt_submit_hook' alias is legacy and kept for backward compatibility; prefer 'hook' wired to the SessionStart event.",
				Action:      handleHook,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "platform",
						Usage: "Platform hint for ambiguous hook payloads: claude-code, codex, cursor",
						Value: "",
					},
				},
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
						Usage: "Path to ccpersona config file (default: .agents/ccpersona.json)",
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
										Usage: "Create global config (~/.agents/ccpersona.json)",
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
				Name:        "notify",
				Aliases:     []string{"notification_hook"},
				Usage:       "Handle notifications (auto-detects Claude Code, Codex, or Cursor)",
				Description: "The 'notification_hook' alias is legacy and kept for backward compatibility; prefer 'notify' wired to the Notification event.",
				Action:      handleNotify,
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
				Usage: "Manage TTS engine background services (built-in: VOICEVOX / AivisSpeech, plus engines defined in config)",
				Commands: []*cli.Command{
					{
						Name:      "install",
						Usage:     "Install engine as a background service",
						ArgsUsage: "[voicevox|aivisspeech|<name>|all]",
						Action:    handleEngineInstall,
					},
					{
						Name:      "uninstall",
						Usage:     "Uninstall engine background service",
						ArgsUsage: "[voicevox|aivisspeech|<name>|all]",
						Action:    handleEngineUninstall,
					},
					{
						Name:      "start",
						Usage:     "Start engine background service",
						ArgsUsage: "[voicevox|aivisspeech|<name>|all]",
						Action:    handleEngineStart,
					},
					{
						Name:      "stop",
						Usage:     "Stop engine background service",
						ArgsUsage: "[voicevox|aivisspeech|<name>|all]",
						Action:    handleEngineStop,
					},
					{
						Name:      "status",
						Usage:     "Show engine service status",
						ArgsUsage: "[voicevox|aivisspeech|<name>]",
						Action:    handleEngineStatus,
					},
				},
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			if c.Bool("verbose") {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			} else {
				zerolog.SetGlobalLevel(zerolog.InfoLevel)
			}
			return ctx, nil
		},
	}
}

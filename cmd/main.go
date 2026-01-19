package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/daikw/ccpersona/internal/hook"
	"github.com/daikw/ccpersona/internal/persona"
	"github.com/daikw/ccpersona/internal/voice"
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
						Usage: "Reading mode: first_line, line_limit, after_first, full_text, char_limit",
						Value: "first_line",
					},
					&cli.StringFlag{
						Name:  "engine",
						Usage: "Voice engine priority: voicevox, aivisspeech (legacy local engines)",
						Value: "aivisspeech",
					},
					&cli.IntFlag{
						Name:  "lines",
						Usage: "Max lines for line_limit mode",
						Value: 3,
					},
					&cli.IntFlag{
						Name:  "chars",
						Usage: "Max characters for char_limit mode",
						Value: 500,
					},
					&cli.BoolFlag{
						Name:  "uuid",
						Usage: "Use UUID mode for complete message extraction",
						Value: false,
					},
					// Cloud provider flags
					&cli.StringFlag{
						Name:  "provider",
						Usage: "TTS provider: openai, elevenlabs, voicevox, aivisspeech",
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

func handleInit(ctx context.Context, c *cli.Command) error {
	log.Info().Msg("Initializing persona configuration...")

	// Check if persona.json already exists
	config, err := persona.LoadConfig(".")
	if err != nil {
		return err
	}

	if config != nil {
		log.Warn().Msg("Persona configuration already exists")
		return nil
	}

	// Create default configuration
	defaultConfig := persona.GetDefaultConfig()

	// Save configuration
	if err := persona.SaveConfig(".", defaultConfig); err != nil {
		return err
	}

	log.Info().Msg("Persona configuration initialized successfully")
	fmt.Println("Created .claude/persona.json with default configuration")
	return nil
}

func handleList(ctx context.Context, c *cli.Command) error {
	manager, err := persona.NewManager()
	if err != nil {
		return err
	}

	personas, err := manager.ListPersonas()
	if err != nil {
		return err
	}

	if len(personas) == 0 {
		fmt.Println("No personas found. Create one with 'ccpersona create <name>'")
		return nil
	}

	fmt.Println("Available personas:")
	for _, p := range personas {
		fmt.Printf("  - %s\n", p)
	}

	return nil
}

func handleCurrent(ctx context.Context, c *cli.Command) error {
	manager, err := persona.NewManager()
	if err != nil {
		return err
	}

	current, err := manager.GetCurrentPersona()
	if err != nil {
		return err
	}

	fmt.Printf("Current persona: %s\n", current)
	return nil
}

func handleSet(ctx context.Context, c *cli.Command) error {
	personaName := c.Args().Get(0)
	if personaName == "" {
		return fmt.Errorf("persona name is required")
	}

	manager, err := persona.NewManager()
	if err != nil {
		return err
	}

	// Check if persona exists
	if !manager.PersonaExists(personaName) {
		return fmt.Errorf("persona '%s' does not exist", personaName)
	}

	// Update project configuration
	config, err := persona.LoadConfig(".")
	if err != nil {
		return err
	}

	if config == nil {
		config = persona.GetDefaultConfig()
	}

	config.Name = personaName

	if err := persona.SaveConfig(".", config); err != nil {
		return err
	}

	// Apply persona
	if err := manager.ApplyPersona(personaName); err != nil {
		return err
	}

	fmt.Printf("Set active persona to: %s\n", personaName)
	return nil
}

func handleShow(ctx context.Context, c *cli.Command) error {
	personaName := c.Args().Get(0)
	if personaName == "" {
		return fmt.Errorf("persona name is required")
	}

	manager, err := persona.NewManager()
	if err != nil {
		return err
	}

	content, err := manager.ReadPersona(personaName)
	if err != nil {
		return err
	}

	fmt.Println(content)
	return nil
}

func handleCreate(ctx context.Context, c *cli.Command) error {
	name := c.Args().Get(0)
	if name == "" {
		return fmt.Errorf("persona name is required")
	}

	manager, err := persona.NewManager()
	if err != nil {
		return err
	}

	if err := manager.CreatePersona(name); err != nil {
		return err
	}

	fmt.Printf("Created new persona: %s\n", name)
	fmt.Printf("Edit it with: ccpersona edit %s\n", name)
	return nil
}

func handleEdit(ctx context.Context, c *cli.Command) error {
	personaName := c.Args().Get(0)
	if personaName == "" {
		return fmt.Errorf("persona name is required")
	}

	manager, err := persona.NewManager()
	if err != nil {
		return err
	}

	if !manager.PersonaExists(personaName) {
		return fmt.Errorf("persona '%s' does not exist", personaName)
	}

	// Get the path to the persona file
	path := manager.GetPersonaPath(personaName)

	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Default to vi
	}

	// Open editor
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	fmt.Printf("Edited persona: %s\n", personaName)
	return nil
}

func handleConfig(ctx context.Context, c *cli.Command) error {
	var configPath string
	var config *persona.Config
	var err error

	if c.Bool("global") {
		// Global configuration
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir := filepath.Join(homeDir, ".claude")
		configPath = filepath.Join(configDir, "persona.json")

		// Ensure directory exists
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// Load or create global config
		config, err = persona.LoadConfig(homeDir)
		if err != nil || config == nil {
			config = persona.GetDefaultConfig()
			if err := persona.SaveConfig(homeDir, config); err != nil {
				return fmt.Errorf("failed to create global config: %w", err)
			}
		}
	} else {
		// Project configuration
		configPath = filepath.Join(".claude", "persona.json")

		// Load or create project config
		config, err = persona.LoadConfig(".")
		if err != nil {
			return err
		}
		if config == nil {
			return fmt.Errorf("no project configuration found. Run 'ccpersona init' first")
		}
	}

	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Default to vi
	}

	// Open editor
	cmd := exec.Command(editor, configPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	if c.Bool("global") {
		fmt.Println("Edited global configuration")
	} else {
		fmt.Println("Edited project configuration")
	}
	return nil
}

func handleHook(ctx context.Context, c *cli.Command) error {
	// Suppress normal output when running as hook
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	// Try to read hook event data from stdin
	event, err := hook.ReadUserPromptSubmitEvent()
	if err != nil {
		// Fallback to legacy behavior if no stdin data
		log.Debug().Err(err).Msg("No hook event data from stdin, using legacy mode")
	} else {
		// Use session ID from hook event if available
		if event.SessionID != "" {
			_ = os.Setenv("CLAUDE_SESSION_ID", event.SessionID)
		}
		log.Debug().
			Str("session_id", event.SessionID).
			Str("prompt", event.Prompt).
			Str("cwd", event.CWD).
			Msg("Received UserPromptSubmit hook event")
	}

	if err := persona.HandleSessionStart(); err != nil {
		// Log error but don't fail the hook
		log.Error().Err(err).Msg("Failed to handle session start")
		return nil
	}
	return nil
}

func handleVoice(ctx context.Context, c *cli.Command) error {
	// Create voice config from flags
	voiceConfig := voice.DefaultConfig()
	voiceConfig.ReadingMode = c.String("mode")
	voiceConfig.EnginePriority = c.String("engine")
	voiceConfig.MaxLines = int(c.Int("lines"))
	voiceConfig.MaxChars = int(c.Int("chars"))
	voiceConfig.UUIDMode = c.Bool("uuid")

	// Load persona config and apply voice settings
	personaConfig, err := persona.LoadConfig(".")
	if err == nil && personaConfig != nil && personaConfig.Voice != nil {
		if personaConfig.Voice.Engine != "" {
			voiceConfig.EnginePriority = personaConfig.Voice.Engine
		}
		if personaConfig.Voice.SpeakerID > 0 {
			// Apply speaker ID to the appropriate engine based on priority
			if voiceConfig.EnginePriority == voice.EngineAivisSpeech {
				voiceConfig.AivisSpeechSpeaker = int64(personaConfig.Voice.SpeakerID)
			} else {
				voiceConfig.VoicevoxSpeaker = personaConfig.Voice.SpeakerID
			}
		}
	}

	// Create voice manager
	manager := voice.NewVoiceManager(voiceConfig)

	// Handle list voices
	if c.Bool("list-voices") {
		providerName := c.String("provider")
		voices, err := manager.ListVoices(ctx, providerName)
		if err != nil {
			return fmt.Errorf("failed to list voices: %w", err)
		}

		if len(voices) == 0 {
			fmt.Println("No voices available")
			return nil
		}

		fmt.Printf("Available voices for provider '%s':\n", providerName)
		for _, v := range voices {
			fmt.Printf("  - %s (%s) - %s\n", v.ID, v.Language, v.Description)
		}
		return nil
	}

	var text string

	if c.Bool("transcript") {
		// User explicitly wants to read from transcript
		log.Debug().Msg("Reading from latest transcript (--transcript flag)")

		reader := voice.NewTranscriptReader(voiceConfig)
		transcriptPath, err := reader.FindLatestTranscript()
		if err != nil {
			return fmt.Errorf("failed to find transcript: %w", err)
		}

		log.Debug().Str("path", transcriptPath).Msg("Using transcript file")

		// Get latest assistant message
		text, err = reader.GetLatestAssistantMessage(transcriptPath)
		if err != nil {
			// If no text content found (e.g., tool_use only messages), skip voice synthesis
			if strings.Contains(err.Error(), "no assistant message found") {
				log.Info().Msg("No text content found in latest assistant message, skipping voice synthesis")
				return nil
			}
			return fmt.Errorf("failed to get assistant message: %w", err)
		}

		// Process text according to reading mode
		text = reader.ProcessText(text)

	} else if c.Bool("plain") {
		// Read plain text from stdin
		log.Debug().Msg("Reading plain text from stdin")

		textBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}

		text = string(textBytes)
		if text == "" {
			return fmt.Errorf("no text provided via stdin")
		}

	} else {
		// Default: try to read Stop hook event from stdin
		log.Debug().Msg("Reading Stop hook event from stdin")

		event, err := hook.ReadStopEvent()
		if err != nil {
			return fmt.Errorf("failed to read Stop hook event from stdin: %w (use --plain flag for plain text input)", err)
		}

		if event.TranscriptPath == "" {
			return fmt.Errorf("no transcript path in Stop hook event")
		}

		log.Debug().
			Str("session_id", event.SessionID).
			Str("transcript_path", event.TranscriptPath).
			Bool("stop_hook_active", event.StopHookActive).
			Msg("Received Stop hook event")

		// Create transcript reader
		reader := voice.NewTranscriptReader(voiceConfig)

		// Get latest assistant message from transcript
		text, err = reader.GetLatestAssistantMessage(event.TranscriptPath)
		if err != nil {
			// If no text content found (e.g., tool_use only messages), skip voice synthesis
			if strings.Contains(err.Error(), "no assistant message found") {
				log.Info().Msg("No text content found in latest assistant message, skipping voice synthesis")
				return nil
			}
			return fmt.Errorf("failed to get assistant message: %w", err)
		}

		// Process text according to reading mode
		text = reader.ProcessText(text)
	}

	// Strip markdown if mdstrip is available
	text = voice.StripMarkdown(text)

	fmt.Fprintf(os.Stderr, "📢 Reading text: %s\n", text)

	// Parse speed as float
	speedStr := c.String("speed")
	speed := 1.0
	if speedStr != "" {
		if parsed, err := strconv.ParseFloat(speedStr, 64); err == nil {
			speed = parsed
		}
	}

	// Parse ElevenLabs-specific settings as floats
	stability := 0.5
	if stabilityStr := c.String("stability"); stabilityStr != "" {
		if parsed, err := strconv.ParseFloat(stabilityStr, 64); err == nil {
			stability = parsed
		}
	}

	similarityBoost := 0.5
	if similarityBoostStr := c.String("similarity-boost"); similarityBoostStr != "" {
		if parsed, err := strconv.ParseFloat(similarityBoostStr, 64); err == nil {
			similarityBoost = parsed
		}
	}

	style := 0.0
	if styleStr := c.String("style"); styleStr != "" {
		if parsed, err := strconv.ParseFloat(styleStr, 64); err == nil {
			style = parsed
		}
	}

	// Set up voice options
	options := voice.VoiceOptions{
		Provider:        c.String("provider"),
		Voice:           c.String("voice"),
		Speed:           speed,
		Format:          c.String("format"),
		Model:           c.String("model"),
		APIKey:          c.String("api-key"),
		Stability:       stability,
		SimilarityBoost: similarityBoost,
		Style:           style,
		UseSpeakerBoost: c.Bool("use-speaker-boost"),
		OutputPath:      c.String("output"),
		PlayAudio:       !c.Bool("stdout"),
		ToStdout:        c.Bool("stdout"),
	}

	// Synthesize voice
	audioFile, err := manager.Synthesize(ctx, text, options)
	if err != nil {
		return fmt.Errorf("failed to synthesize voice: %w", err)
	}

	// Handle output
	if options.ToStdout {
		// Audio was already streamed to stdout
		return nil
	}

	// Play audio if requested
	if options.PlayAudio {
		if err := manager.PlayAudio(audioFile); err != nil {
			return fmt.Errorf("failed to play audio: %w", err)
		}
	}

	if audioFile != "" {
		fmt.Fprintf(os.Stderr, "🎵 Audio saved to: %s\n", audioFile)
	}
	fmt.Fprintf(os.Stderr, "✅ Voice synthesis complete\n")
	return nil
}

func handleNotify(ctx context.Context, c *cli.Command) error {
	// Read notification event from stdin
	event, err := hook.ReadNotificationEvent()
	if err != nil {
		return fmt.Errorf("failed to read notification event: %w", err)
	}

	log.Info().
		Str("session_id", event.SessionID).
		Str("message", event.Message).
		Msg("Received notification")

	// Determine notification urgency based on message content
	urgency := "normal"

	// Analyze message content for urgency level only
	switch {
	case strings.Contains(strings.ToLower(event.Message), "permission"):
		urgency = "critical"

	case strings.Contains(strings.ToLower(event.Message), "idle"):
		urgency = "low"

	case strings.Contains(strings.ToLower(event.Message), "error"):
		urgency = "high"
	}

	// Desktop notification (if enabled)
	if c.Bool("desktop") {
		if err := showDesktopNotification(event.Message, urgency); err != nil {
			log.Warn().Err(err).Msg("Failed to show desktop notification")
		}
	}

	// Voice notification (if enabled)
	if c.Bool("voice") {
		// Load persona config to get voice settings
		config, _ := persona.LoadConfig(".")
		voiceConfig := voice.DefaultConfig()

		if config != nil && config.Voice != nil {
			if config.Voice.Engine != "" {
				voiceConfig.EnginePriority = config.Voice.Engine
			}
			if config.Voice.SpeakerID > 0 {
				// Apply speaker ID to the appropriate engine based on priority
				if voiceConfig.EnginePriority == voice.EngineAivisSpeech {
					voiceConfig.AivisSpeechSpeaker = int64(config.Voice.SpeakerID)
				} else {
					voiceConfig.VoicevoxSpeaker = config.Voice.SpeakerID
				}
			}
		}

		// Synthesize and play voice with original message
		engine := voice.NewVoiceEngine(voiceConfig)
		audioFile, err := engine.Synthesize(event.Message)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to synthesize voice")
		} else {
			if err := engine.Play(audioFile); err != nil {
				log.Warn().Err(err).Msg("Failed to play audio")
			}
		}
	}

	return nil
}

func showDesktopNotification(message, urgency string) error {
	title := "Claude Code"

	switch runtime.GOOS {
	case "darwin":
		// macOS notification using osascript
		script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)
		cmd := exec.Command("osascript", "-e", script)
		return cmd.Run()

	case "linux":
		// Linux notification using notify-send
		cmd := exec.Command("notify-send", "-u", urgency, title, message)
		return cmd.Run()

	case "windows":
		// Windows notification using PowerShell
		script := fmt.Sprintf(`
			[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
			[Windows.UI.Notifications.ToastNotification, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
			[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
			
			$template = @"
			<toast>
				<visual>
					<binding template="ToastText02">
						<text id="1">%s</text>
						<text id="2">%s</text>
					</binding>
				</visual>
			</toast>
"@
			
			$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
			$xml.LoadXml($template)
			$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
			[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("Claude Code").Show($toast)
		`, title, message)
		cmd := exec.Command("powershell", "-Command", script)
		return cmd.Run()

	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

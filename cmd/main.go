package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
						Usage: "Voice engine priority: voicevox, aivisspeech (legacy)",
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
					// New cloud provider flags
					&cli.StringFlag{
						Name:  "provider",
						Usage: "TTS provider: local, openai, elevenlabs, polly, gcp",
						Value: "local",
					},
					&cli.StringFlag{
						Name:  "api-key",
						Usage: "API key for cloud providers (or use environment variables)",
					},
					&cli.StringFlag{
						Name:  "voice",
						Usage: "Voice ID or name (provider-specific)",
					},
					&cli.StringFlag{
						Name:  "format",
						Usage: "Audio format: mp3, wav, ogg, flac, aac",
					},
					&cli.StringFlag{
						Name:  "region",
						Usage: "AWS region for Polly (default: us-east-1)",
						Value: "us-east-1",
					},
					&cli.StringFlag{
						Name:  "project-id",
						Usage: "Google Cloud project ID for GCP TTS",
					},
					&cli.StringFlag{
						Name:  "output",
						Aliases: []string{"o"},
						Usage: "Output file path (default: play audio)",
					},
					&cli.BoolFlag{
						Name:  "stdout",
						Usage: "Stream audio to stdout instead of playing",
						Value: false,
					},
					&cli.Float64Flag{
						Name:  "speed",
						Usage: "Speech speed (0.25-4.0, provider dependent)",
						Value: 1.0,
					},
					&cli.BoolFlag{
						Name:  "list-voices",
						Usage: "List available voices for the selected provider",
						Value: false,
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
	// Handle list-voices flag first
	if c.Bool("list-voices") {
		return handleListVoices(ctx, c)
	}

	// Create voice config from flags
	config := voice.DefaultConfig()
	config.ReadingMode = c.String("mode")
	config.MaxLines = int(c.Int("lines"))
	config.MaxChars = int(c.Int("chars"))
	config.UUIDMode = c.Bool("uuid")

	// Configure provider based on flags
	providerName := c.String("provider")
	if err := configureProvider(config, c, providerName); err != nil {
		return fmt.Errorf("failed to configure provider: %w", err)
	}

	// Create voice manager
	manager, err := voice.NewVoiceManager(config)
	if err != nil {
		return fmt.Errorf("failed to create voice manager: %w", err)
	}

	// Check if provider is available
	if !manager.IsAvailable(ctx) {
		return fmt.Errorf("voice provider '%s' is not available", manager.GetProviderName())
	}

	var text string

	if c.Bool("transcript") {
		// User explicitly wants to read from transcript
		log.Debug().Msg("Reading from latest transcript (--transcript flag)")
		
		reader := voice.NewTranscriptReader(config)
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
		reader := voice.NewTranscriptReader(config)
		
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

	// Strip markdown if mdstrip is available (only for local providers to maintain compatibility)
	if providerName == "local" {
		text = voice.StripMarkdown(text)
	}

	fmt.Fprintf(os.Stderr, "📢 Reading text with %s: %s\n", manager.GetProviderName(), text)

	// Create synthesis options from flags
	options := &voice.SynthesizeOptions{
		Voice:          c.String("voice"),
		Speed:          float32(c.Float64("speed")),
		Format:         voice.AudioFormat(c.String("format")),
		StreamToStdout: c.Bool("stdout"),
		OutputFile:     c.String("output"),
	}

	// Handle different output modes
	if c.Bool("stdout") {
		// Stream to stdout
		if err := manager.SynthesizeToStdout(ctx, text, options); err != nil {
			return fmt.Errorf("failed to synthesize to stdout: %w", err)
		}
	} else if outputFile := c.String("output"); outputFile != "" {
		// Save to file
		if err := manager.SynthesizeToFile(ctx, text, outputFile, options); err != nil {
			return fmt.Errorf("failed to synthesize to file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "✅ Audio saved to %s\n", outputFile)
	} else {
		// Play audio (default)
		if err := manager.SynthesizeAndPlay(ctx, text, options); err != nil {
			return fmt.Errorf("failed to synthesize and play: %w", err)
		}
		fmt.Fprintf(os.Stderr, "✅ Voice synthesis complete\n")
	}

	return nil
}

// configureProvider configures the TTS provider based on CLI flags
func configureProvider(config *voice.Config, c *cli.Command, providerName string) error {
	if config.Provider == nil {
		config.Provider = voice.DefaultProviderConfig()
	}

	switch providerName {
	case "local":
		config.Provider.Provider = voice.ProviderLocal
		if config.Provider.Local == nil {
			config.Provider.Local = &voice.LocalConfig{}
		}
		// Use legacy engine setting for backward compatibility
		config.Provider.Local.Engine = c.String("engine")
		config.Provider.Local.VoicevoxSpeaker = config.VoicevoxSpeaker
		config.Provider.Local.AivisSpeechSpeaker = config.AivisSpeechSpeaker

	case "openai":
		config.Provider.Provider = voice.ProviderOpenAI
		config.Provider.APIKey = getAPIKey(c, "OPENAI_API_KEY")
		if config.Provider.APIKey == "" {
			return fmt.Errorf("OpenAI API key is required (use --api-key or set OPENAI_API_KEY environment variable)")
		}
		if config.Provider.OpenAI == nil {
			config.Provider.OpenAI = &voice.OpenAIConfig{
				Model:  "tts-1",
				Voice:  "alloy",
				Speed:  1.0,
				Format: "mp3",
			}
		}

	case "elevenlabs":
		config.Provider.Provider = voice.ProviderElevenLabs
		config.Provider.APIKey = getAPIKey(c, "ELEVENLABS_API_KEY")
		if config.Provider.APIKey == "" {
			return fmt.Errorf("ElevenLabs API key is required (use --api-key or set ELEVENLABS_API_KEY environment variable)")
		}
		if config.Provider.ElevenLabs == nil {
			config.Provider.ElevenLabs = &voice.ElevenLabsConfig{
				VoiceID: "21m00Tcm4TlvDq8ikWAM", // Rachel
				Model:   "eleven_monolingual_v1",
				VoiceSettings: &voice.VoiceSettings{
					Stability:       0.5,
					SimilarityBoost: 0.5,
					Style:           0.0,
					UseSpeakerBoost: true,
				},
			}
		}

	case "polly":
		config.Provider.Provider = voice.ProviderPolly
		config.Provider.Region = c.String("region")
		if config.Provider.Polly == nil {
			config.Provider.Polly = &voice.PollyConfig{
				VoiceID:      "Joanna",
				Engine:       "neural",
				LanguageCode: "en-US",
				OutputFormat: "mp3",
				SampleRate:   "22050",
			}
		}
		// AWS credentials will be loaded from environment or IAM role

	case "gcp":
		config.Provider.Provider = voice.ProviderGCP
		config.Provider.ProjectID = c.String("project-id")
		if config.Provider.ProjectID == "" {
			config.Provider.ProjectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
		}
		if config.Provider.ProjectID == "" {
			return fmt.Errorf("Google Cloud project ID is required (use --project-id or set GOOGLE_CLOUD_PROJECT environment variable)")
		}
		if config.Provider.GCP == nil {
			config.Provider.GCP = &voice.GCPConfig{
				VoiceName:       "en-US-Wavenet-D",
				LanguageCode:    "en-US",
				SsmlGender:      "NEUTRAL",
				AudioEncoding:   "MP3",
				SampleRateHertz: 24000,
				SpeakingRate:    1.0,
				Pitch:           0.0,
				VolumeGainDb:    0.0,
			}
		}

	default:
		return fmt.Errorf("unknown provider: %s (supported: local, openai, elevenlabs, polly, gcp)", providerName)
	}

	return nil
}

// getAPIKey gets an API key from CLI flag or environment variable
func getAPIKey(c *cli.Command, envVar string) string {
	if apiKey := c.String("api-key"); apiKey != "" {
		return apiKey
	}
	return os.Getenv(envVar)
}

// handleListVoices lists available voices for the selected provider
func handleListVoices(ctx context.Context, c *cli.Command) error {
	// Create voice config
	config := voice.DefaultConfig()
	providerName := c.String("provider")
	
	if err := configureProvider(config, c, providerName); err != nil {
		return fmt.Errorf("failed to configure provider: %w", err)
	}

	// Create voice manager
	manager, err := voice.NewVoiceManager(config)
	if err != nil {
		return fmt.Errorf("failed to create voice manager: %w", err)
	}

	// Check if provider is available
	if !manager.IsAvailable(ctx) {
		return fmt.Errorf("voice provider '%s' is not available", manager.GetProviderName())
	}

	fmt.Printf("Available voices for %s:\n\n", manager.GetProviderName())

	// Get provider-specific voice list
	factory := voice.NewProviderFactory(config.Provider)
	provider, err := factory.CreateProvider(ctx)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	switch p := provider.(type) {
	case *voice.OpenAIProvider:
		voices := p.GetSupportedVoices()
		for _, voiceID := range voices {
			fmt.Printf("  %s\n", voiceID)
		}

	case *voice.ElevenLabsProvider:
		voices, err := p.GetVoices(ctx)
		if err != nil {
			return fmt.Errorf("failed to get ElevenLabs voices: %w", err)
		}
		for _, voice := range voices {
			fmt.Printf("  %s - %s (%s)\n", voice.VoiceID, voice.Name, voice.Category)
		}

	case *voice.PollyProvider:
		voices, err := p.GetVoices(ctx)
		if err != nil {
			return fmt.Errorf("failed to get Polly voices: %w", err)
		}
		for _, voice := range voices {
			fmt.Printf("  %s - %s (%s, %s)\n", voice.Id, voice.Name, voice.Gender, voice.LanguageCode)
		}

	case *voice.GCPProvider:
		voices, err := p.GetVoices(ctx)
		if err != nil {
			return fmt.Errorf("failed to get GCP voices: %w", err)
		}
		for _, voice := range voices {
			langs := strings.Join(voice.LanguageCodes, ", ")
			fmt.Printf("  %s (%s, %s)\n", voice.Name, langs, voice.SsmlGender)
		}

	case *voice.LocalProvider:
		fmt.Printf("Local engines use numeric speaker IDs:\n")
		fmt.Printf("  VOICEVOX: 3 (ずんだもん), 1 (四国めたん), etc.\n")
		fmt.Printf("  AivisSpeech: 1512153248 (default), etc.\n")
		fmt.Printf("Use the original voice command flags: --engine and numeric speaker IDs\n")

	default:
		fmt.Printf("Voice listing not implemented for this provider\n")
	}

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
				voiceConfig.VoicevoxSpeaker = config.Voice.SpeakerID
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
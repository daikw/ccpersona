package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/daikw/ccpersona/internal/hook"
	"github.com/daikw/ccpersona/internal/persona"
	"github.com/daikw/ccpersona/internal/voice"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

func handleVoice(ctx context.Context, c *cli.Command) error {
	// Create voice config from flags
	voiceConfig := voice.DefaultConfig()
	voiceConfig.ReadingMode = c.String("mode")
	voiceConfig.EnginePriority = c.String("engine")
	voiceConfig.MaxChars = int(c.Int("chars"))
	voiceConfig.UUIDMode = c.Bool("uuid")
	voiceConfig.VolumeScale = c.Float("volume")

	// Load voice config file (if not disabled)
	fileConfig := loadVoiceConfig(c)
	if fileConfig != nil {
		// Apply file config as base, CLI flags override
		provider := fileConfig.GetEffectiveProvider(c.String("provider"))
		if providerConfig := fileConfig.GetProviderConfig(provider); providerConfig != nil {
			// Apply local engine settings
			if provider == "voicevox" || provider == "aivisspeech" {
				if providerConfig.Speaker > 0 && c.Int("speaker") == 0 {
					if provider == "aivisspeech" {
						voiceConfig.AivisSpeechSpeaker = int64(providerConfig.Speaker)
					} else {
						voiceConfig.VoicevoxSpeaker = providerConfig.Speaker
					}
				}
				if providerConfig.Volume > 0 && c.Float("volume") == 1.0 {
					voiceConfig.VolumeScale = providerConfig.Volume
				}
			}
			log.Debug().Str("provider", provider).Msg("Applied voice config from file")
		}
	}

	// Apply speaker ID from CLI flag (takes priority over config file)
	cliSpeakerID := c.Int("speaker")
	if cliSpeakerID > 0 {
		voiceConfig.VoicevoxSpeaker = int(cliSpeakerID)
		voiceConfig.AivisSpeechSpeaker = cliSpeakerID
	}

	// Load persona config and apply voice settings (if CLI flag not specified)
	personaConfig, err := persona.LoadConfig(".")
	if err == nil && personaConfig != nil && personaConfig.Voice != nil {
		if personaConfig.Voice.Engine != "" {
			voiceConfig.EnginePriority = personaConfig.Voice.Engine
		}
		if personaConfig.Voice.SpeakerID > 0 && cliSpeakerID == 0 {
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
		Region:          c.String("region"),
		Engine:          c.String("polly-engine"),
		SampleRate:      c.String("sample-rate"),
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

// Voice config management handlers

func handleVoiceConfigShow(ctx context.Context, c *cli.Command) error {
	loader := voice.NewVoiceConfigLoader()

	// Try to load config
	workDir, _ := os.Getwd()
	config, err := loader.LoadConfig(workDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if config == nil {
		fmt.Println("No voice configuration file found.")
		fmt.Println("\nSearched locations:")
		fmt.Println("  - .claude/voice.json (project)")
		fmt.Println("  - ~/.claude/voice.json (global)")
		fmt.Println("\nRun 'ccpersona voice config init' to create one.")
		return nil
	}

	// Mask secrets before displaying
	masked := config.MaskSecrets()

	// Pretty print
	output, err := json.MarshalIndent(masked, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format config: %w", err)
	}

	fmt.Println("Current voice configuration (secrets masked):")
	fmt.Println(string(output))

	return nil
}

func handleVoiceConfigValidate(ctx context.Context, c *cli.Command) error {
	loader := voice.NewVoiceConfigLoader()

	// Try to load config
	workDir, _ := os.Getwd()
	config, err := loader.LoadConfig(workDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if config == nil {
		fmt.Println("No voice configuration file found.")
		return nil
	}

	// Validate
	errors := config.Validate()
	if len(errors) == 0 {
		fmt.Println("✅ Configuration is valid.")
		return nil
	}

	fmt.Println("❌ Configuration has errors:")
	for _, err := range errors {
		fmt.Printf("  - %s\n", err)
	}
	return fmt.Errorf("configuration validation failed")
}

func handleVoiceConfigInit(ctx context.Context, c *cli.Command) error {
	var configPath string

	if c.Bool("global") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(homeDir, ".claude", "voice.json")
	} else {
		configPath = ".claude/voice.json"
	}

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config file already exists: %s", configPath)
	}

	// Create directory if needed
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate example config
	example := voice.GenerateExampleConfig()

	// Write with secure permissions
	if err := os.WriteFile(configPath, []byte(example), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("✅ Created voice configuration: %s\n", configPath)
	fmt.Println("\nEdit the file to configure your preferred voice providers.")
	fmt.Println("Use ${ENV_VAR} syntax for sensitive values like API keys.")

	return nil
}

// Helper function for loading voice config in handlers
func loadVoiceConfig(c *cli.Command) *voice.VoiceConfigFile {
	if c.Bool("no-config") {
		return nil
	}

	loader := voice.NewVoiceConfigLoader()

	// Custom config path
	if configPath := c.String("config"); configPath != "" {
		config, err := loader.LoadFromPath(configPath)
		if err != nil {
			log.Warn().Err(err).Str("path", configPath).Msg("Failed to load custom config")
			return nil
		}
		return config
	}

	// Default locations
	workDir, _ := os.Getwd()
	config, _ := loader.LoadConfig(workDir)
	return config
}

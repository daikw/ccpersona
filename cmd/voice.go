package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/daikw/ccpersona/internal/hook"
	"github.com/daikw/ccpersona/internal/persona"
	"github.com/daikw/ccpersona/internal/voice"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

func handleVoice(ctx context.Context, c *cli.Command) error {
	// Create voice config with defaults
	voiceConfig := voice.DefaultConfig()
	voiceConfig.ReadingMode = c.String("mode")

	// Load voice config file
	fileConfig := loadVoiceConfig(c)

	// Determine provider (CLI flag takes priority, then config file, then default)
	provider := c.String("provider")
	if fileConfig != nil && provider == "aivisspeech" {
		// If using default, check if config has a different default
		if effectiveProvider := fileConfig.GetEffectiveProvider(""); effectiveProvider != "" {
			provider = effectiveProvider
		}
	}

	// Apply provider to engine priority for local engines
	if provider == "voicevox" || provider == "aivisspeech" {
		voiceConfig.EnginePriority = provider
	}

	// Apply settings from config file
	if fileConfig != nil {
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
				if providerConfig.Volume > 0 {
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
	personaConfig, err := persona.LoadConfigWithFallback()
	if err == nil && personaConfig != nil && personaConfig.Voice != nil {
		if personaConfig.Voice.Provider != "" && c.String("provider") == "aivisspeech" {
			voiceConfig.EnginePriority = personaConfig.Voice.Provider
			provider = personaConfig.Voice.Provider
		}
		if personaConfig.Voice.Speaker > 0 && cliSpeakerID == 0 {
			if voiceConfig.EnginePriority == voice.EngineAivisSpeech {
				voiceConfig.AivisSpeechSpeaker = int64(personaConfig.Voice.Speaker)
			} else {
				voiceConfig.VoicevoxSpeaker = personaConfig.Voice.Speaker
			}
		}
		if personaConfig.Voice.Volume > 0 {
			voiceConfig.VolumeScale = personaConfig.Voice.Volume
		}
		if personaConfig.Voice.Speed > 0 {
			voiceConfig.SpeedScale = personaConfig.Voice.Speed
		}
		log.Debug().
			Str("persona", personaConfig.Name).
			Str("voice_provider", personaConfig.Voice.Provider).
			Int("voice_speaker", personaConfig.Voice.Speaker).
			Float64("voice_volume", personaConfig.Voice.Volume).
			Float64("voice_speed", personaConfig.Voice.Speed).
			Msg("Applied persona voice config")
	} else {
		log.Debug().Msg("No persona voice config found, using defaults or file config")
	}

	// Create voice manager
	manager := voice.NewVoiceManager(voiceConfig)

	// Handle list voices
	if c.Bool("list-voices") {
		voices, err := manager.ListVoices(ctx, provider)
		if err != nil {
			return fmt.Errorf("failed to list voices: %w", err)
		}

		if len(voices) == 0 {
			fmt.Println("No voices available")
			return nil
		}

		fmt.Printf("Available voices for provider '%s':\n", provider)
		for _, v := range voices {
			fmt.Printf("  - %s (%s) - %s\n", v.ID, v.Language, v.Description)
		}
		return nil
	}

	var text string
	var dedupSessionID string // set when running as stop hook

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
		dedupSessionID = event.SessionID
	}

	// Strip markdown if mdstrip is available
	text = voice.StripMarkdown(text)
	text = strings.TrimSpace(text)

	if text == "" {
		log.Debug().Msg("No text to synthesize after processing, skipping")
		return nil
	}

	// Skip duplicate messages when running as stop hook
	if dedupSessionID != "" {
		dedup := voice.NewDedupTracker(dedupSessionID)
		if dedup.IsDuplicate(text) {
			log.Debug().Msg("Skipping duplicate voice synthesis")
			return nil
		}
		defer func() {
			dedup.Record(text)
			go dedup.Cleanup()
		}()
	}

	fmt.Fprintf(os.Stderr, "📢 Reading text: %s\n", text)

	// Build voice options from config file
	options := buildVoiceOptions(c, provider, fileConfig)

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

// buildVoiceOptions creates VoiceOptions from CLI flags and config file
func buildVoiceOptions(c *cli.Command, provider string, fileConfig *voice.VoiceConfigFile) voice.VoiceOptions {
	output := c.String("output")
	toStdout := output == "-"
	playAudio := output == "" // Play if no output specified

	options := voice.VoiceOptions{
		Provider:   provider,
		Voice:      c.String("voice"),
		OutputPath: output,
		PlayAudio:  playAudio,
		ToStdout:   toStdout,
		// Defaults
		Speed:           1.0,
		Format:          "mp3",
		Model:           "tts-1",
		Stability:       0.5,
		SimilarityBoost: 0.5,
		Style:           0.0,
		UseSpeakerBoost: true,
		Region:          "us-east-1",
		Engine:          "neural",
		SampleRate:      "22050",
	}

	// If stdout, don't save to output path
	if toStdout {
		options.OutputPath = ""
	}

	// Apply settings from config file
	if fileConfig != nil {
		if providerConfig := fileConfig.GetProviderConfig(provider); providerConfig != nil {
			// Common settings
			if providerConfig.Voice != "" && options.Voice == "" {
				options.Voice = providerConfig.Voice
			}
			if providerConfig.Model != "" {
				options.Model = providerConfig.Model
			}
			if providerConfig.Format != "" {
				options.Format = providerConfig.Format
			}
			if providerConfig.Speed > 0 {
				options.Speed = providerConfig.Speed
			}
			if providerConfig.APIKey != "" {
				options.APIKey = providerConfig.APIKey
			}

			// ElevenLabs-specific
			if providerConfig.Stability > 0 {
				options.Stability = providerConfig.Stability
			}
			if providerConfig.SimilarityBoost > 0 {
				options.SimilarityBoost = providerConfig.SimilarityBoost
			}
			if providerConfig.Style > 0 {
				options.Style = providerConfig.Style
			}
			// UseSpeakerBoost defaults to true, only set if explicitly configured
			if providerConfig.UseSpeakerBoost != nil {
				options.UseSpeakerBoost = *providerConfig.UseSpeakerBoost
			}

			// Polly-specific
			if providerConfig.Region != "" {
				options.Region = providerConfig.Region
			}
			if providerConfig.Engine != "" {
				options.Engine = providerConfig.Engine
			}
			if providerConfig.SampleRate != "" {
				options.SampleRate = providerConfig.SampleRate
			}
		}
	}

	return options
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
		fmt.Println("No configuration file found.")
		fmt.Println("\nSearched locations:")
		fmt.Println("  - .claude/config.json (project)")
		fmt.Println("  - ~/.claude/config.json (global)")
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
		configPath = filepath.Join(homeDir, ".claude", "config.json")
	} else {
		configPath = ".claude/config.json"
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

	fmt.Printf("✅ Created configuration: %s\n", configPath)
	fmt.Println("\nEdit the file to configure your preferred voice providers.")
	fmt.Println("Use ${ENV_VAR} syntax for sensitive values like API keys.")

	return nil
}

// Helper function for loading voice config in handlers
func loadVoiceConfig(c *cli.Command) *voice.VoiceConfigFile {
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

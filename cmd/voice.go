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
	if !c.Bool("force") && voice.IsMuted() {
		log.Debug().Msg("voice synthesis is globally muted, skipping")
		return nil
	}

	personaConfig := loadUnifiedConfig(c, "")
	fileConfig := personaConfig.ToVoiceConfigFile()
	personaInput := personaConfig.ToVoiceInput()
	if personaConfig != nil && personaConfig.Voice != nil {
		log.Debug().
			Str("persona", personaConfig.Name).
			Str("voice_provider", personaConfig.Voice.Provider).
			Int("voice_speaker", personaConfig.Voice.Speaker).
			Float64("voice_volume", personaConfig.Voice.Volume).
			Float64("voice_speed", personaConfig.Voice.Speed).
			Msg("Applied persona voice config")
	} else {
		log.Debug().Msg("No unified voice config found, using defaults")
	}

	// Only pass cliProvider when explicitly set by the user.
	// The flag has a non-empty default ("aivisspeech"), so c.String("provider") is always non-empty;
	// using IsSet() prevents the default from silently overriding persona/file config.
	cliProvider := ""
	if c.IsSet("provider") {
		cliProvider = c.String("provider")
	}
	baseOpts := voice.Resolve(personaInput, fileConfig, cliProvider)

	// CLI speaker overrides everything (highest priority)
	if cliSpeaker := int(c.Int("speaker")); cliSpeaker > 0 {
		baseOpts.VoicevoxSpeaker = cliSpeaker
		baseOpts.AivisSpeechSpeaker = cliSpeaker
	}

	voiceConfig := baseOpts.ToConfig(voice.DefaultConfig())
	voiceConfig.ReadingMode = c.String("mode")

	// Create voice manager
	manager := voice.NewVoiceManager(voiceConfig)

	provider := baseOpts.Provider

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

	// Overlay output-only CLI flags on top of resolved options.
	options := buildVoiceOptions(c, baseOpts)

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

// buildVoiceOptions overlays output-destination CLI flags onto already-resolved options.
// All provider/speed/volume settings are expected to be set by voice.Resolve() already.
func buildVoiceOptions(c *cli.Command, base voice.VoiceOptions) voice.VoiceOptions {
	output := c.String("output")
	toStdout := output == "-"

	base.OutputPath = output
	base.PlayAudio = output == "" // play if no output path specified
	base.ToStdout = toStdout

	if toStdout {
		base.OutputPath = ""
	}

	// --voice CLI flag overrides resolved voice name
	if v := c.String("voice"); v != "" {
		base.Voice = v
	}

	return base
}

// Voice config management handlers

func handleVoiceConfigShow(ctx context.Context, c *cli.Command) error {
	config := loadUnifiedConfig(c, "")
	if config == nil {
		fmt.Println("No configuration file found.")
		fmt.Println("\nSearched locations:")
		fmt.Println("  - .agents/ccpersona.json (project)")
		fmt.Println("  - ~/.agents/ccpersona.json (global)")
		fmt.Println("\nRun 'ccpersona voice config init' to create one, or 'ccpersona config migrate' to migrate legacy files.")
		return nil
	}

	masked := maskUnifiedConfig(config)
	output, err := json.MarshalIndent(masked, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format config: %w", err)
	}

	fmt.Println("Current ccpersona configuration (secrets masked):")
	fmt.Println(string(output))

	return nil
}

func handleVoiceConfigValidate(ctx context.Context, c *cli.Command) error {
	config := loadUnifiedConfig(c, "")
	if config == nil {
		fmt.Println("No ccpersona configuration file found.")
		return nil
	}

	if err := persona.ValidateConfig(config); err == nil {
		fmt.Println("✅ Configuration is valid.")
		return nil
	} else {
		fmt.Println("❌ Configuration has errors:")
		fmt.Printf("  - %s\n", err)
		return fmt.Errorf("configuration validation failed")
	}
}

func handleVoiceConfigInit(ctx context.Context, c *cli.Command) error {
	var configPath string

	if c.Bool("global") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = persona.ConfigPath(homeDir)
	} else {
		configPath = persona.ConfigPath(".")
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

	exampleConfig := &persona.Config{
		Name: "default",
		Voice: &persona.VoiceConfig{
			Provider: "openai",
			APIKey:   "${OPENAI_API_KEY}",
			Model:    "tts-1",
			Voice:    "nova",
			Format:   "mp3",
			Speed:    1.0,
			Volume:   1.0,
		},
		Engines: map[string]voice.EngineUserConfig{
			"irodori": {
				BaseURL: "http://127.0.0.1:8088/v1",
				Health:  "openai",
			},
		},
	}
	example, err := json.MarshalIndent(exampleConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}
	example = append(example, '\n')

	// Write with secure permissions
	if err := os.WriteFile(configPath, example, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("✅ Created configuration: %s\n", configPath)
	fmt.Println("\nEdit the file to configure your preferred voice providers.")
	fmt.Println("Use ${ENV_VAR} syntax for sensitive values like API keys.")

	return nil
}

// Voice mute gate handlers

func handleVoiceMute(ctx context.Context, c *cli.Command) error {
	status, err := voice.Mute(c.String("reason"))
	if err != nil {
		return fmt.Errorf("failed to enable mute: %w", err)
	}
	path, _ := voice.MutePath()
	fmt.Printf("🔇 Voice synthesis muted globally.\n")
	fmt.Printf("   Marker : %s\n", path)
	fmt.Printf("   At     : %s\n", status.MutedAt.Format("2006-01-02 15:04:05 MST"))
	if status.Reason != "" {
		fmt.Printf("   Reason : %s\n", status.Reason)
	}
	fmt.Println("\nRun 'ccpersona voice unmute' to re-enable, or pass --force to bypass for one call.")
	return nil
}

func handleVoiceUnmute(ctx context.Context, c *cli.Command) error {
	wasMuted := voice.IsMuted()
	if err := voice.Unmute(); err != nil {
		return fmt.Errorf("failed to disable mute: %w", err)
	}
	if wasMuted {
		fmt.Println("🔊 Voice synthesis unmuted.")
	} else {
		fmt.Println("🔊 Voice synthesis was not muted; nothing to do.")
	}
	return nil
}

func handleVoiceStatus(ctx context.Context, c *cli.Command) error {
	status, err := voice.LoadMuteStatus()
	if err != nil {
		return fmt.Errorf("failed to read mute status: %w", err)
	}
	if status == nil {
		fmt.Println("🔊 Voice synthesis: ACTIVE (not muted)")
		return nil
	}

	path, _ := voice.MutePath()
	fmt.Println("🔇 Voice synthesis: MUTED")
	fmt.Printf("   Marker : %s\n", path)
	if !status.MutedAt.IsZero() {
		fmt.Printf("   Since  : %s\n", status.MutedAt.Local().Format("2006-01-02 15:04:05 MST"))
	}
	if status.Reason != "" {
		fmt.Printf("   Reason : %s\n", status.Reason)
	}
	return nil
}

func loadUnifiedConfig(c *cli.Command, platform string) *persona.Config {
	if configPath := c.String("config"); configPath != "" {
		config, err := persona.LoadConfigFromPath(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ccpersona: failed to load %s; using built-in defaults: %v\n", configPath, err)
			return nil
		}
		return config
	}

	config, err := persona.LoadConfigWithFallbackForPlatform(platform)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load ccpersona config")
		return nil
	}
	return config
}

// Helper function for loading voice config in handlers.
func loadVoiceConfig(c *cli.Command) *voice.ConfigFile {
	return loadUnifiedConfig(c, "").ToVoiceConfigFile()
}

func maskUnifiedConfig(config *persona.Config) *persona.Config {
	if config == nil {
		return nil
	}
	masked := *config
	if config.Voice != nil {
		voiceCfg := *config.Voice
		if voiceCfg.APIKey != "" {
			voiceCfg.APIKey = fmt.Sprintf("[set, %d chars]", len(voiceCfg.APIKey))
		}
		masked.Voice = &voiceCfg
	}
	return &masked
}

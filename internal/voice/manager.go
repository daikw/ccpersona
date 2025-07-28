package voice

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/rs/zerolog/log"
)

// VoiceManager manages both local and cloud TTS providers
type VoiceManager struct {
	config    *Config
	providers map[string]Provider
	engine    *VoiceEngine // For backward compatibility with local engines
}

// NewVoiceManager creates a new voice manager with all available providers
func NewVoiceManager(config *Config) *VoiceManager {
	vm := &VoiceManager{
		config:    config,
		providers: make(map[string]Provider),
		engine:    NewVoiceEngine(config),
	}

	// Initialize cloud providers from environment variables
	vm.initializeProviders()

	return vm
}

// initializeProviders initializes all available providers
func (vm *VoiceManager) initializeProviders() {
	// OpenAI provider
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config := &OpenAIConfig{
			APIKey: apiKey,
			Model:  getEnvWithDefault("OPENAI_TTS_MODEL", "tts-1"),
			Voice:  getEnvWithDefault("OPENAI_TTS_VOICE", "alloy"),
			Speed:  1.0,
		}
		vm.providers["openai"] = NewOpenAIProvider(config)
		log.Debug().Msg("OpenAI TTS provider initialized")
	}

	// ElevenLabs provider
	if apiKey := os.Getenv("ELEVENLABS_API_KEY"); apiKey != "" {
		config := &ElevenLabsConfig{
			APIKey: apiKey,
			Voice:  getEnvWithDefault("ELEVENLABS_VOICE", "21m00Tcm4TlvDq8ikWAM"),
			Settings: &ElevenLabsVoiceSettings{
				Stability:       0.5,
				SimilarityBoost: 0.5,
				Style:           0.0,
				UseSpeakerBoost: true,
			},
		}
		vm.providers["elevenlabs"] = NewElevenLabsProvider(config)
		log.Debug().Msg("ElevenLabs TTS provider initialized")
	}

	// Amazon Polly provider
	if region := os.Getenv("AWS_REGION"); region != "" {
		config := &PollyConfig{
			Region:          region,
			Voice:           getEnvWithDefault("AWS_POLLY_VOICE", "Joanna"),
			Engine:          getEnvWithDefault("AWS_POLLY_ENGINE", "neural"),
			Language:        getEnvWithDefault("AWS_POLLY_LANGUAGE", "en-US"),
			OutputFormat:    getEnvWithDefault("AWS_POLLY_FORMAT", "mp3"),
			SampleRate:      getEnvWithDefault("AWS_POLLY_SAMPLE_RATE", "22050"),
			AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		}
		vm.providers["polly"] = NewPollyProvider(config)
		log.Debug().Msg("Amazon Polly TTS provider initialized")
	}

	// Google Cloud TTS provider
	if projectID := os.Getenv("GOOGLE_CLOUD_PROJECT"); projectID != "" {
		config := &GCPConfig{
			ProjectID:       projectID,
			Voice:           getEnvWithDefault("GCP_TTS_VOICE", "en-US-Wavenet-D"),
			LanguageCode:    getEnvWithDefault("GCP_TTS_LANGUAGE", "en-US"),
			SpeakingRate:    1.0,
			Pitch:           0.0,
			VolumeGainDb:    0.0,
			AudioEncoding:   getEnvWithDefault("GCP_TTS_FORMAT", "MP3"),
			CredentialsFile: os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
		}
		vm.providers["gcp"] = NewGCPProvider(config)
		log.Debug().Msg("Google Cloud TTS provider initialized")
	}
}

// ListProviders returns all available providers
func (vm *VoiceManager) ListProviders(ctx context.Context) []string {
	var providers []string

	// Add local engines if available
	voicevox, aivisspeech := vm.engine.CheckEngines()
	if voicevox {
		providers = append(providers, "voicevox")
	}
	if aivisspeech {
		providers = append(providers, "aivisspeech")
	}

	// Add cloud providers
	for name, provider := range vm.providers {
		if provider.IsAvailable(ctx) {
			providers = append(providers, name)
		}
	}

	return providers
}

// ListVoices returns all available voices from all providers
func (vm *VoiceManager) ListVoices(ctx context.Context, providerName string) ([]Voice, error) {
	if providerName == "" {
		// Return voices from all providers
		var allVoices []Voice

		for name, provider := range vm.providers {
			if provider.IsAvailable(ctx) {
				voices, err := provider.ListVoices(ctx)
				if err != nil {
					log.Warn().Err(err).Str("provider", name).Msg("Failed to get voices from provider")
					continue
				}
				allVoices = append(allVoices, voices...)
			}
		}

		// Add local engine "voices" (speakers)
		voicevox, aivisspeech := vm.engine.CheckEngines()
		if voicevox {
			allVoices = append(allVoices, Voice{
				ID:          "voicevox-3",
				Name:        "ずんだもん (VOICEVOX)",
				Language:    "ja-JP",
				Gender:      "female",
				Description: "VOICEVOX ずんだもん voice",
				Provider:    "voicevox",
				Tags:        []string{"local", "japanese"},
			})
		}
		if aivisspeech {
			allVoices = append(allVoices, Voice{
				ID:          "aivisspeech-1512153248",
				Name:        "Default (AivisSpeech)",
				Language:    "ja-JP",
				Gender:      "female",
				Description: "AivisSpeech default voice",
				Provider:    "aivisspeech",
				Tags:        []string{"local", "japanese"},
			})
		}

		return allVoices, nil
	}

	// Return voices from specific provider
	if provider, exists := vm.providers[providerName]; exists {
		if !provider.IsAvailable(ctx) {
			return nil, fmt.Errorf("provider %s is not available", providerName)
		}
		return provider.ListVoices(ctx)
	}

	// Handle local engines
	if providerName == "voicevox" || providerName == "aivisspeech" {
		voicevox, aivisspeech := vm.engine.CheckEngines()

		if providerName == "voicevox" && voicevox {
			return []Voice{{
				ID:          "voicevox-3",
				Name:        "ずんだもん",
				Language:    "ja-JP",
				Gender:      "female",
				Description: "VOICEVOX ずんだもん voice",
				Provider:    "voicevox",
				Tags:        []string{"local", "japanese"},
			}}, nil
		}

		if providerName == "aivisspeech" && aivisspeech {
			return []Voice{{
				ID:          "aivisspeech-1512153248",
				Name:        "Default",
				Language:    "ja-JP",
				Gender:      "female",
				Description: "AivisSpeech default voice",
				Provider:    "aivisspeech",
				Tags:        []string{"local", "japanese"},
			}}, nil
		}
	}

	return nil, fmt.Errorf("provider %s not found or not available", providerName)
}

// Synthesize generates audio using the specified provider
func (vm *VoiceManager) Synthesize(ctx context.Context, text, providerName string, options *SynthesizeOptions) (string, error) {
	// Handle local engines (backward compatibility)
	if providerName == "" || providerName == "voicevox" || providerName == "aivisspeech" {
		return vm.engine.Synthesize(text)
	}

	// Handle cloud providers
	provider, exists := vm.providers[providerName]
	if !exists {
		return "", fmt.Errorf("provider %s not found", providerName)
	}

	if !provider.IsAvailable(ctx) {
		return "", fmt.Errorf("provider %s is not available", providerName)
	}

	// Synthesize audio
	audioStream, format, err := provider.Synthesize(ctx, text, options)
	if err != nil {
		return "", fmt.Errorf("synthesis failed: %w", err)
	}
	defer audioStream.Close()

	// Save to temporary file
	extension := getExtensionForFormat(format)
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("voice_%s_*.%s", providerName, extension))
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	// Copy audio data to file
	_, err = io.Copy(tmpFile, audioStream)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to save audio: %w", err)
	}

	log.Debug().
		Str("provider", providerName).
		Str("file", tmpFile.Name()).
		Str("format", string(format)).
		Msg("Audio synthesis completed")

	return tmpFile.Name(), nil
}

// Play plays an audio file using the appropriate player
func (vm *VoiceManager) Play(audioFile string) error {
	return vm.engine.Play(audioFile)
}

// SynthesizeAndPlay combines synthesis and playback
func (vm *VoiceManager) SynthesizeAndPlay(ctx context.Context, text, providerName string, options *SynthesizeOptions) error {
	audioFile, err := vm.Synthesize(ctx, text, providerName, options)
	if err != nil {
		return err
	}

	return vm.Play(audioFile)
}

// WriteToOutput writes audio to a specific output (file or stdout)
func (vm *VoiceManager) WriteToOutput(ctx context.Context, text, providerName, outputPath string, options *SynthesizeOptions) error {
	// Handle local engines
	if providerName == "" || providerName == "voicevox" || providerName == "aivisspeech" {
		audioFile, err := vm.engine.Synthesize(text)
		if err != nil {
			return err
		}
		defer os.Remove(audioFile)

		return copyFile(audioFile, outputPath)
	}

	// Handle cloud providers
	provider, exists := vm.providers[providerName]
	if !exists {
		return fmt.Errorf("provider %s not found", providerName)
	}

	if !provider.IsAvailable(ctx) {
		return fmt.Errorf("provider %s is not available", providerName)
	}

	// Synthesize audio
	audioStream, _, err := provider.Synthesize(ctx, text, options)
	if err != nil {
		return fmt.Errorf("synthesis failed: %w", err)
	}
	defer audioStream.Close()

	// Write to output
	if outputPath == "-" || outputPath == "stdout" {
		_, err = io.Copy(os.Stdout, audioStream)
	} else {
		outFile, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, audioStream)
		if err != nil {
			return fmt.Errorf("failed to copy audio data: %w", err)
		}
	}

	return nil
}

// getEnvWithDefault returns environment variable value or default
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getExtensionForFormat returns file extension for audio format
func getExtensionForFormat(format AudioFormat) string {
	switch format {
	case FormatMP3:
		return "mp3"
	case FormatWAV:
		return "wav"
	case FormatOGG:
		return "ogg"
	case FormatFLAC:
		return "flac"
	case FormatAAC:
		return "aac"
	default:
		return "mp3"
	}
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	if dst == "-" || dst == "stdout" {
		srcFile, err := os.Open(src)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		_, err = io.Copy(os.Stdout, srcFile)
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

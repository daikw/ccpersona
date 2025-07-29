package voice

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/daikw/ccpersona/internal/voice/provider"
	"github.com/rs/zerolog/log"
)

// VoiceManager manages both local engines and cloud providers
type VoiceManager struct {
	config          *Config
	legacyEngine    *VoiceEngine
	providerFactory provider.Factory
}

// NewVoiceManager creates a new voice manager
func NewVoiceManager(config *Config) *VoiceManager {
	return &VoiceManager{
		config:          config,
		legacyEngine:    NewVoiceEngine(config),
		providerFactory: provider.NewFactory(),
	}
}

// VoiceOptions contains options for voice synthesis
type VoiceOptions struct {
	Provider string
	Voice    string
	Speed    float64
	Format   string
	Quality  string
	APIKey   string
	Model    string

	// Output options
	OutputPath string
	PlayAudio  bool
	ToStdout   bool
}

// ListVoices lists available voices for all providers
func (vm *VoiceManager) ListVoices(ctx context.Context, providerName string) ([]provider.Voice, error) {
	if providerName == "" {
		// List all providers
		var allVoices []provider.Voice

		// Add local engines as "voices"
		voicevoxAvail, aivisAvail := vm.legacyEngine.CheckEngines()
		if voicevoxAvail {
			allVoices = append(allVoices, provider.Voice{
				ID:          "voicevox",
				Name:        "VOICEVOX Local Engine",
				Language:    "ja",
				Description: "Local VOICEVOX engine (multiple speakers available)",
			})
		}
		if aivisAvail {
			allVoices = append(allVoices, provider.Voice{
				ID:          "aivisspeech",
				Name:        "AivisSpeech Local Engine",
				Language:    "ja",
				Description: "Local AivisSpeech engine (multiple speakers available)",
			})
		}

		// Add cloud providers
		for _, provName := range vm.providerFactory.ListProviders() {
			prov, err := vm.providerFactory.GetProviderWithDefaults(provName)
			if err != nil {
				log.Debug().Err(err).Str("provider", provName).Msg("Failed to create provider")
				continue
			}

			if prov.IsAvailable(ctx) {
				voices, err := prov.ListVoices(ctx)
				if err != nil {
					log.Debug().Err(err).Str("provider", provName).Msg("Failed to list voices")
					continue
				}
				allVoices = append(allVoices, voices...)
			}
		}

		return allVoices, nil
	}

	// Handle local engines
	if providerName == "voicevox" || providerName == "aivisspeech" {
		voicevoxAvail, aivisAvail := vm.legacyEngine.CheckEngines()
		var voices []provider.Voice

		if providerName == "voicevox" && voicevoxAvail {
			voices = append(voices, provider.Voice{
				ID:          "voicevox",
				Name:        "VOICEVOX Local Engine",
				Language:    "ja",
				Description: "Local VOICEVOX engine with configurable speakers",
			})
		}

		if providerName == "aivisspeech" && aivisAvail {
			voices = append(voices, provider.Voice{
				ID:          "aivisspeech",
				Name:        "AivisSpeech Local Engine",
				Language:    "ja",
				Description: "Local AivisSpeech engine with configurable speakers",
			})
		}

		return voices, nil
	}

	// Handle cloud providers
	prov, err := vm.providerFactory.GetProviderWithDefaults(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", providerName, err)
	}

	if !prov.IsAvailable(ctx) {
		return nil, fmt.Errorf("provider %s is not available", providerName)
	}

	return prov.ListVoices(ctx)
}

// Synthesize generates audio using the specified provider
func (vm *VoiceManager) Synthesize(ctx context.Context, text string, options VoiceOptions) (string, error) {
	if text == "" {
		return "", fmt.Errorf("text cannot be empty")
	}

	// Handle local engines (legacy)
	if options.Provider == "" || options.Provider == "voicevox" || options.Provider == "aivisspeech" {
		return vm.synthesizeLocal(text, options)
	}

	// Handle cloud providers
	return vm.synthesizeCloud(ctx, text, options)
}

// synthesizeLocal uses the legacy local engines
func (vm *VoiceManager) synthesizeLocal(text string, options VoiceOptions) (string, error) {
	if options.Provider != "" {
		// Override engine priority if provider specified
		if options.Provider == "voicevox" {
			vm.config.EnginePriority = EngineVoicevox
		} else if options.Provider == "aivisspeech" {
			vm.config.EnginePriority = EngineAivisSpeech
		}
	}

	return vm.legacyEngine.Synthesize(text)
}

// synthesizeCloud uses cloud providers
func (vm *VoiceManager) synthesizeCloud(ctx context.Context, text string, options VoiceOptions) (string, error) {
	// Create provider config
	config := make(map[string]interface{})

	if options.APIKey != "" {
		config["api_key"] = options.APIKey
	}

	// Create provider
	prov, err := vm.providerFactory.CreateProvider(options.Provider, config)
	if err != nil {
		return "", fmt.Errorf("failed to create provider: %w", err)
	}

	if !prov.IsAvailable(ctx) {
		return "", fmt.Errorf("provider %s is not available", options.Provider)
	}

	// Set synthesis options
	synthOptions := provider.SynthesizeOptions{
		Voice:   options.Voice,
		Speed:   options.Speed,
		Format:  options.Format,
		Quality: options.Quality,
		Model:   options.Model,
	}

	// Synthesize
	audioStream, err := prov.Synthesize(ctx, text, synthOptions)
	if err != nil {
		return "", fmt.Errorf("synthesis failed: %w", err)
	}
	defer audioStream.Close()

	// Handle output
	if options.ToStdout {
		// Stream directly to stdout
		_, err := io.Copy(os.Stdout, audioStream)
		return "", err
	}

	// Save to file
	var outputPath string
	if options.OutputPath != "" {
		outputPath = options.OutputPath
	} else {
		// Create temporary file
		tmpFile, err := os.CreateTemp("", "voice_*."+getFileExtension(options.Format))
		if err != nil {
			return "", fmt.Errorf("failed to create temp file: %w", err)
		}
		outputPath = tmpFile.Name()
		tmpFile.Close()
	}

	// Write audio data to file
	file, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, audioStream)
	if err != nil {
		os.Remove(outputPath)
		return "", fmt.Errorf("failed to write audio data: %w", err)
	}

	log.Debug().Str("path", outputPath).Msg("Audio saved")
	return outputPath, nil
}

// PlayAudio plays an audio file using the legacy engine's player
func (vm *VoiceManager) PlayAudio(audioPath string) error {
	return vm.legacyEngine.Play(audioPath)
}

// getFileExtension returns the file extension for a format
func getFileExtension(format string) string {
	switch format {
	case "mp3":
		return "mp3"
	case "wav":
		return "wav"
	case "ogg":
		return "ogg"
	case "flac":
		return "flac"
	case "aac":
		return "aac"
	default:
		return "mp3"
	}
}

// CleanupTempFiles removes temporary audio files older than specified duration
func (vm *VoiceManager) CleanupTempFiles(maxAge time.Duration) error {
	tempDir := os.TempDir()

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return err
	}

	now := time.Now()

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !isVoiceFile(name) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > maxAge {
			path := filepath.Join(tempDir, name)
			if err := os.Remove(path); err != nil {
				log.Debug().Err(err).Str("path", path).Msg("Failed to remove temp file")
			} else {
				log.Debug().Str("path", path).Msg("Removed old temp file")
			}
		}
	}

	return nil
}

// isVoiceFile checks if a filename looks like a voice synthesis temp file
func isVoiceFile(filename string) bool {
	return (len(filename) > 6 && filename[:6] == "voice_") ||
		(len(filename) > 8 && filename[:8] == "ccpersona_voice_")
}

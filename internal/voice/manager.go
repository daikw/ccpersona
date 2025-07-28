package voice

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/rs/zerolog/log"
)

// VoiceManager handles voice synthesis using either legacy engines or cloud providers
type VoiceManager struct {
	config   *Config
	provider Provider
	legacy   *VoiceEngine // For backward compatibility
}

// NewVoiceManager creates a new voice manager
func NewVoiceManager(config *Config) (*VoiceManager, error) {
	manager := &VoiceManager{
		config: config,
	}

	// If cloud provider is configured, use the new provider system
	if config.Provider != nil && config.Provider.Provider != ProviderLocal {
		factory := NewProviderFactory(config.Provider)
		provider, err := factory.CreateProvider(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to create provider: %w", err)
		}
		manager.provider = provider
	} else {
		// Fall back to legacy system for local engines or if no provider config
		if config.Provider != nil && config.Provider.Local != nil {
			// Use new local provider
			provider, err := NewLocalProvider(config.Provider)
			if err != nil {
				return nil, fmt.Errorf("failed to create local provider: %w", err)
			}
			manager.provider = provider
		} else {
			// Use legacy voice engine for backward compatibility
			manager.legacy = NewVoiceEngine(config)
		}
	}

	return manager, nil
}

// IsAvailable checks if the voice synthesis system is available
func (vm *VoiceManager) IsAvailable(ctx context.Context) bool {
	if vm.provider != nil {
		return vm.provider.IsAvailable(ctx)
	}
	
	if vm.legacy != nil {
		_, _ = vm.legacy.CheckEngines()
		engine, err := vm.legacy.SelectEngine()
		return err == nil && engine != ""
	}
	
	return false
}

// Synthesize converts text to speech
func (vm *VoiceManager) Synthesize(ctx context.Context, text string, options *SynthesizeOptions) (io.ReadCloser, error) {
	if vm.provider != nil {
		return vm.provider.Synthesize(ctx, text, options)
	}
	
	if vm.legacy != nil {
		// Use legacy engine
		audioFile, err := vm.legacy.Synthesize(text)
		if err != nil {
			return nil, err
		}
		
		// Return file as ReadCloser
		file, err := os.Open(audioFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open audio file: %w", err)
		}
		
		// Wrap with cleanup
		return &fileReadCloser{
			file:     file,
			filePath: audioFile,
		}, nil
	}
	
	return nil, fmt.Errorf("no voice synthesis system available")
}

// SynthesizeAndPlay converts text to speech and plays it
func (vm *VoiceManager) SynthesizeAndPlay(ctx context.Context, text string, options *SynthesizeOptions) error {
	audioStream, err := vm.Synthesize(ctx, text, options)
	if err != nil {
		return err
	}
	
	// If using legacy system, it returns a file path - play it directly
	if vm.legacy != nil {
		if frc, ok := audioStream.(*fileReadCloser); ok {
			defer audioStream.Close()
			return vm.playAudioFile(frc.filePath)
		}
	}
	
	// For cloud providers, save to temp file first
	tempFile, err := vm.saveToTempFile(audioStream)
	if err != nil {
		return err
	}
	
	defer func() {
		// Clean up temp file after a delay
		go func() {
			time.Sleep(10 * time.Second)
			os.Remove(tempFile)
		}()
	}()
	
	return vm.playAudioFile(tempFile)
}

// SynthesizeToFile converts text to speech and saves to a file
func (vm *VoiceManager) SynthesizeToFile(ctx context.Context, text string, outputFile string, options *SynthesizeOptions) error {
	audioStream, err := vm.Synthesize(ctx, text, options)
	if err != nil {
		return err
	}
	defer audioStream.Close()
	
	// Create output file
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()
	
	// Copy audio data to file
	_, err = io.Copy(file, audioStream)
	if err != nil {
		return fmt.Errorf("failed to save audio data: %w", err)
	}
	
	log.Info().Str("file", outputFile).Msg("Audio saved to file")
	return nil
}

// SynthesizeToStdout converts text to speech and streams to stdout
func (vm *VoiceManager) SynthesizeToStdout(ctx context.Context, text string, options *SynthesizeOptions) error {
	audioStream, err := vm.Synthesize(ctx, text, options)
	if err != nil {
		return err
	}
	defer audioStream.Close()
	
	// Copy audio data to stdout
	_, err = io.Copy(os.Stdout, audioStream)
	if err != nil {
		return fmt.Errorf("failed to stream audio to stdout: %w", err)
	}
	
	return nil
}

// GetProviderName returns the name of the current provider
func (vm *VoiceManager) GetProviderName() string {
	if vm.provider != nil {
		return vm.provider.Name()
	}
	
	if vm.legacy != nil {
		return "Legacy Local Engine"
	}
	
	return "Unknown"
}

// GetSupportedFormats returns supported audio formats
func (vm *VoiceManager) GetSupportedFormats() []AudioFormat {
	if vm.provider != nil {
		return vm.provider.GetSupportedFormats()
	}
	
	// Legacy system only supports WAV
	return []AudioFormat{AudioFormatWAV}
}

// GetDefaultFormat returns the default audio format
func (vm *VoiceManager) GetDefaultFormat() AudioFormat {
	if vm.provider != nil {
		return vm.provider.GetDefaultFormat()
	}
	
	return AudioFormatWAV
}

// saveToTempFile saves an audio stream to a temporary file
func (vm *VoiceManager) saveToTempFile(audioStream io.ReadCloser) (string, error) {
	defer audioStream.Close()
	
	// Determine file extension based on provider format
	ext := ".wav" // Default
	if vm.provider != nil {
		switch vm.provider.GetDefaultFormat() {
		case AudioFormatMP3:
			ext = ".mp3"
		case AudioFormatOGG:
			ext = ".ogg"
		case AudioFormatFLAC:
			ext = ".flac"
		case AudioFormatAAC:
			ext = ".aac"
		}
	}
	
	tmpFile, err := os.CreateTemp("", "ccpersona_tts_*"+ext)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()
	
	_, err = io.Copy(tmpFile, audioStream)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to save audio data: %w", err)
	}
	
	log.Debug().Str("file", tmpFile.Name()).Msg("Audio saved to temp file")
	return tmpFile.Name(), nil
}

// playAudioFile plays an audio file using the appropriate system player
func (vm *VoiceManager) playAudioFile(audioFile string) error {
	// Use the same audio playback logic as the legacy system
	var cmd *exec.Cmd
	
	switch {
	case isCommandAvailable("afplay"):
		// macOS
		cmd = exec.Command("afplay", audioFile)
	case isCommandAvailable("aplay"):
		// Linux with ALSA
		cmd = exec.Command("aplay", audioFile)
	case isCommandAvailable("paplay"):
		// Linux with PulseAudio
		cmd = exec.Command("paplay", audioFile)
	case isCommandAvailable("ffplay"):
		// Cross-platform with ffmpeg
		cmd = exec.Command("ffplay", "-nodisp", "-autoexit", audioFile)
	default:
		return fmt.Errorf("no audio player found")
	}
	
	// Start playing
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to play audio: %w", err)
	}
	
	// Wait for playback to complete
	if err := cmd.Wait(); err != nil {
		log.Warn().Err(err).Msg("Audio playback may have been interrupted")
	}
	
	return nil
}

// fileReadCloser wraps a file with cleanup functionality
type fileReadCloser struct {
	file     *os.File
	filePath string
}

// Read implements io.Reader
func (frc *fileReadCloser) Read(p []byte) (n int, err error) {
	return frc.file.Read(p)
}

// Close implements io.Closer and cleans up the file
func (frc *fileReadCloser) Close() error {
	err := frc.file.Close()
	// Don't remove the file immediately - let the caller handle cleanup
	return err
}

// isCommandAvailable checks if a command is available (copied from engine.go)
func isCommandAvailable(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// UpdateLegacyVoiceEngine updates the existing VoiceEngine to use the new provider system
func UpdateLegacyVoiceEngine(config *Config) *VoiceManager {
	// Create provider config from legacy config
	if config.Provider == nil {
		providerConfig := &ProviderConfig{
			Provider: ProviderLocal,
			Local: &LocalConfig{
				Engine:             config.EnginePriority,
				VoicevoxSpeaker:    config.VoicevoxSpeaker,
				AivisSpeechSpeaker: config.AivisSpeechSpeaker,
				VoicevoxURL:        VoicevoxURL,
				AivisSpeechURL:     AivisSpeechURL,
			},
		}
		config.Provider = providerConfig
	}
	
	manager, err := NewVoiceManager(config)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create voice manager, falling back to legacy")
		return &VoiceManager{
			config: config,
			legacy: NewVoiceEngine(config),
		}
	}
	
	return manager
}
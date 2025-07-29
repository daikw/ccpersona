package provider

import (
	"context"
	"io"
)

// Provider defines the interface for TTS providers
type Provider interface {
	// Name returns the provider name
	Name() string

	// ListVoices returns available voices for this provider
	ListVoices(ctx context.Context) ([]Voice, error)

	// Synthesize generates audio from text and returns an audio stream
	Synthesize(ctx context.Context, text string, options SynthesizeOptions) (io.ReadCloser, error)

	// IsAvailable checks if the provider is available (can be used)
	IsAvailable(ctx context.Context) bool
}

// Voice represents a voice option
type Voice struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Language    string `json:"language"`
	Gender      string `json:"gender,omitempty"`
	Description string `json:"description,omitempty"`
}

// SynthesizeOptions contains options for text synthesis
type SynthesizeOptions struct {
	Voice    string  `json:"voice"`
	Speed    float64 `json:"speed,omitempty"`    // Speed multiplier (0.25-4.0)
	Format   string  `json:"format,omitempty"`   // Output format (mp3, wav, etc.)
	Quality  string  `json:"quality,omitempty"`  // Quality setting (hd, standard)
	Language string  `json:"language,omitempty"` // Language code
	Model    string  `json:"model,omitempty"`    // Model to use (tts-1, tts-1-hd)
}

// Config contains provider-specific configuration
type Config struct {
	Provider string                 `json:"provider"`
	Settings map[string]interface{} `json:"settings"`
}

// Factory creates provider instances
type Factory interface {
	CreateProvider(providerName string, config map[string]interface{}) (Provider, error)
	GetProviderWithDefaults(providerName string) (Provider, error)
	ListProviders() []string
}

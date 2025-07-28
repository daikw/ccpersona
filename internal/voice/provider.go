package voice

import (
	"context"
	"io"
)

// Provider represents a TTS provider interface
type Provider interface {
	// Name returns the provider name
	Name() string

	// Synthesize generates audio from text and returns an audio stream
	Synthesize(ctx context.Context, text string, options *SynthesizeOptions) (io.ReadCloser, AudioFormat, error)

	// ListVoices returns available voices for this provider
	ListVoices(ctx context.Context) ([]Voice, error)

	// IsAvailable checks if the provider is available/configured
	IsAvailable(ctx context.Context) bool
}

// SynthesizeOptions contains options for text-to-speech synthesis
type SynthesizeOptions struct {
	Voice    string  `json:"voice"`    // Voice ID or name
	Speed    float64 `json:"speed"`    // Speech speed (0.25-4.0)
	Format   string  `json:"format"`   // Audio format preference
	Language string  `json:"language"` // Language code (optional)
}

// Voice represents a voice option from a provider
type Voice struct {
	ID          string   `json:"id"`          // Unique identifier
	Name        string   `json:"name"`        // Display name
	Language    string   `json:"language"`    // Language code
	Gender      string   `json:"gender"`      // Gender (male/female/neutral)
	Description string   `json:"description"` // Description
	Provider    string   `json:"provider"`    // Provider name
	Tags        []string `json:"tags"`        // Additional tags
}

// AudioFormat represents supported audio formats
type AudioFormat string

const (
	FormatMP3  AudioFormat = "mp3"
	FormatWAV  AudioFormat = "wav"
	FormatOGG  AudioFormat = "ogg"
	FormatFLAC AudioFormat = "flac"
	FormatAAC  AudioFormat = "aac"
)

// ProviderConfig contains configuration for cloud providers
type ProviderConfig struct {
	// OpenAI configuration
	OpenAI *OpenAIConfig `json:"openai,omitempty"`

	// ElevenLabs configuration
	ElevenLabs *ElevenLabsConfig `json:"elevenlabs,omitempty"`

	// Amazon Polly configuration
	Polly *PollyConfig `json:"polly,omitempty"`

	// Google Cloud TTS configuration
	GCP *GCPConfig `json:"gcp,omitempty"`
}

// OpenAIConfig contains OpenAI Audio API configuration
type OpenAIConfig struct {
	APIKey string  `json:"api_key"`
	Model  string  `json:"model"` // tts-1 or tts-1-hd
	Voice  string  `json:"voice"` // alloy, echo, fable, onyx, nova, shimmer
	Speed  float64 `json:"speed"` // 0.25 to 4.0
}

// ElevenLabsConfig contains ElevenLabs TTS configuration
type ElevenLabsConfig struct {
	APIKey   string                   `json:"api_key"`
	Voice    string                   `json:"voice"`    // Voice ID
	Model    string                   `json:"model"`    // Model ID (optional)
	Settings *ElevenLabsVoiceSettings `json:"settings"` // Voice settings
}

// ElevenLabsVoiceSettings contains ElevenLabs voice settings
type ElevenLabsVoiceSettings struct {
	Stability       float64 `json:"stability"`        // 0.0 to 1.0
	SimilarityBoost float64 `json:"similarity_boost"` // 0.0 to 1.0
	Style           float64 `json:"style"`            // 0.0 to 1.0 (v2 models only)
	UseSpeakerBoost bool    `json:"use_speaker_boost"`
}

// PollyConfig contains Amazon Polly configuration
type PollyConfig struct {
	Region       string `json:"region"`        // AWS region
	Voice        string `json:"voice"`         // Voice ID
	Engine       string `json:"engine"`        // standard, neural, generative
	Language     string `json:"language"`      // Language code
	OutputFormat string `json:"output_format"` // mp3, ogg_vorbis, pcm
	SampleRate   string `json:"sample_rate"`   // Sample rate

	// AWS credentials (optional - can use environment/IAM)
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
}

// GCPConfig contains Google Cloud TTS configuration
type GCPConfig struct {
	ProjectID     string  `json:"project_id"`
	Voice         string  `json:"voice"`          // Voice name
	LanguageCode  string  `json:"language_code"`  // Language code
	SpeakingRate  float64 `json:"speaking_rate"`  // 0.25 to 4.0
	Pitch         float64 `json:"pitch"`          // -20.0 to 20.0
	VolumeGainDb  float64 `json:"volume_gain_db"` // -96.0 to 16.0
	AudioEncoding string  `json:"audio_encoding"` // MP3, WAV, OGG_OPUS

	// Service account key file path (optional - can use ADC)
	CredentialsFile string `json:"credentials_file,omitempty"`
}

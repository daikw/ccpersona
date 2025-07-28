package voice

import (
	"context"
	"io"
)

// Provider defines the interface for TTS providers
type Provider interface {
	// Name returns the provider name
	Name() string
	
	// IsAvailable checks if the provider is available (has credentials, connection, etc.)
	IsAvailable(ctx context.Context) bool
	
	// Synthesize converts text to speech and returns an audio stream
	// The returned ReadCloser should be closed by the caller
	Synthesize(ctx context.Context, text string, options *SynthesizeOptions) (io.ReadCloser, error)
	
	// GetSupportedFormats returns supported audio formats
	GetSupportedFormats() []AudioFormat
	
	// GetDefaultFormat returns the default audio format for this provider
	GetDefaultFormat() AudioFormat
}

// SynthesizeOptions contains options for text-to-speech synthesis
type SynthesizeOptions struct {
	// Voice settings
	Voice    string  `json:"voice,omitempty"`     // Voice ID or name
	Speed    float32 `json:"speed,omitempty"`     // Speech speed (0.25-4.0)
	Pitch    float32 `json:"pitch,omitempty"`     // Pitch adjustment
	Volume   float32 `json:"volume,omitempty"`    // Volume (0.0-1.0)
	
	// Audio format
	Format     AudioFormat `json:"format,omitempty"`      // Output format
	SampleRate int         `json:"sample_rate,omitempty"` // Sample rate in Hz
	
	// Streaming options
	StreamToStdout bool   `json:"stream_to_stdout,omitempty"` // Stream directly to stdout
	OutputFile     string `json:"output_file,omitempty"`      // Save to file instead of temp
}

// AudioFormat represents supported audio formats
type AudioFormat string

const (
	AudioFormatMP3  AudioFormat = "mp3"
	AudioFormatWAV  AudioFormat = "wav"
	AudioFormatOGG  AudioFormat = "ogg"
	AudioFormatFLAC AudioFormat = "flac"
	AudioFormatAAC  AudioFormat = "aac"
	AudioFormatPCM  AudioFormat = "pcm"
)

// CloudProvider represents cloud-based TTS providers
type CloudProvider string

const (
	ProviderOpenAI     CloudProvider = "openai"
	ProviderElevenLabs CloudProvider = "elevenlabs"
	ProviderPolly      CloudProvider = "polly"
	ProviderGCP        CloudProvider = "gcp"
	ProviderLocal      CloudProvider = "local" // For existing VOICEVOX/AivisSpeech
)

// ProviderConfig contains provider-specific configuration
type ProviderConfig struct {
	// Provider selection
	Provider CloudProvider `json:"provider"`
	
	// Authentication
	APIKey    string `json:"api_key,omitempty"`    // For OpenAI, ElevenLabs
	Region    string `json:"region,omitempty"`     // For AWS Polly
	ProjectID string `json:"project_id,omitempty"` // For GCP
	
	// Provider-specific settings
	OpenAI     *OpenAIConfig     `json:"openai,omitempty"`
	ElevenLabs *ElevenLabsConfig `json:"elevenlabs,omitempty"`
	Polly      *PollyConfig      `json:"polly,omitempty"`
	GCP        *GCPConfig        `json:"gcp,omitempty"`
	Local      *LocalConfig      `json:"local,omitempty"`
}

// OpenAIConfig contains OpenAI-specific settings
type OpenAIConfig struct {
	Model  string  `json:"model"`            // tts-1, tts-1-hd
	Voice  string  `json:"voice"`            // alloy, echo, fable, onyx, nova, shimmer
	Speed  float32 `json:"speed,omitempty"`  // 0.25-4.0
	Format string  `json:"format,omitempty"` // mp3, opus, aac, flac
}

// ElevenLabsConfig contains ElevenLabs-specific settings
type ElevenLabsConfig struct {
	VoiceID           string            `json:"voice_id"`
	Model             string            `json:"model,omitempty"`              // eleven_monolingual_v1, etc.
	VoiceSettings     *VoiceSettings    `json:"voice_settings,omitempty"`
	PronunciationDict map[string]string `json:"pronunciation_dict,omitempty"`
}

// VoiceSettings for ElevenLabs
type VoiceSettings struct {
	Stability       float32 `json:"stability"`        // 0.0-1.0
	SimilarityBoost float32 `json:"similarity_boost"` // 0.0-1.0
	Style           float32 `json:"style,omitempty"`  // 0.0-1.0
	UseSpeakerBoost bool    `json:"use_speaker_boost,omitempty"`
}

// PollyConfig contains Amazon Polly-specific settings
type PollyConfig struct {
	VoiceID      string            `json:"voice_id"`
	Engine       string            `json:"engine,omitempty"`        // standard, neural
	LanguageCode string            `json:"language_code,omitempty"` // en-US, ja-JP, etc.
	OutputFormat string            `json:"output_format,omitempty"` // mp3, ogg_vorbis, pcm
	SampleRate   string            `json:"sample_rate,omitempty"`   // 8000, 16000, 22050, 24000
	SpeechMarks  []string          `json:"speech_marks,omitempty"`  // sentence, ssml, viseme, word
	Lexicons     []string          `json:"lexicons,omitempty"`      // Pronunciation lexicon names
	SSML         bool              `json:"ssml,omitempty"`          // Input is SSML
	MaxRetries   int               `json:"max_retries,omitempty"`   // AWS SDK retry count
	Credentials  *AWSCredentials   `json:"credentials,omitempty"`   // AWS credentials
}

// AWSCredentials for Polly authentication
type AWSCredentials struct {
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	SessionToken    string `json:"session_token,omitempty"`
	Profile         string `json:"profile,omitempty"`         // AWS profile name
	UseInstanceRole bool   `json:"use_instance_role,omitempty"` // Use EC2 instance role
}

// GCPConfig contains Google Cloud TTS-specific settings
type GCPConfig struct {
	VoiceName        string                 `json:"voice_name"`          // e.g., "en-US-Wavenet-D"
	LanguageCode     string                 `json:"language_code"`       // e.g., "en-US"
	SsmlGender       string                 `json:"ssml_gender"`         // MALE, FEMALE, NEUTRAL
	AudioEncoding    string                 `json:"audio_encoding"`      // MP3, LINEAR16, OGG_OPUS
	SampleRateHertz  int32                  `json:"sample_rate_hertz,omitempty"`
	SpeakingRate     float64                `json:"speaking_rate,omitempty"`     // 0.25-4.0
	Pitch            float64                `json:"pitch,omitempty"`             // -20.0 to 20.0
	VolumeGainDb     float64                `json:"volume_gain_db,omitempty"`    // -96.0 to 16.0
	EffectsProfileId []string               `json:"effects_profile_id,omitempty"`
	CustomVoice      *CustomVoiceConfig     `json:"custom_voice,omitempty"`
	Credentials      *GCPCredentials        `json:"credentials,omitempty"`
}

// CustomVoiceConfig for GCP custom voices
type CustomVoiceConfig struct {
	Model      string `json:"model"`
	ReportedUsage string `json:"reported_usage,omitempty"`
}

// GCPCredentials for GCP authentication
type GCPCredentials struct {
	ServiceAccountKey string `json:"service_account_key,omitempty"` // Path to JSON key file
	UseADC            bool   `json:"use_adc,omitempty"`             // Use Application Default Credentials
}

// LocalConfig contains settings for local engines (VOICEVOX, AivisSpeech)
type LocalConfig struct {
	Engine             string `json:"engine"`                        // voicevox, aivisspeech
	VoicevoxSpeaker    int    `json:"voicevox_speaker,omitempty"`    // VOICEVOX speaker ID
	AivisSpeechSpeaker int64  `json:"aivisspeech_speaker,omitempty"` // AivisSpeech speaker ID
	VoicevoxURL        string `json:"voicevox_url,omitempty"`        // Custom VOICEVOX URL
	AivisSpeechURL     string `json:"aivisspeech_url,omitempty"`     // Custom AivisSpeech URL
}

// ProviderFactory creates providers based on configuration
type ProviderFactory struct {
	config *ProviderConfig
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory(config *ProviderConfig) *ProviderFactory {
	return &ProviderFactory{config: config}
}

// CreateProvider creates a provider instance
func (f *ProviderFactory) CreateProvider(ctx context.Context) (Provider, error) {
	switch f.config.Provider {
	case ProviderOpenAI:
		return NewOpenAIProvider(f.config)
	case ProviderElevenLabs:
		return NewElevenLabsProvider(f.config)
	case ProviderPolly:
		return NewPollyProvider(f.config)
	case ProviderGCP:
		return NewGCPProvider(f.config)
	case ProviderLocal:
		return NewLocalProvider(f.config)
	default:
		// Default to local provider for backward compatibility
		return NewLocalProvider(f.config)
	}
}

// GetAvailableProviders returns a list of available providers
func GetAvailableProviders(ctx context.Context, configs []*ProviderConfig) []Provider {
	var providers []Provider
	
	for _, config := range configs {
		factory := NewProviderFactory(config)
		provider, err := factory.CreateProvider(ctx)
		if err != nil {
			continue
		}
		
		if provider.IsAvailable(ctx) {
			providers = append(providers, provider)
		}
	}
	
	return providers
}
package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/rs/zerolog/log"
)

// GCPProvider implements the Provider interface for Google Cloud Text-to-Speech
type GCPProvider struct {
	client    *texttospeech.Client
	projectID string
	region    string
	voice     string
	language  string
	engine    string // standard, wavenet, neural2, studio
}

// GCPProviderOption is a functional option for configuring GCPProvider
type GCPProviderOption func(*GCPProvider)

// WithGCPProjectID sets the Google Cloud project ID
func WithGCPProjectID(projectID string) GCPProviderOption {
	return func(p *GCPProvider) {
		p.projectID = projectID
	}
}

// WithGCPRegion sets the region for the GCP client
func WithGCPRegion(region string) GCPProviderOption {
	return func(p *GCPProvider) {
		p.region = region
	}
}

// WithGCPVoice sets the default voice
func WithGCPVoice(voice string) GCPProviderOption {
	return func(p *GCPProvider) {
		p.voice = voice
	}
}

// WithGCPLanguage sets the default language code
func WithGCPLanguage(language string) GCPProviderOption {
	return func(p *GCPProvider) {
		p.language = language
	}
}

// WithGCPEngine sets the voice engine type
func WithGCPEngine(engine string) GCPProviderOption {
	return func(p *GCPProvider) {
		p.engine = engine
	}
}

// NewGCPProvider creates a new Google Cloud TTS provider
// Authentication is handled via GOOGLE_APPLICATION_CREDENTIALS environment variable
// or Application Default Credentials (ADC)
func NewGCPProvider(ctx context.Context, opts ...GCPProviderOption) (*GCPProvider, error) {
	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP TTS client: %w", err)
	}

	p := &GCPProvider{
		client:   client,
		voice:    "ja-JP-Neural2-B", // Default Japanese male voice
		language: "ja-JP",
		engine:   "neural2",
	}

	for _, opt := range opts {
		opt(p)
	}

	return p, nil
}

// Name returns the provider name
func (p *GCPProvider) Name() string {
	return "gcp"
}

// ListVoices returns available voices from Google Cloud TTS
func (p *GCPProvider) ListVoices(ctx context.Context) ([]Voice, error) {
	req := &texttospeechpb.ListVoicesRequest{}

	resp, err := p.client.ListVoices(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list GCP voices: %w", err)
	}

	var voices []Voice
	for _, v := range resp.Voices {
		for _, langCode := range v.LanguageCodes {
			gender := "unknown"
			switch v.SsmlGender {
			case texttospeechpb.SsmlVoiceGender_MALE:
				gender = "male"
			case texttospeechpb.SsmlVoiceGender_FEMALE:
				gender = "female"
			case texttospeechpb.SsmlVoiceGender_NEUTRAL:
				gender = "neutral"
			}

			// Determine engine type from voice name
			engineType := p.detectEngineType(v.Name)

			voices = append(voices, Voice{
				ID:          v.Name,
				Name:        v.Name,
				Language:    langCode,
				Gender:      gender,
				Description: fmt.Sprintf("%s voice (%s)", engineType, strings.Join(v.LanguageCodes, ", ")),
			})
		}
	}

	log.Debug().Int("count", len(voices)).Msg("Listed GCP TTS voices")
	return voices, nil
}

// detectEngineType determines the engine type from voice name
func (p *GCPProvider) detectEngineType(voiceName string) string {
	name := strings.ToLower(voiceName)
	switch {
	case strings.Contains(name, "wavenet"):
		return "WaveNet"
	case strings.Contains(name, "neural2"):
		return "Neural2"
	case strings.Contains(name, "studio"):
		return "Studio"
	case strings.Contains(name, "polyglot"):
		return "Polyglot"
	case strings.Contains(name, "news"):
		return "News"
	case strings.Contains(name, "casual"):
		return "Casual"
	default:
		return "Standard"
	}
}

// Synthesize generates audio from text using Google Cloud TTS
func (p *GCPProvider) Synthesize(ctx context.Context, text string, options SynthesizeOptions) (io.ReadCloser, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Determine voice to use
	voice := p.voice
	if options.Voice != "" {
		voice = options.Voice
	}

	// Determine language from voice name or options
	language := p.language
	if options.Language != "" {
		language = options.Language
	} else if voice != "" {
		// Extract language from voice name (e.g., ja-JP-Neural2-B -> ja-JP)
		parts := strings.Split(voice, "-")
		if len(parts) >= 2 {
			language = parts[0] + "-" + parts[1]
		}
	}

	// Build synthesis input (detect SSML)
	var input *texttospeechpb.SynthesisInput
	if isSSML(text) {
		input = &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Ssml{
				Ssml: text,
			},
		}
		log.Debug().Msg("Using SSML input for GCP TTS")
	} else {
		input = &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{
				Text: text,
			},
		}
	}

	// Build voice selection
	voiceSelection := &texttospeechpb.VoiceSelectionParams{
		LanguageCode: language,
		Name:         voice,
	}

	// Build audio config
	audioConfig := &texttospeechpb.AudioConfig{
		AudioEncoding: p.getAudioEncoding(options.Format),
		SpeakingRate:  p.getSpeakingRate(options.Speed),
		SampleRateHertz: p.getSampleRate(options.SampleRate),
	}

	log.Debug().
		Str("voice", voice).
		Str("language", language).
		Str("format", options.Format).
		Float64("speed", options.Speed).
		Msg("Making GCP TTS synthesis request")

	// Make the API call
	resp, err := p.client.SynthesizeSpeech(ctx, &texttospeechpb.SynthesizeSpeechRequest{
		Input:       input,
		Voice:       voiceSelection,
		AudioConfig: audioConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to synthesize speech: %w", err)
	}

	log.Debug().
		Int("audio_bytes", len(resp.AudioContent)).
		Msg("GCP TTS synthesis successful")

	return io.NopCloser(bytes.NewReader(resp.AudioContent)), nil
}

// isSSML checks if the text contains SSML tags
func isSSML(text string) bool {
	trimmed := strings.TrimSpace(text)
	return strings.HasPrefix(trimmed, "<speak") ||
		strings.Contains(trimmed, "<prosody") ||
		strings.Contains(trimmed, "<break") ||
		strings.Contains(trimmed, "<emphasis")
}

// getAudioEncoding converts format string to GCP audio encoding
func (p *GCPProvider) getAudioEncoding(format string) texttospeechpb.AudioEncoding {
	switch strings.ToLower(format) {
	case "mp3":
		return texttospeechpb.AudioEncoding_MP3
	case "wav", "linear16":
		return texttospeechpb.AudioEncoding_LINEAR16
	case "ogg", "ogg_opus":
		return texttospeechpb.AudioEncoding_OGG_OPUS
	case "mulaw":
		return texttospeechpb.AudioEncoding_MULAW
	case "alaw":
		return texttospeechpb.AudioEncoding_ALAW
	default:
		return texttospeechpb.AudioEncoding_MP3
	}
}

// getSpeakingRate converts speed to GCP speaking rate (0.25 to 4.0)
func (p *GCPProvider) getSpeakingRate(speed float64) float64 {
	if speed <= 0 {
		return 1.0
	}
	if speed < 0.25 {
		return 0.25
	}
	if speed > 4.0 {
		return 4.0
	}
	return speed
}

// getSampleRate returns the sample rate in Hz
func (p *GCPProvider) getSampleRate(sampleRate string) int32 {
	switch sampleRate {
	case "8000":
		return 8000
	case "16000":
		return 16000
	case "22050":
		return 22050
	case "24000":
		return 24000
	case "32000":
		return 32000
	case "44100":
		return 44100
	case "48000":
		return 48000
	default:
		return 0 // Use default
	}
}

// IsAvailable checks if the GCP TTS service is available
func (p *GCPProvider) IsAvailable(ctx context.Context) bool {
	// Try to list voices as a health check
	_, err := p.client.ListVoices(ctx, &texttospeechpb.ListVoicesRequest{})
	return err == nil
}

// Close closes the GCP client
func (p *GCPProvider) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

// GCPProviderFromConfig creates a GCPProvider from configuration map
func GCPProviderFromConfig(config map[string]interface{}) (*GCPProvider, error) {
	ctx := context.Background()

	var opts []GCPProviderOption

	if projectID, ok := config["project_id"].(string); ok && projectID != "" {
		opts = append(opts, WithGCPProjectID(projectID))
	}

	if region, ok := config["region"].(string); ok && region != "" {
		opts = append(opts, WithGCPRegion(region))
	}

	if voice, ok := config["voice"].(string); ok && voice != "" {
		opts = append(opts, WithGCPVoice(voice))
	}

	if language, ok := config["language"].(string); ok && language != "" {
		opts = append(opts, WithGCPLanguage(language))
	}

	if engine, ok := config["engine"].(string); ok && engine != "" {
		opts = append(opts, WithGCPEngine(engine))
	}

	return NewGCPProvider(ctx, opts...)
}

// ListJapaneseVoices returns only Japanese voices for convenience
func (p *GCPProvider) ListJapaneseVoices(ctx context.Context) ([]Voice, error) {
	req := &texttospeechpb.ListVoicesRequest{
		LanguageCode: "ja-JP",
	}

	resp, err := p.client.ListVoices(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list Japanese voices: %w", err)
	}

	var voices []Voice
	for _, v := range resp.Voices {
		gender := "unknown"
		switch v.SsmlGender {
		case texttospeechpb.SsmlVoiceGender_MALE:
			gender = "male"
		case texttospeechpb.SsmlVoiceGender_FEMALE:
			gender = "female"
		case texttospeechpb.SsmlVoiceGender_NEUTRAL:
			gender = "neutral"
		}

		engineType := p.detectEngineType(v.Name)

		voices = append(voices, Voice{
			ID:          v.Name,
			Name:        v.Name,
			Language:    "ja-JP",
			Gender:      gender,
			Description: fmt.Sprintf("%s voice", engineType),
		})
	}

	return voices, nil
}

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	ElevenLabsBaseURL        = "https://api.elevenlabs.io/v1"
	ElevenLabsTTSEndpoint    = "/text-to-speech"
	ElevenLabsVoicesEndpoint = "/voices"
)

// ElevenLabsProvider implements the Provider interface for ElevenLabs TTS API v1
type ElevenLabsProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewElevenLabsProvider creates a new ElevenLabs TTS provider
func NewElevenLabsProvider(apiKey string) *ElevenLabsProvider {
	return &ElevenLabsProvider{
		apiKey:  apiKey,
		baseURL: ElevenLabsBaseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // ElevenLabs can be slower than OpenAI
		},
	}
}

// Name returns the provider name
func (p *ElevenLabsProvider) Name() string {
	return "elevenlabs"
}

// ElevenLabsVoice represents a voice from ElevenLabs API
type ElevenLabsVoice struct {
	VoiceID         string            `json:"voice_id"`
	Name            string            `json:"name"`
	Samples         []Sample          `json:"samples"`
	Category        string            `json:"category"`
	Finetuning      Finetuning        `json:"fine_tuning"`
	Labels          map[string]string `json:"labels"`
	Description     string            `json:"description"`
	PreviewURL      string            `json:"preview_url"`
	AvailableForTts bool              `json:"available_for_tts"`
	Settings        VoiceSettings     `json:"settings"`
}

type Sample struct {
	SampleID  string `json:"sample_id"`
	FileName  string `json:"file_name"`
	MimeType  string `json:"mime_type"`
	SizeBytes int    `json:"size_bytes"`
	Hash      string `json:"hash"`
}

type Finetuning struct {
	Language                          string        `json:"language"`
	IsAllowedToFineTune               bool          `json:"is_allowed_to_fine_tune"`
	FinetuningRequested               bool          `json:"finetuning_requested"`
	FinetuningState                   string        `json:"finetuning_state"`
	VerificationAttempts              []interface{} `json:"verification_attempts"`
	VerificationFailures              []string      `json:"verification_failures"`
	VerificationAttemptCountThisMonth int           `json:"verification_attempts_count"`
	StorageSize                       int           `json:"storage_size_bytes"`
}

type VoiceSettings struct {
	Stability       float64 `json:"stability"`
	SimilarityBoost float64 `json:"similarity_boost"`
	Style           float64 `json:"style,omitempty"`
	UseSpeakerBoost bool    `json:"use_speaker_boost,omitempty"`
}

// ElevenLabsVoicesResponse represents the response from voices API
type ElevenLabsVoicesResponse struct {
	Voices []ElevenLabsVoice `json:"voices"`
}

// ListVoices returns available ElevenLabs voices
func (p *ElevenLabsProvider) ListVoices(ctx context.Context) ([]Voice, error) {
	// Create HTTP request to list voices
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+ElevenLabsVoicesEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create voices request: %w", err)
	}

	// Set headers
	req.Header.Set("xi-api-key", p.apiKey)

	log.Debug().
		Str("endpoint", p.baseURL+ElevenLabsVoicesEndpoint).
		Msg("Making ElevenLabs voices request")

	// Make request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make voices request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ElevenLabs voices API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var voicesResp ElevenLabsVoicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&voicesResp); err != nil {
		return nil, fmt.Errorf("failed to decode voices response: %w", err)
	}

	// Convert to standard Voice format
	voices := make([]Voice, 0, len(voicesResp.Voices))
	for _, v := range voicesResp.Voices {
		if !v.AvailableForTts {
			continue // Skip voices not available for TTS
		}

		// Determine gender from labels or description
		gender := ""
		if genderLabel, ok := v.Labels["gender"]; ok {
			gender = genderLabel
		}

		// Use language from fine-tuning info or default to multilingual
		language := "multilingual"
		if v.Finetuning.Language != "" {
			language = v.Finetuning.Language
		}

		voice := Voice{
			ID:          v.VoiceID,
			Name:        v.Name,
			Language:    language,
			Gender:      gender,
			Description: v.Description,
		}
		voices = append(voices, voice)
	}

	log.Debug().
		Int("voice_count", len(voices)).
		Msg("ElevenLabs voices retrieved successfully")

	return voices, nil
}

// ElevenLabsTTSRequest represents the request body for TTS synthesis
type ElevenLabsTTSRequest struct {
	Text          string        `json:"text"`
	ModelID       string        `json:"model_id,omitempty"`
	VoiceSettings VoiceSettings `json:"voice_settings,omitempty"`
	OutputFormat  string        `json:"output_format,omitempty"`
}

// Synthesize generates audio from text using ElevenLabs TTS API
func (p *ElevenLabsProvider) Synthesize(ctx context.Context, text string, options SynthesizeOptions) (io.ReadCloser, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Set defaults
	voice := options.Voice
	if voice == "" {
		voice = "21m00Tcm4TlvDq8ikWAM" // Rachel (default pre-built voice)
	}

	model := options.Model
	if model == "" {
		model = "eleven_multilingual_v2" // Default model
	}

	format := options.Format
	if format == "" {
		format = "mp3"
	}

	// Convert format to ElevenLabs format
	outputFormat := convertToElevenLabsFormat(format)

	// Voice settings from options with sensible defaults
	voiceSettings := VoiceSettings{
		Stability:       0.5,
		SimilarityBoost: 0.5,
		Style:           0.0,
		UseSpeakerBoost: true,
	}

	// Override with provided options
	if options.Stability > 0 {
		voiceSettings.Stability = options.Stability
	}
	if options.SimilarityBoost > 0 {
		voiceSettings.SimilarityBoost = options.SimilarityBoost
	}
	if options.Style >= 0 {
		voiceSettings.Style = options.Style
	}
	voiceSettings.UseSpeakerBoost = options.UseSpeakerBoost

	// Prepare request payload
	requestBody := ElevenLabsTTSRequest{
		Text:          text,
		ModelID:       model,
		VoiceSettings: voiceSettings,
		OutputFormat:  outputFormat,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	endpoint := fmt.Sprintf("%s%s/%s", p.baseURL, ElevenLabsTTSEndpoint, voice)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", p.apiKey)

	log.Debug().
		Str("endpoint", endpoint).
		Str("voice", voice).
		Str("model", model).
		Str("format", outputFormat).
		Msg("Making ElevenLabs TTS request")

	// Make request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		// Parse error response if possible
		var errorResp ElevenLabsError
		if json.Unmarshal(body, &errorResp) == nil && errorResp.Detail != nil {
			return nil, fmt.Errorf("ElevenLabs API error: %s", errorResp.String())
		}

		return nil, fmt.Errorf("ElevenLabs API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	log.Debug().
		Int("status", resp.StatusCode).
		Str("content_type", resp.Header.Get("Content-Type")).
		Msg("ElevenLabs TTS request successful")

	return resp.Body, nil
}

// IsAvailable checks if ElevenLabs provider is available
func (p *ElevenLabsProvider) IsAvailable(ctx context.Context) bool {
	if p.apiKey == "" {
		return false
	}

	// Test by listing voices (minimal request)
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+ElevenLabsVoicesEndpoint, nil)
	if err != nil {
		return false
	}

	req.Header.Set("xi-api-key", p.apiKey)

	// Use a shorter timeout for availability check
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// ElevenLabsProviderFromConfig creates an ElevenLabs provider from configuration
func ElevenLabsProviderFromConfig(config map[string]interface{}) (*ElevenLabsProvider, error) {
	apiKey, ok := config["api_key"].(string)
	if !ok || apiKey == "" {
		return nil, fmt.Errorf("api_key is required for ElevenLabs provider")
	}

	provider := NewElevenLabsProvider(apiKey)

	// Optional base URL override
	if baseURL, ok := config["base_url"].(string); ok && baseURL != "" {
		provider.baseURL = strings.TrimSuffix(baseURL, "/")
	}

	return provider, nil
}

// convertToElevenLabsFormat converts common format names to ElevenLabs format names
func convertToElevenLabsFormat(format string) string {
	format = strings.ToLower(format)
	switch format {
	case "mp3", "mpeg":
		return "mp3_44100_128"
	case "wav", "wave":
		return "pcm_44100"
	case "ogg":
		return "ulaw_8000"
	case "flac":
		return "pcm_44100" // ElevenLabs doesn't support FLAC directly, use PCM
	case "aac":
		return "mp3_44100_128" // ElevenLabs doesn't support AAC directly, use MP3
	default:
		return "mp3_44100_128" // Default to MP3
	}
}

// ElevenLabsError represents an error from ElevenLabs API
type ElevenLabsError struct {
	Detail interface{} `json:"detail"`
}

func (e ElevenLabsError) String() string {
	switch detail := e.Detail.(type) {
	case string:
		return fmt.Sprintf("ElevenLabs API Error: %s", detail)
	case map[string]interface{}:
		if msg, ok := detail["message"].(string); ok {
			return fmt.Sprintf("ElevenLabs API Error: %s", msg)
		}
		return fmt.Sprintf("ElevenLabs API Error: %v", detail)
	case []interface{}:
		if len(detail) > 0 {
			if firstError, ok := detail[0].(map[string]interface{}); ok {
				if msg, ok := firstError["msg"].(string); ok {
					return fmt.Sprintf("ElevenLabs API Error: %s", msg)
				}
			}
		}
		return fmt.Sprintf("ElevenLabs API Error: %v", detail)
	default:
		return fmt.Sprintf("ElevenLabs API Error: %v", detail)
	}
}

// GetPrebuiltVoices returns the well-known pre-built ElevenLabs voices
func GetPrebuiltVoices() []Voice {
	return []Voice{
		{ID: "21m00Tcm4TlvDq8ikWAM", Name: "Rachel", Language: "en", Gender: "female", Description: "American female voice"},
		{ID: "AZnzlk1XvdvUeBnXmlld", Name: "Domi", Language: "en", Gender: "female", Description: "American female voice"},
		{ID: "EXAVITQu4vr4xnSDxMaL", Name: "Bella", Language: "en", Gender: "female", Description: "American female voice"},
		{ID: "ErXwobaYiN019PkySvjV", Name: "Antoni", Language: "en", Gender: "male", Description: "American male voice"},
		{ID: "MF3mGyEYCl7XYWbV9V6O", Name: "Elli", Language: "en", Gender: "female", Description: "American female voice"},
		{ID: "TxGEqnHWrfWFTfGW9XjX", Name: "Josh", Language: "en", Gender: "male", Description: "American male voice"},
		{ID: "VR6AewLTigWG4xSOukaG", Name: "Arnold", Language: "en", Gender: "male", Description: "American male voice"},
		{ID: "pNInz6obpgDQGcFmaJgB", Name: "Adam", Language: "en", Gender: "male", Description: "American male voice"},
		{ID: "yoZ06aMxZJJ28mfd3POQ", Name: "Sam", Language: "en", Gender: "male", Description: "American male voice"},
	}
}

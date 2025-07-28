package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// ElevenLabsProvider implements the Provider interface for ElevenLabs TTS
type ElevenLabsProvider struct {
	config     *ElevenLabsConfig
	httpClient *http.Client
}

// NewElevenLabsProvider creates a new ElevenLabs TTS provider
func NewElevenLabsProvider(config *ElevenLabsConfig) *ElevenLabsProvider {
	if config == nil {
		config = &ElevenLabsConfig{
			Voice: "21m00Tcm4TlvDq8ikWAM", // Default Rachel voice
			Settings: &ElevenLabsVoiceSettings{
				Stability:       0.5,
				SimilarityBoost: 0.5,
				Style:           0.0,
				UseSpeakerBoost: true,
			},
		}
	}

	return &ElevenLabsProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the provider name
func (p *ElevenLabsProvider) Name() string {
	return "elevenlabs"
}

// Synthesize generates audio from text using ElevenLabs API
func (p *ElevenLabsProvider) Synthesize(ctx context.Context, text string, options *SynthesizeOptions) (io.ReadCloser, AudioFormat, error) {
	voiceID := p.config.Voice
	if options != nil && options.Voice != "" {
		voiceID = options.Voice
	}

	// Prepare request payload
	payload := map[string]interface{}{
		"text":           text,
		"voice_settings": p.config.Settings,
	}

	// Add model if specified
	if p.config.Model != "" {
		payload["model_id"] = p.config.Model
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s", voiceID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "audio/mpeg")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", p.config.APIKey)

	// Make request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("API request failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	log.Debug().
		Str("provider", "elevenlabs").
		Str("voice", voiceID).
		Msg("ElevenLabs TTS synthesis successful")

	// ElevenLabs returns MP3 by default
	return resp.Body, FormatMP3, nil
}

// ListVoices returns available ElevenLabs voices
func (p *ElevenLabsProvider) ListVoices(ctx context.Context) ([]Voice, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.elevenlabs.io/v1/voices", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("xi-api-key", p.config.APIKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Voices []struct {
			VoiceID     string            `json:"voice_id"`
			Name        string            `json:"name"`
			Category    string            `json:"category"`
			Description string            `json:"description"`
			Labels      map[string]string `json:"labels"`
		} `json:"voices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var voices []Voice
	for _, v := range response.Voices {
		gender := "neutral"
		if g, ok := v.Labels["gender"]; ok {
			gender = g
		}

		language := "en-US"
		if lang, ok := v.Labels["language"]; ok {
			language = lang
		}

		var tags []string
		for k, v := range v.Labels {
			tags = append(tags, fmt.Sprintf("%s:%s", k, v))
		}

		voices = append(voices, Voice{
			ID:          v.VoiceID,
			Name:        v.Name,
			Language:    language,
			Gender:      gender,
			Description: v.Description,
			Provider:    "elevenlabs",
			Tags:        tags,
		})
	}

	return voices, nil
}

// IsAvailable checks if ElevenLabs provider is configured and available
func (p *ElevenLabsProvider) IsAvailable(ctx context.Context) bool {
	if p.config.APIKey == "" {
		return false
	}

	// Test API availability
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(testCtx, "GET", "https://api.elevenlabs.io/v1/user", nil)
	if err != nil {
		return false
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("xi-api-key", p.config.APIKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("ElevenLabs API not available")
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

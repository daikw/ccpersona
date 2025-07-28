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

// OpenAIProvider implements the Provider interface for OpenAI Audio API
type OpenAIProvider struct {
	config     *OpenAIConfig
	httpClient *http.Client
}

// NewOpenAIProvider creates a new OpenAI TTS provider
func NewOpenAIProvider(config *OpenAIConfig) *OpenAIProvider {
	if config == nil {
		config = &OpenAIConfig{
			Model: "tts-1",
			Voice: "alloy",
			Speed: 1.0,
		}
	}

	return &OpenAIProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// Synthesize generates audio from text using OpenAI Audio API
func (p *OpenAIProvider) Synthesize(ctx context.Context, text string, options *SynthesizeOptions) (io.ReadCloser, AudioFormat, error) {
	// Prepare request payload
	payload := map[string]interface{}{
		"model": p.config.Model,
		"input": text,
		"voice": p.config.Voice,
		"speed": p.config.Speed,
	}

	// Override with options if provided
	if options != nil {
		if options.Voice != "" {
			payload["voice"] = options.Voice
		}
		if options.Speed > 0 {
			payload["speed"] = options.Speed
		}
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/audio/speech", bytes.NewReader(jsonData))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

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
		Str("provider", "openai").
		Str("model", p.config.Model).
		Str("voice", payload["voice"].(string)).
		Msg("OpenAI TTS synthesis successful")

	// OpenAI returns MP3 by default
	return resp.Body, FormatMP3, nil
}

// ListVoices returns available OpenAI voices
func (p *OpenAIProvider) ListVoices(ctx context.Context) ([]Voice, error) {
	// OpenAI has fixed set of voices
	voices := []Voice{
		{
			ID:          "alloy",
			Name:        "Alloy",
			Language:    "en-US",
			Gender:      "neutral",
			Description: "Neutral, balanced voice",
			Provider:    "openai",
			Tags:        []string{"neutral", "clear"},
		},
		{
			ID:          "echo",
			Name:        "Echo",
			Language:    "en-US",
			Gender:      "male",
			Description: "Male voice with good clarity",
			Provider:    "openai",
			Tags:        []string{"male", "clear"},
		},
		{
			ID:          "fable",
			Name:        "Fable",
			Language:    "en-US",
			Gender:      "male",
			Description: "Expressive male voice",
			Provider:    "openai",
			Tags:        []string{"male", "expressive"},
		},
		{
			ID:          "onyx",
			Name:        "Onyx",
			Language:    "en-US",
			Gender:      "male",
			Description: "Deep male voice",
			Provider:    "openai",
			Tags:        []string{"male", "deep"},
		},
		{
			ID:          "nova",
			Name:        "Nova",
			Language:    "en-US",
			Gender:      "female",
			Description: "Female voice with warmth",
			Provider:    "openai",
			Tags:        []string{"female", "warm"},
		},
		{
			ID:          "shimmer",
			Name:        "Shimmer",
			Language:    "en-US",
			Gender:      "female",
			Description: "Bright female voice",
			Provider:    "openai",
			Tags:        []string{"female", "bright"},
		},
	}

	return voices, nil
}

// IsAvailable checks if OpenAI provider is configured and available
func (p *OpenAIProvider) IsAvailable(ctx context.Context) bool {
	if p.config.APIKey == "" {
		return false
	}

	// Test API availability with a minimal request
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(testCtx, "GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return false
	}

	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("OpenAI API not available")
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

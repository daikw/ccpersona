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
	OpenAIBaseURL     = "https://api.openai.com/v1"
	OpenAITTSEndpoint = "/audio/speech"
)

// OpenAIProvider implements the Provider interface for OpenAI Audio API
type OpenAIProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewOpenAIProvider creates a new OpenAI TTS provider
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: OpenAIBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// ListVoices returns available OpenAI voices
func (p *OpenAIProvider) ListVoices(ctx context.Context) ([]Voice, error) {
	// OpenAI Audio API voices (as of 2024)
	voices := []Voice{
		{ID: "alloy", Name: "Alloy", Language: "en", Gender: "neutral", Description: "Balanced, clear voice"},
		{ID: "echo", Name: "Echo", Language: "en", Gender: "male", Description: "Deep, resonant voice"},
		{ID: "fable", Name: "Fable", Language: "en", Gender: "neutral", Description: "Expressive, storytelling voice"},
		{ID: "onyx", Name: "Onyx", Language: "en", Gender: "male", Description: "Strong, authoritative voice"},
		{ID: "nova", Name: "Nova", Language: "en", Gender: "female", Description: "Bright, energetic voice"},
		{ID: "shimmer", Name: "Shimmer", Language: "en", Gender: "female", Description: "Warm, friendly voice"},
	}
	return voices, nil
}

// Synthesize generates audio from text using OpenAI Audio API
func (p *OpenAIProvider) Synthesize(ctx context.Context, text string, options SynthesizeOptions) (io.ReadCloser, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Set defaults
	voice := options.Voice
	if voice == "" {
		voice = "alloy"
	}

	model := options.Model
	if model == "" {
		model = "tts-1"
	}

	format := options.Format
	if format == "" {
		format = "mp3"
	}

	speed := options.Speed
	if speed <= 0 {
		speed = 1.0
	}
	// Clamp speed to OpenAI limits
	if speed < 0.25 {
		speed = 0.25
	}
	if speed > 4.0 {
		speed = 4.0
	}

	// Prepare request payload
	requestBody := map[string]interface{}{
		"model":           model,
		"input":           text,
		"voice":           voice,
		"response_format": format,
		"speed":           speed,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	endpoint := p.baseURL + OpenAITTSEndpoint
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	log.Debug().
		Str("endpoint", endpoint).
		Str("voice", voice).
		Str("model", model).
		Str("format", format).
		Float64("speed", speed).
		Msg("Making OpenAI TTS request")

	// Make request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	log.Debug().
		Int("status", resp.StatusCode).
		Str("content_type", resp.Header.Get("Content-Type")).
		Msg("OpenAI TTS request successful")

	return resp.Body, nil
}

// IsAvailable checks if OpenAI provider is available
func (p *OpenAIProvider) IsAvailable(ctx context.Context) bool {
	if p.apiKey == "" {
		return false
	}

	// Test with a minimal request to validate API key
	testText := "test"
	requestBody := map[string]interface{}{
		"model": "tts-1",
		"input": testText,
		"voice": "alloy",
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return false
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+OpenAITTSEndpoint, bytes.NewReader(jsonData))
	if err != nil {
		return false
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Use a shorter timeout for availability check
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// OpenAIProviderFromConfig creates an OpenAI provider from configuration
func OpenAIProviderFromConfig(config map[string]interface{}) (*OpenAIProvider, error) {
	apiKey, ok := config["api_key"].(string)
	if !ok || apiKey == "" {
		return nil, fmt.Errorf("api_key is required for OpenAI provider")
	}

	provider := NewOpenAIProvider(apiKey)

	// Optional base URL override
	if baseURL, ok := config["base_url"].(string); ok && baseURL != "" {
		provider.baseURL = strings.TrimSuffix(baseURL, "/")
	}

	return provider, nil
}

// OpenAIError represents an error from OpenAI API
type OpenAIError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

func (e OpenAIError) String() string {
	return fmt.Sprintf("OpenAI API Error: %s (type: %s, code: %s)", e.Error.Message, e.Error.Type, e.Error.Code)
}

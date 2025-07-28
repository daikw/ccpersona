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

// OpenAIProvider implements TTS using OpenAI Audio API
type OpenAIProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
	apiKey     string
}

// NewOpenAIProvider creates a new OpenAI TTS provider
func NewOpenAIProvider(config *ProviderConfig) (Provider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	// Set defaults if not provided
	if config.OpenAI == nil {
		config.OpenAI = &OpenAIConfig{
			Model:  "tts-1",
			Voice:  "alloy",
			Speed:  1.0,
			Format: "mp3",
		}
	}

	return &OpenAIProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		apiKey: config.APIKey,
	}, nil
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "OpenAI Audio API"
}

// IsAvailable checks if OpenAI API is available
func (p *OpenAIProvider) IsAvailable(ctx context.Context) bool {
	// Simple health check - try to make a minimal request to test API key
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return false
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("User-Agent", "ccpersona-tts/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("OpenAI API availability check failed")
		return false
	}
	defer resp.Body.Close()

	available := resp.StatusCode == http.StatusOK
	log.Debug().Bool("available", available).Int("status", resp.StatusCode).Msg("OpenAI API availability")
	return available
}

// Synthesize converts text to speech using OpenAI API
func (p *OpenAIProvider) Synthesize(ctx context.Context, text string, options *SynthesizeOptions) (io.ReadCloser, error) {
	// Build request payload
	payload := map[string]interface{}{
		"model": p.config.OpenAI.Model,
		"input": text,
		"voice": p.config.OpenAI.Voice,
	}

	// Apply options if provided
	if options != nil {
		if options.Voice != "" {
			payload["voice"] = options.Voice
		}
		if options.Speed > 0 {
			payload["speed"] = options.Speed
		}
		if options.Format != "" {
			payload["response_format"] = string(options.Format)
		}
	}

	// Apply config settings
	if p.config.OpenAI.Speed > 0 {
		payload["speed"] = p.config.OpenAI.Speed
	}
	if p.config.OpenAI.Format != "" {
		payload["response_format"] = p.config.OpenAI.Format
	}

	// Serialize payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	log.Debug().
		Str("model", p.config.OpenAI.Model).
		Str("voice", p.config.OpenAI.Voice).
		Float32("speed", p.config.OpenAI.Speed).
		Str("format", p.config.OpenAI.Format).
		Msg("Synthesizing with OpenAI")

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/audio/speech", bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ccpersona-tts/1.0")

	// Make request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("OpenAI API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	log.Info().Msg("OpenAI TTS synthesis successful")
	return resp.Body, nil
}

// GetSupportedFormats returns supported audio formats for OpenAI
func (p *OpenAIProvider) GetSupportedFormats() []AudioFormat {
	return []AudioFormat{
		AudioFormatMP3,
		AudioFormatOGG, // opus
		AudioFormatAAC,
		AudioFormatFLAC,
	}
}

// GetDefaultFormat returns the default format for OpenAI
func (p *OpenAIProvider) GetDefaultFormat() AudioFormat {
	return AudioFormatMP3
}

// GetSupportedVoices returns available voices for OpenAI
func (p *OpenAIProvider) GetSupportedVoices() []string {
	return []string{
		"alloy",
		"echo", 
		"fable",
		"onyx",
		"nova",
		"shimmer",
	}
}

// GetSupportedModels returns available models for OpenAI
func (p *OpenAIProvider) GetSupportedModels() []string {
	return []string{
		"tts-1",    // Standard quality, faster
		"tts-1-hd", // High quality, slower
	}
}
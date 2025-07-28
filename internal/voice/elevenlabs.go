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

// ElevenLabsProvider implements TTS using ElevenLabs API
type ElevenLabsProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
	apiKey     string
	baseURL    string
}

// NewElevenLabsProvider creates a new ElevenLabs TTS provider
func NewElevenLabsProvider(config *ProviderConfig) (Provider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("ElevenLabs API key is required")
	}

	// Set defaults if not provided
	if config.ElevenLabs == nil {
		config.ElevenLabs = &ElevenLabsConfig{
			VoiceID: "21m00Tcm4TlvDq8ikWAM", // Rachel (default voice)
			Model:   "eleven_monolingual_v1",
			VoiceSettings: &VoiceSettings{
				Stability:       0.5,
				SimilarityBoost: 0.5,
				Style:           0.0,
				UseSpeakerBoost: true,
			},
		}
	}

	return &ElevenLabsProvider{
		config:  config,
		apiKey:  config.APIKey,
		baseURL: "https://api.elevenlabs.io/v1",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// Name returns the provider name
func (p *ElevenLabsProvider) Name() string {
	return "ElevenLabs TTS"
}

// IsAvailable checks if ElevenLabs API is available
func (p *ElevenLabsProvider) IsAvailable(ctx context.Context) bool {
	// Check API key validity by getting user info
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/user", nil)
	if err != nil {
		return false
	}

	req.Header.Set("xi-api-key", p.apiKey)
	req.Header.Set("User-Agent", "ccpersona-tts/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("ElevenLabs API availability check failed")
		return false
	}
	defer resp.Body.Close()

	available := resp.StatusCode == http.StatusOK
	log.Debug().Bool("available", available).Int("status", resp.StatusCode).Msg("ElevenLabs API availability")
	return available
}

// Synthesize converts text to speech using ElevenLabs API
func (p *ElevenLabsProvider) Synthesize(ctx context.Context, text string, options *SynthesizeOptions) (io.ReadCloser, error) {
	voiceID := p.config.ElevenLabs.VoiceID
	if options != nil && options.Voice != "" {
		voiceID = options.Voice
	}

	// Build request payload
	payload := map[string]interface{}{
		"text":           text,
		"model_id":       p.config.ElevenLabs.Model,
		"voice_settings": p.config.ElevenLabs.VoiceSettings,
	}

	// Add pronunciation dictionary if configured
	if p.config.ElevenLabs.PronunciationDict != nil && len(p.config.ElevenLabs.PronunciationDict) > 0 {
		payload["pronunciation_dictionary_locators"] = []map[string]interface{}{
			{
				"pronunciation_dictionary_id": "", // This would need to be set up separately
				"version_id":                  "",
			},
		}
	}

	// Serialize payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	log.Debug().
		Str("voice_id", voiceID).
		Str("model", p.config.ElevenLabs.Model).
		Interface("voice_settings", p.config.ElevenLabs.VoiceSettings).
		Msg("Synthesizing with ElevenLabs")

	// Create request URL
	url := fmt.Sprintf("%s/text-to-speech/%s", p.baseURL, voiceID)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("xi-api-key", p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg")
	req.Header.Set("User-Agent", "ccpersona-tts/1.0")

	// Make request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("ElevenLabs API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	log.Info().Msg("ElevenLabs TTS synthesis successful")
	return resp.Body, nil
}

// GetSupportedFormats returns supported audio formats for ElevenLabs
func (p *ElevenLabsProvider) GetSupportedFormats() []AudioFormat {
	return []AudioFormat{
		AudioFormatMP3, // Default format
	}
}

// GetDefaultFormat returns the default format for ElevenLabs
func (p *ElevenLabsProvider) GetDefaultFormat() AudioFormat {
	return AudioFormatMP3
}

// GetVoices fetches available voices from ElevenLabs API
func (p *ElevenLabsProvider) GetVoices(ctx context.Context) ([]ElevenLabsVoice, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/voices", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("xi-api-key", p.apiKey)
	req.Header.Set("User-Agent", "ccpersona-tts/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ElevenLabs API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var voicesResp struct {
		Voices []ElevenLabsVoice `json:"voices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&voicesResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return voicesResp.Voices, nil
}

// GetModels fetches available models from ElevenLabs API
func (p *ElevenLabsProvider) GetModels(ctx context.Context) ([]ElevenLabsModel, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("xi-api-key", p.apiKey)
	req.Header.Set("User-Agent", "ccpersona-tts/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ElevenLabs API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var models []ElevenLabsModel
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return models, nil
}

// ElevenLabsVoice represents a voice from ElevenLabs API
type ElevenLabsVoice struct {
	VoiceID                   string                     `json:"voice_id"`
	Name                      string                     `json:"name"`
	Samples                   []ElevenLabsSample         `json:"samples,omitempty"`
	Category                  string                     `json:"category"`
	FinetuningState           string                     `json:"fine_tuning_state"`
	Labels                    map[string]string          `json:"labels,omitempty"`
	Description               string                     `json:"description,omitempty"`
	PreviewURL                string                     `json:"preview_url,omitempty"`
	AvailableForTiers         []string                   `json:"available_for_tiers,omitempty"`
	Settings                  *VoiceSettings             `json:"settings,omitempty"`
	Sharing                   *ElevenLabsVoiceSharing    `json:"sharing,omitempty"`
	HighQualityBaseModelIDs   []string                   `json:"high_quality_base_model_ids,omitempty"`
}

// ElevenLabsSample represents a voice sample
type ElevenLabsSample struct {
	SampleID   string `json:"sample_id"`
	FileName   string `json:"file_name"`
	MimeType   string `json:"mime_type"`
	SizeBytes  int    `json:"size_bytes"`
	Hash       string `json:"hash"`
}

// ElevenLabsVoiceSharing represents voice sharing settings
type ElevenLabsVoiceSharing struct {
	Status                     string    `json:"status"`
	HistoryItemSampleID        string    `json:"history_item_sample_id,omitempty"`
	OriginalVoiceID            string    `json:"original_voice_id,omitempty"`
	PublicOwnerID              string    `json:"public_owner_id,omitempty"`
	LikedByCount               int       `json:"liked_by_count"`
	ClonedByCount              int       `json:"cloned_by_count"`
	Name                       string    `json:"name,omitempty"`
	Description                string    `json:"description,omitempty"`
	Labels                     map[string]string `json:"labels,omitempty"`
	ReviewStatus               string    `json:"review_status,omitempty"`
	ReviewMessage              string    `json:"review_message,omitempty"`
	EnabledInLibrary           bool      `json:"enabled_in_library"`
}

// ElevenLabsModel represents a TTS model
type ElevenLabsModel struct {
	ModelID                    string   `json:"model_id"`
	Name                       string   `json:"name"`
	CanBeFinetuned             bool     `json:"can_be_finetuned"`
	CanDoTextToSpeech          bool     `json:"can_do_text_to_speech"`
	CanDoVoiceConversion       bool     `json:"can_do_voice_conversion"`
	TokenCostFactor            float64  `json:"token_cost_factor"`
	Description                string   `json:"description"`
	RequiresAlphaAccess        bool     `json:"requires_alpha_access"`
	MaxCharactersRequestFreeUser int     `json:"max_characters_request_free_user"`
	MaxCharactersRequestSubscribed int   `json:"max_characters_request_subscribed"`
	Languages                  []ElevenLabsLanguage `json:"languages"`
}

// ElevenLabsLanguage represents a supported language
type ElevenLabsLanguage struct {
	LanguageID string `json:"language_id"`
	Name       string `json:"name"`
}
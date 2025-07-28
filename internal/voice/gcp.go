package voice

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// GCPProvider implements TTS using Google Cloud Text-to-Speech
type GCPProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
	projectID  string
	credentials *GCPCredentials
	accessToken string
	tokenExpiry time.Time
}

// NewGCPProvider creates a new Google Cloud TTS provider
func NewGCPProvider(config *ProviderConfig) (Provider, error) {
	// Set defaults if not provided
	if config.GCP == nil {
		config.GCP = &GCPConfig{
			VoiceName:       "en-US-Wavenet-D",
			LanguageCode:    "en-US",
			SsmlGender:      "NEUTRAL",
			AudioEncoding:   "MP3",
			SampleRateHertz: 24000,
			SpeakingRate:    1.0,
			Pitch:           0.0,
			VolumeGainDb:    0.0,
		}
	}

	projectID := config.ProjectID
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
		if projectID == "" {
			return nil, fmt.Errorf("Google Cloud project ID is required")
		}
	}

	// Get GCP credentials
	credentials := config.GCP.Credentials
	if credentials == nil {
		credentials = &GCPCredentials{}
		if err := loadGCPCredentials(credentials); err != nil {
			return nil, fmt.Errorf("failed to load GCP credentials: %w", err)
		}
	}

	return &GCPProvider{
		config:      config,
		projectID:   projectID,
		credentials: credentials,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// Name returns the provider name
func (p *GCPProvider) Name() string {
	return "Google Cloud Text-to-Speech"
}

// IsAvailable checks if Google Cloud TTS is available
func (p *GCPProvider) IsAvailable(ctx context.Context) bool {
	// Try to get an access token and make a simple API call
	if err := p.ensureAccessToken(ctx); err != nil {
		log.Debug().Err(err).Msg("Failed to get GCP access token")
		return false
	}

	// Try to list voices to test connectivity
	req, err := p.createGCPRequest(ctx, "GET", "/v1/voices", nil)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create GCP test request")
		return false
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("GCP availability check failed")
		return false
	}
	defer resp.Body.Close()

	available := resp.StatusCode == http.StatusOK
	log.Debug().Bool("available", available).Int("status", resp.StatusCode).Msg("Google Cloud TTS availability")
	return available
}

// Synthesize converts text to speech using Google Cloud TTS
func (p *GCPProvider) Synthesize(ctx context.Context, text string, options *SynthesizeOptions) (io.ReadCloser, error) {
	// Ensure we have a valid access token
	if err := p.ensureAccessToken(ctx); err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Build request payload
	input := map[string]interface{}{
		"text": text,
	}

	voice := map[string]interface{}{
		"languageCode": p.config.GCP.LanguageCode,
		"name":         p.config.GCP.VoiceName,
		"ssmlGender":   p.config.GCP.SsmlGender,
	}

	audioConfig := map[string]interface{}{
		"audioEncoding":   p.config.GCP.AudioEncoding,
		"speakingRate":    p.config.GCP.SpeakingRate,
		"pitch":           p.config.GCP.Pitch,
		"volumeGainDb":    p.config.GCP.VolumeGainDb,
	}

	if p.config.GCP.SampleRateHertz > 0 {
		audioConfig["sampleRateHertz"] = p.config.GCP.SampleRateHertz
	}

	if len(p.config.GCP.EffectsProfileId) > 0 {
		audioConfig["effectsProfileId"] = p.config.GCP.EffectsProfileId
	}

	// Apply options if provided
	if options != nil {
		if options.Voice != "" {
			voice["name"] = options.Voice
		}
		if options.Speed > 0 {
			audioConfig["speakingRate"] = options.Speed
		}
		if options.Pitch != 0 {
			audioConfig["pitch"] = options.Pitch
		}
		if options.Volume > 0 {
			// Convert volume (0.0-1.0) to gain in dB (rough approximation)
			gainDb := 20 * (float64(options.Volume) - 1.0) // -20dB to 0dB range
			audioConfig["volumeGainDb"] = gainDb
		}
		if options.SampleRate > 0 {
			audioConfig["sampleRateHertz"] = options.SampleRate
		}
		if options.Format != "" {
			switch options.Format {
			case AudioFormatMP3:
				audioConfig["audioEncoding"] = "MP3"
			case AudioFormatWAV:
				audioConfig["audioEncoding"] = "LINEAR16"
			case AudioFormatOGG:
				audioConfig["audioEncoding"] = "OGG_OPUS"
			}
		}
	}

	payload := map[string]interface{}{
		"input":       input,
		"voice":       voice,
		"audioConfig": audioConfig,
	}

	// Handle custom voice if configured
	if p.config.GCP.CustomVoice != nil {
		voice["customVoice"] = map[string]interface{}{
			"model":         p.config.GCP.CustomVoice.Model,
			"reportedUsage": p.config.GCP.CustomVoice.ReportedUsage,
		}
	}

	// Serialize payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	log.Debug().
		Str("voice_name", p.config.GCP.VoiceName).
		Str("language_code", p.config.GCP.LanguageCode).
		Str("audio_encoding", p.config.GCP.AudioEncoding).
		Float64("speaking_rate", p.config.GCP.SpeakingRate).
		Msg("Synthesizing with Google Cloud TTS")

	// Create request
	endpoint := fmt.Sprintf("/v1/projects/%s/locations/global/text:synthesize", p.projectID)
	req, err := p.createGCPRequest(ctx, "POST", endpoint, payloadBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Make request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("Google Cloud TTS API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response to get base64-encoded audio
	var result struct {
		AudioContent string `json:"audioContent"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	resp.Body.Close()

	// Decode base64 audio
	audioData, err := base64.StdEncoding.DecodeString(result.AudioContent)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio data: %w", err)
	}

	log.Info().Msg("Google Cloud TTS synthesis successful")
	return io.NopCloser(bytes.NewReader(audioData)), nil
}

// GetSupportedFormats returns supported audio formats for Google Cloud TTS
func (p *GCPProvider) GetSupportedFormats() []AudioFormat {
	return []AudioFormat{
		AudioFormatMP3,     // MP3
		AudioFormatWAV,     // LINEAR16
		AudioFormatOGG,     // OGG_OPUS
	}
}

// GetDefaultFormat returns the default format for Google Cloud TTS
func (p *GCPProvider) GetDefaultFormat() AudioFormat {
	return AudioFormatMP3
}

// createGCPRequest creates an authenticated request for Google Cloud TTS
func (p *GCPProvider) createGCPRequest(ctx context.Context, method, path string, body []byte) (*http.Request, error) {
	baseURL := "https://texttospeech.googleapis.com"
	endpoint := baseURL + path

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+p.accessToken)
	req.Header.Set("User-Agent", "ccpersona-tts/1.0")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// ensureAccessToken ensures we have a valid access token
func (p *GCPProvider) ensureAccessToken(ctx context.Context) error {
	// Check if we have a valid token
	if p.accessToken != "" && time.Now().Before(p.tokenExpiry) {
		return nil
	}

	// Get a new access token
	if p.credentials.UseADC {
		return p.getTokenFromADC(ctx)
	}

	if p.credentials.ServiceAccountKey != "" {
		return p.getTokenFromServiceAccount(ctx)
	}

	return fmt.Errorf("no valid authentication method configured")
}

// getTokenFromADC gets an access token using Application Default Credentials
func (p *GCPProvider) getTokenFromADC(ctx context.Context) error {
	// This is a simplified implementation
	// In a real implementation, we would use the Google Cloud SDK's ADC mechanism
	
	// Try to get token from gcloud CLI
	return fmt.Errorf("Application Default Credentials not implemented yet - please use service account key")
}

// getTokenFromServiceAccount gets an access token using a service account key
func (p *GCPProvider) getTokenFromServiceAccount(ctx context.Context) error {
	// This is a simplified implementation
	// In a real implementation, we would:
	// 1. Load the service account JSON key file
	// 2. Create a JWT assertion
	// 3. Exchange it for an access token
	
	return fmt.Errorf("Service account authentication not implemented yet - please use Application Default Credentials")
}

// loadGCPCredentials loads GCP credentials from environment or default locations
func loadGCPCredentials(creds *GCPCredentials) error {
	// Try service account key file from environment
	if keyPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); keyPath != "" {
		creds.ServiceAccountKey = keyPath
		return nil
	}

	// Default to ADC
	creds.UseADC = true
	return nil
}

// GetVoices fetches available voices from Google Cloud TTS
func (p *GCPProvider) GetVoices(ctx context.Context) ([]GCPVoice, error) {
	if err := p.ensureAccessToken(ctx); err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	req, err := p.createGCPRequest(ctx, "GET", "/v1/voices", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Google Cloud TTS API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var voicesResp struct {
		Voices []GCPVoice `json:"voices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&voicesResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return voicesResp.Voices, nil
}

// GCPVoice represents a voice from Google Cloud TTS
type GCPVoice struct {
	LanguageCodes    []string `json:"languageCodes"`
	Name             string   `json:"name"`
	SsmlGender       string   `json:"ssmlGender"`
	NaturalSampleRateHertz int32 `json:"naturalSampleRateHertz"`
}
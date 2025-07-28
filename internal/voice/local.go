package voice

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

// LocalProvider implements TTS using local engines (VOICEVOX, AivisSpeech)
type LocalProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
}

// NewLocalProvider creates a new local TTS provider
func NewLocalProvider(config *ProviderConfig) (Provider, error) {
	// Set defaults if not provided
	if config.Local == nil {
		config.Local = &LocalConfig{
			Engine:             EngineAivisSpeech,
			VoicevoxSpeaker:    3,              // ずんだもん
			AivisSpeechSpeaker: 1512153248,     // Default AivisSpeech speaker
			VoicevoxURL:        VoicevoxURL,
			AivisSpeechURL:     AivisSpeechURL,
		}
	}

	return &LocalProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Name returns the provider name
func (p *LocalProvider) Name() string {
	switch p.config.Local.Engine {
	case EngineVoicevox:
		return "VOICEVOX Engine"
	case EngineAivisSpeech:
		return "AivisSpeech"
	default:
		return "Local TTS Engine"
	}
}

// IsAvailable checks if the local engine is available
func (p *LocalProvider) IsAvailable(ctx context.Context) bool {
	switch p.config.Local.Engine {
	case EngineVoicevox:
		return p.checkVoicevoxAvailable(ctx)
	case EngineAivisSpeech:
		return p.checkAivisSpeechAvailable(ctx)
	default:
		// Try both engines
		return p.checkVoicevoxAvailable(ctx) || p.checkAivisSpeechAvailable(ctx)
	}
}

// checkVoicevoxAvailable checks if VOICEVOX is available
func (p *LocalProvider) checkVoicevoxAvailable(ctx context.Context) bool {
	baseURL := p.config.Local.VoicevoxURL
	if baseURL == "" {
		baseURL = VoicevoxURL
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/version", nil)
	if err != nil {
		return false
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("VOICEVOX availability check failed")
		return false
	}
	defer resp.Body.Close()

	available := resp.StatusCode == http.StatusOK
	log.Debug().Bool("available", available).Msg("VOICEVOX availability")
	return available
}

// checkAivisSpeechAvailable checks if AivisSpeech is available
func (p *LocalProvider) checkAivisSpeechAvailable(ctx context.Context) bool {
	baseURL := p.config.Local.AivisSpeechURL
	if baseURL == "" {
		baseURL = AivisSpeechURL
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/speakers", nil)
	if err != nil {
		return false
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("AivisSpeech availability check failed")
		return false
	}
	defer resp.Body.Close()

	available := resp.StatusCode == http.StatusOK
	log.Debug().Bool("available", available).Msg("AivisSpeech availability")
	return available
}

// Synthesize converts text to speech using local engines
func (p *LocalProvider) Synthesize(ctx context.Context, text string, options *SynthesizeOptions) (io.ReadCloser, error) {
	// Determine which engine to use
	engine := p.config.Local.Engine
	if engine == "" {
		// Auto-detect available engine
		if p.checkAivisSpeechAvailable(ctx) {
			engine = EngineAivisSpeech
		} else if p.checkVoicevoxAvailable(ctx) {
			engine = EngineVoicevox
		} else {
			return nil, fmt.Errorf("no local TTS engine available")
		}
	}

	log.Debug().Str("engine", engine).Msg("Using local TTS engine")

	switch engine {
	case EngineVoicevox:
		return p.synthesizeVoicevox(ctx, text, options)
	case EngineAivisSpeech:
		return p.synthesizeAivisSpeech(ctx, text, options)
	default:
		return nil, fmt.Errorf("unknown local engine: %s", engine)
	}
}

// synthesizeVoicevox uses VOICEVOX ENGINE for synthesis
func (p *LocalProvider) synthesizeVoicevox(ctx context.Context, text string, options *SynthesizeOptions) (io.ReadCloser, error) {
	baseURL := p.config.Local.VoicevoxURL
	if baseURL == "" {
		baseURL = VoicevoxURL
	}

	speakerID := p.config.Local.VoicevoxSpeaker
	if speakerID == 0 {
		speakerID = 3 // Default to ずんだもん
	}

	// Create audio query
	queryURL := fmt.Sprintf("%s/audio_query?speaker=%d", baseURL, speakerID)
	queryURL += "&text=" + url.QueryEscape(text)

	req, err := http.NewRequestWithContext(ctx, "POST", queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio query request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("audio query failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	queryData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read query response: %w", err)
	}

	// Synthesize audio
	synthURL := fmt.Sprintf("%s/synthesis?speaker=%d", baseURL, speakerID)
	
	req, err = http.NewRequestWithContext(ctx, "POST", synthURL, bytes.NewReader(queryData))
	if err != nil {
		return nil, fmt.Errorf("failed to create synthesis request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to synthesize: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("synthesis failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	log.Info().Msg("VOICEVOX synthesis successful")
	return resp.Body, nil
}

// synthesizeAivisSpeech uses AivisSpeech for synthesis
func (p *LocalProvider) synthesizeAivisSpeech(ctx context.Context, text string, options *SynthesizeOptions) (io.ReadCloser, error) {
	baseURL := p.config.Local.AivisSpeechURL
	if baseURL == "" {
		baseURL = AivisSpeechURL
	}

	speakerID := p.config.Local.AivisSpeechSpeaker
	if speakerID == 0 {
		speakerID = 1512153248 // Default AivisSpeech speaker
	}

	// Create audio query (VOICEVOX compatible API)
	queryURL := fmt.Sprintf("%s/audio_query?speaker=%d", baseURL, speakerID)
	queryURL += "&text=" + url.QueryEscape(text)

	req, err := http.NewRequestWithContext(ctx, "POST", queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio query request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("audio query failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	queryData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read query response: %w", err)
	}

	// Synthesize audio
	synthURL := fmt.Sprintf("%s/synthesis?speaker=%d", baseURL, speakerID)
	
	req, err = http.NewRequestWithContext(ctx, "POST", synthURL, bytes.NewReader(queryData))
	if err != nil {
		return nil, fmt.Errorf("failed to create synthesis request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to synthesize: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("synthesis failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	log.Info().Msg("AivisSpeech synthesis successful")
	return resp.Body, nil
}

// GetSupportedFormats returns supported audio formats for local engines
func (p *LocalProvider) GetSupportedFormats() []AudioFormat {
	return []AudioFormat{
		AudioFormatWAV, // Both VOICEVOX and AivisSpeech output WAV
	}
}

// GetDefaultFormat returns the default format for local engines
func (p *LocalProvider) GetDefaultFormat() AudioFormat {
	return AudioFormatWAV
}

// SaveToFile saves the audio stream to a file
func (p *LocalProvider) SaveToFile(audioStream io.ReadCloser, filename string) error {
	defer audioStream.Close()

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, audioStream)
	if err != nil {
		return fmt.Errorf("failed to write audio data: %w", err)
	}

	log.Info().Str("file", filename).Msg("Audio saved to file")
	return nil
}

// SaveToTempFile saves the audio stream to a temporary file and returns the path
func (p *LocalProvider) SaveToTempFile(audioStream io.ReadCloser) (string, error) {
	defer audioStream.Close()

	tmpFile, err := os.CreateTemp("", "ccpersona_tts_*.wav")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, audioStream)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write audio data: %w", err)
	}

	log.Debug().Str("file", tmpFile.Name()).Msg("Audio saved to temp file")
	return tmpFile.Name(), nil
}
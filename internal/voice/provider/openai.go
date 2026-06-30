package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	OpenAIBaseURL        = "https://api.openai.com/v1"
	OpenAITTSEndpoint    = "/audio/speech"
	OpenAIModelsEndpoint = "/models"
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
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

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
	// Official OpenAI requires an API key; an OpenAI-compatible local server
	// (non-official host) is reachable without one. Classify by host so URL
	// spelling variations cannot bypass the key requirement.
	if official, err := isOfficialOpenAIBaseURL(p.baseURL); err != nil || (p.apiKey == "" && official) {
		return false
	}

	// Probe with a non-billed GET {baseURL}/models rather than a real
	// /audio/speech synthesis, which would incur TTS charges per check.
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+OpenAIModelsEndpoint, nil)
	if err != nil {
		return false
	}

	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	// Use a shorter timeout for availability check
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// openAIOfficialHost is the host of the official OpenAI API. A base_url that
// resolves to this host is treated as the official endpoint regardless of path
// or trailing slash, so api_key cannot be bypassed via spelling variations.
var openAIOfficialHost = func() string {
	u, _ := url.Parse(OpenAIBaseURL)
	return u.Host
}()

// isOfficialOpenAIBaseURL reports whether baseURL points at the official OpenAI
// API host. An empty baseURL means "use the official default" and is official.
// The bool return distinguishes a parse/scheme error from a non-official host.
func isOfficialOpenAIBaseURL(baseURL string) (official bool, err error) {
	if baseURL == "" {
		return true, nil
	}
	u, parseErr := url.Parse(baseURL)
	if parseErr != nil {
		return false, fmt.Errorf("invalid base_url %q: %w", baseURL, parseErr)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false, fmt.Errorf("invalid base_url %q: scheme must be http or https", baseURL)
	}
	return u.Host == openAIOfficialHost, nil
}

// normalizeOpenAIBaseURL returns the API base URL used when appending endpoint
// paths. Local OpenAI-compatible server examples often use the server root; in
// that case we normalize to /v1 so requests hit /v1/models and /v1/audio/speech.
func normalizeOpenAIBaseURL(baseURL string) (string, error) {
	if baseURL == "" {
		return OpenAIBaseURL, nil
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base_url %q: %w", baseURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("invalid base_url %q: scheme must be http or https", baseURL)
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = "/v1"
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

// OpenAIProviderFromConfig creates an OpenAI provider from configuration.
// api_key is required for the official OpenAI endpoint, but optional when
// base_url points at an OpenAI-compatible local TTS server (no auth).
func OpenAIProviderFromConfig(config map[string]interface{}) (*OpenAIProvider, error) {
	apiKey, _ := config["api_key"].(string)
	baseURL, _ := config["base_url"].(string)

	official, err := isOfficialOpenAIBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	if apiKey == "" && official {
		return nil, fmt.Errorf("api_key is required for OpenAI provider")
	}

	provider := NewOpenAIProvider(apiKey)

	// Optional base URL override
	if baseURL != "" {
		normalized, err := normalizeOpenAIBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		provider.baseURL = normalized
	}

	// Optional HTTP timeout override (local GPU inference can be slow on first call)
	if timeout, ok := configInt(config["timeout_seconds"]); ok && timeout > 0 {
		provider.httpClient.Timeout = time.Duration(timeout) * time.Second
	}

	return provider, nil
}

// configInt coerces a config value to int, tolerating the float64 that
// encoding/json produces for numbers as well as a plain int from Go callers.
func configInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
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

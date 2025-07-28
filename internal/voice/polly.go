package voice

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// PollyProvider implements TTS using Amazon Polly
type PollyProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
	region     string
	credentials *AWSCredentials
}

// NewPollyProvider creates a new Amazon Polly TTS provider
func NewPollyProvider(config *ProviderConfig) (Provider, error) {
	// Set defaults if not provided
	if config.Polly == nil {
		config.Polly = &PollyConfig{
			VoiceID:      "Joanna",
			Engine:       "neural",
			LanguageCode: "en-US",
			OutputFormat: "mp3",
			SampleRate:   "22050",
		}
	}

	region := config.Region
	if region == "" {
		region = "us-east-1" // Default region
	}

	// Get AWS credentials
	credentials := config.Polly.Credentials
	if credentials == nil {
		// Try to get credentials from environment or default locations
		credentials = &AWSCredentials{}
		if err := loadAWSCredentials(credentials); err != nil {
			return nil, fmt.Errorf("failed to load AWS credentials: %w", err)
		}
	}

	return &PollyProvider{
		config:      config,
		region:      region,
		credentials: credentials,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// Name returns the provider name
func (p *PollyProvider) Name() string {
	return "Amazon Polly"
}

// IsAvailable checks if Amazon Polly is available
func (p *PollyProvider) IsAvailable(ctx context.Context) bool {
	// Try to list voices to test credentials and connectivity
	req, err := p.createPollyRequest(ctx, "GET", "/v1/voices", nil, nil)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create Polly test request")
		return false
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("Polly availability check failed")
		return false
	}
	defer resp.Body.Close()

	available := resp.StatusCode == http.StatusOK
	log.Debug().Bool("available", available).Int("status", resp.StatusCode).Msg("Amazon Polly availability")
	return available
}

// Synthesize converts text to speech using Amazon Polly
func (p *PollyProvider) Synthesize(ctx context.Context, text string, options *SynthesizeOptions) (io.ReadCloser, error) {
	// Build request payload
	payload := map[string]interface{}{
		"Text":         text,
		"VoiceId":      p.config.Polly.VoiceID,
		"OutputFormat": p.config.Polly.OutputFormat,
		"SampleRate":   p.config.Polly.SampleRate,
		"Engine":       p.config.Polly.Engine,
	}

	// Apply language code if specified
	if p.config.Polly.LanguageCode != "" {
		payload["LanguageCode"] = p.config.Polly.LanguageCode
	}

	// Apply options if provided
	if options != nil {
		if options.Voice != "" {
			payload["VoiceId"] = options.Voice
		}
		if options.Format != "" {
			payload["OutputFormat"] = string(options.Format)
		}
		if options.SampleRate > 0 {
			payload["SampleRate"] = fmt.Sprintf("%d", options.SampleRate)
		}
	}

	// Handle SSML
	textType := "text"
	if p.config.Polly.SSML {
		textType = "ssml"
	}
	payload["TextType"] = textType

	// Add speech marks if configured
	if len(p.config.Polly.SpeechMarks) > 0 {
		payload["SpeechMarkTypes"] = p.config.Polly.SpeechMarks
	}

	// Add lexicons if configured
	if len(p.config.Polly.Lexicons) > 0 {
		payload["LexiconNames"] = p.config.Polly.Lexicons
	}

	// Serialize payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	log.Debug().
		Str("voice_id", p.config.Polly.VoiceID).
		Str("engine", p.config.Polly.Engine).
		Str("output_format", p.config.Polly.OutputFormat).
		Str("language_code", p.config.Polly.LanguageCode).
		Msg("Synthesizing with Amazon Polly")

	// Create request
	req, err := p.createPollyRequest(ctx, "POST", "/v1/speech", payloadBytes, map[string]string{
		"Content-Type": "application/x-amz-json-1.0",
	})
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
		return nil, fmt.Errorf("Amazon Polly API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	log.Info().Msg("Amazon Polly TTS synthesis successful")
	return resp.Body, nil
}

// GetSupportedFormats returns supported audio formats for Amazon Polly
func (p *PollyProvider) GetSupportedFormats() []AudioFormat {
	return []AudioFormat{
		AudioFormatMP3,
		AudioFormatOGG, // ogg_vorbis
		AudioFormatPCM,
	}
}

// GetDefaultFormat returns the default format for Amazon Polly
func (p *PollyProvider) GetDefaultFormat() AudioFormat {
	return AudioFormatMP3
}

// createPollyRequest creates an authenticated request for Amazon Polly
func (p *PollyProvider) createPollyRequest(ctx context.Context, method, path string, body []byte, headers map[string]string) (*http.Request, error) {
	endpoint := fmt.Sprintf("https://polly.%s.amazonaws.com%s", p.region, path)
	
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set headers
	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	// Sign the request using AWS Signature Version 4
	if err := p.signRequest(req, body); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	return req, nil
}

// signRequest signs the request using AWS Signature Version 4
func (p *PollyProvider) signRequest(req *http.Request, body []byte) error {
	now := time.Now().UTC()
	
	// Step 1: Create canonical request
	canonicalHeaders, signedHeaders := p.getCanonicalHeaders(req, now)
	
	bodyHash := sha256.Sum256(body)
	if body == nil {
		bodyHash = sha256.Sum256([]byte{})
	}

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%x",
		req.Method,
		req.URL.Path,
		req.URL.RawQuery,
		canonicalHeaders,
		signedHeaders,
		bodyHash,
	)

	// Step 2: Create string to sign
	algorithm := "AWS4-HMAC-SHA256"
	credentialScope := fmt.Sprintf("%s/%s/polly/aws4_request",
		now.Format("20060102"),
		p.region,
	)
	
	hasher := sha256.New()
	hasher.Write([]byte(canonicalRequest))
	canonicalRequestHash := hex.EncodeToString(hasher.Sum(nil))

	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s",
		algorithm,
		now.Format("20060102T150405Z"),
		credentialScope,
		canonicalRequestHash,
	)

	// Step 3: Calculate signature
	signature, err := p.calculateSignature(stringToSign, now)
	if err != nil {
		return err
	}

	// Step 4: Add authorization header
	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm,
		p.credentials.AccessKeyID,
		credentialScope,
		signedHeaders,
		signature,
	)

	req.Header.Set("Authorization", authorization)
	req.Header.Set("X-Amz-Date", now.Format("20060102T150405Z"))

	if p.credentials.SessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", p.credentials.SessionToken)
	}

	return nil
}

// getCanonicalHeaders creates canonical headers for AWS signature
func (p *PollyProvider) getCanonicalHeaders(req *http.Request, now time.Time) (string, string) {
	// Set required headers
	req.Header.Set("Host", req.Host)
	req.Header.Set("X-Amz-Date", now.Format("20060102T150405Z"))

	var headerNames []string
	headerMap := make(map[string]string)

	for name, values := range req.Header {
		lowerName := strings.ToLower(name)
		headerNames = append(headerNames, lowerName)
		headerMap[lowerName] = strings.Join(values, ",")
	}

	sort.Strings(headerNames)

	var canonicalHeaders strings.Builder
	for _, name := range headerNames {
		canonicalHeaders.WriteString(name)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(headerMap[name])
		canonicalHeaders.WriteString("\n")
	}

	signedHeaders := strings.Join(headerNames, ";")
	return canonicalHeaders.String(), signedHeaders
}

// calculateSignature calculates the AWS signature
func (p *PollyProvider) calculateSignature(stringToSign string, now time.Time) (string, error) {
	dateKey := p.hmacSHA256([]byte("AWS4"+p.credentials.SecretAccessKey), now.Format("20060102"))
	regionKey := p.hmacSHA256(dateKey, p.region)
	serviceKey := p.hmacSHA256(regionKey, "polly")
	signingKey := p.hmacSHA256(serviceKey, "aws4_request")
	
	signature := p.hmacSHA256(signingKey, stringToSign)
	return hex.EncodeToString(signature), nil
}

// hmacSHA256 computes HMAC-SHA256
func (p *PollyProvider) hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

// loadAWSCredentials loads AWS credentials from environment or default locations
func loadAWSCredentials(creds *AWSCredentials) error {
	// Try environment variables first
	if accessKey := os.Getenv("AWS_ACCESS_KEY_ID"); accessKey != "" {
		creds.AccessKeyID = accessKey
		creds.SecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		creds.SessionToken = os.Getenv("AWS_SESSION_TOKEN")
		return nil
	}

	// Try AWS profile if specified
	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		creds.Profile = profile
		// In a real implementation, we would parse ~/.aws/credentials
		// For now, return an error if no env vars are set
		return fmt.Errorf("AWS profile loading not implemented yet - please use environment variables")
	}

	// Check if we should use instance role
	if os.Getenv("AWS_USE_INSTANCE_ROLE") == "true" {
		creds.UseInstanceRole = true
		// In a real implementation, we would fetch credentials from the instance metadata service
		return fmt.Errorf("AWS instance role authentication not implemented yet")
	}

	return fmt.Errorf("no AWS credentials found - set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables")
}

// GetVoices fetches available voices from Amazon Polly
func (p *PollyProvider) GetVoices(ctx context.Context) ([]PollyVoice, error) {
	req, err := p.createPollyRequest(ctx, "GET", "/v1/voices", nil, nil)
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
		return nil, fmt.Errorf("Amazon Polly API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var voicesResp struct {
		Voices []PollyVoice `json:"Voices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&voicesResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return voicesResp.Voices, nil
}

// PollyVoice represents a voice from Amazon Polly
type PollyVoice struct {
	Gender         string   `json:"Gender"`
	Id             string   `json:"Id"`
	LanguageCode   string   `json:"LanguageCode"`
	LanguageName   string   `json:"LanguageName"`
	Name           string   `json:"Name"`
	AdditionalLanguageCodes []string `json:"AdditionalLanguageCodes,omitempty"`
	SupportedEngines []string `json:"SupportedEngines,omitempty"`
}
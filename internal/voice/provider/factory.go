package provider

import (
	"fmt"
	"os"
)

// DefaultFactory is the default provider factory
type DefaultFactory struct{}

// NewFactory creates a new provider factory
func NewFactory() *DefaultFactory {
	return &DefaultFactory{}
}

// CreateProvider creates a provider instance by name
func (f *DefaultFactory) CreateProvider(providerName string, config map[string]interface{}) (Provider, error) {
	switch providerName {
	case "openai":
		return f.createOpenAIProvider(config)
	case "elevenlabs":
		return f.createElevenLabsProvider(config)
	case "polly":
		return f.createPollyProvider(config)
	case "gcp":
		return f.createGCPProvider(config)
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}
}

// ListProviders returns available provider names
func (f *DefaultFactory) ListProviders() []string {
	return []string{"openai", "elevenlabs", "polly", "gcp"}
}

// createOpenAIProvider creates an OpenAI provider with configuration
func (f *DefaultFactory) createOpenAIProvider(config map[string]interface{}) (Provider, error) {
	// Try to get API key from config first
	apiKey, ok := config["api_key"].(string)
	if !ok || apiKey == "" {
		// Fallback to environment variable
		apiKey = os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OpenAI API key not found in config or OPENAI_API_KEY environment variable")
		}
	}

	// Add the API key to config if it was from env
	if _, exists := config["api_key"]; !exists {
		config["api_key"] = apiKey
	}

	return OpenAIProviderFromConfig(config)
}

// createElevenLabsProvider creates an ElevenLabs provider with configuration
func (f *DefaultFactory) createElevenLabsProvider(config map[string]interface{}) (Provider, error) {
	// Try to get API key from config first
	apiKey, ok := config["api_key"].(string)
	if !ok || apiKey == "" {
		// Fallback to environment variable
		apiKey = os.Getenv("ELEVENLABS_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("ElevenLabs API key not found in config or ELEVENLABS_API_KEY environment variable")
		}
	}

	// Add the API key to config if it was from env
	if _, exists := config["api_key"]; !exists {
		config["api_key"] = apiKey
	}

	return ElevenLabsProviderFromConfig(config)
}

// createPollyProvider creates an Amazon Polly provider with configuration
func (f *DefaultFactory) createPollyProvider(config map[string]interface{}) (Provider, error) {
	// Region is optional, defaults to us-east-1 in the provider
	return PollyProviderFromConfig(config)
}

// createGCPProvider creates a Google Cloud TTS provider with configuration
func (f *DefaultFactory) createGCPProvider(config map[string]interface{}) (Provider, error) {
	// Authentication is handled via GOOGLE_APPLICATION_CREDENTIALS or ADC
	return GCPProviderFromConfig(config)
}

// GetProviderWithDefaults creates a provider with default configuration
func (f *DefaultFactory) GetProviderWithDefaults(providerName string) (Provider, error) {
	config := make(map[string]interface{})

	switch providerName {
	case "openai":
		// OpenAI defaults - API key will be loaded from environment
		config["model"] = "tts-1"
		config["voice"] = "alloy"
		config["format"] = "mp3"
		config["speed"] = 1.0
	case "elevenlabs":
		// ElevenLabs defaults - API key will be loaded from environment
		config["model"] = "eleven_multilingual_v2"
		config["voice"] = "21m00Tcm4TlvDq8ikWAM" // Rachel
		config["format"] = "mp3"
		config["stability"] = 0.5
		config["similarity_boost"] = 0.5
		config["style"] = 0.0
		config["use_speaker_boost"] = true
	case "polly":
		// Amazon Polly defaults - uses AWS credentials from environment/profile
		config["region"] = "us-east-1"
		config["voice"] = "Joanna"
		config["model"] = "neural" // neural engine
		config["format"] = "mp3"
		config["quality"] = "22050" // sample rate
	case "gcp":
		// Google Cloud TTS defaults - uses GOOGLE_APPLICATION_CREDENTIALS or ADC
		config["voice"] = "ja-JP-Neural2-B" // Japanese Neural2 male voice
		config["language"] = "ja-JP"
		config["engine"] = "neural2"
		config["format"] = "mp3"
	}

	return f.CreateProvider(providerName, config)
}

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
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}
}

// ListProviders returns available provider names
func (f *DefaultFactory) ListProviders() []string {
	return []string{"openai"}
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
	}

	return f.CreateProvider(providerName, config)
}

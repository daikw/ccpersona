package provider

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	assert.NotNil(t, factory)
}

func TestListProviders(t *testing.T) {
	factory := NewFactory()
	providers := factory.ListProviders()

	assert.Len(t, providers, 4)
	assert.Contains(t, providers, "openai")
	assert.Contains(t, providers, "elevenlabs")
	assert.Contains(t, providers, "polly")
	assert.Contains(t, providers, "gcp")
}

func TestCreateProvider_UnknownProvider(t *testing.T) {
	factory := NewFactory()

	_, err := factory.CreateProvider("unknown", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestCreateProvider_OpenAI(t *testing.T) {
	factory := NewFactory()

	t.Run("fails without API key", func(t *testing.T) {
		// Ensure env var is not set
		originalKey := os.Getenv("OPENAI_API_KEY")
		os.Unsetenv("OPENAI_API_KEY")
		defer func() {
			if originalKey != "" {
				os.Setenv("OPENAI_API_KEY", originalKey)
			}
		}()

		_, err := factory.CreateProvider("openai", map[string]interface{}{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API key not found")
	})

	t.Run("succeeds with API key in config", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key": "test-api-key",
		}
		provider, err := factory.CreateProvider("openai", config)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "openai", provider.Name())
	})

	t.Run("succeeds with API key in env", func(t *testing.T) {
		originalKey := os.Getenv("OPENAI_API_KEY")
		os.Setenv("OPENAI_API_KEY", "test-env-api-key")
		defer func() {
			if originalKey != "" {
				os.Setenv("OPENAI_API_KEY", originalKey)
			} else {
				os.Unsetenv("OPENAI_API_KEY")
			}
		}()

		provider, err := factory.CreateProvider("openai", map[string]interface{}{})
		assert.NoError(t, err)
		assert.NotNil(t, provider)
	})
}

func TestCreateProvider_ElevenLabs(t *testing.T) {
	factory := NewFactory()

	t.Run("fails without API key", func(t *testing.T) {
		originalKey := os.Getenv("ELEVENLABS_API_KEY")
		os.Unsetenv("ELEVENLABS_API_KEY")
		defer func() {
			if originalKey != "" {
				os.Setenv("ELEVENLABS_API_KEY", originalKey)
			}
		}()

		_, err := factory.CreateProvider("elevenlabs", map[string]interface{}{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API key not found")
	})

	t.Run("succeeds with API key in config", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key": "test-api-key",
		}
		provider, err := factory.CreateProvider("elevenlabs", config)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "elevenlabs", provider.Name())
	})
}

func TestCreateProvider_Polly(t *testing.T) {
	factory := NewFactory()

	t.Run("creates polly provider with default config", func(t *testing.T) {
		provider, err := factory.CreateProvider("polly", map[string]interface{}{})
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "polly", provider.Name())
	})

	t.Run("creates polly provider with region", func(t *testing.T) {
		config := map[string]interface{}{
			"region": "us-west-2",
		}
		provider, err := factory.CreateProvider("polly", config)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
	})
}

func TestCreateProvider_GCP(t *testing.T) {
	// GCP provider creation requires network access, skip in short mode
	if testing.Short() {
		t.Skip("Skipping GCP provider test in short mode - requires network access")
	}

	factory := NewFactory()

	t.Run("creates GCP provider", func(t *testing.T) {
		provider, err := factory.CreateProvider("gcp", map[string]interface{}{})
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "gcp", provider.Name())
	})

	t.Run("creates GCP provider with project ID", func(t *testing.T) {
		config := map[string]interface{}{
			"project_id": "my-project",
			"region":     "us-central1",
		}
		provider, err := factory.CreateProvider("gcp", config)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
	})
}

func TestGetProviderWithDefaults(t *testing.T) {
	factory := NewFactory()

	t.Run("openai with defaults", func(t *testing.T) {
		originalKey := os.Getenv("OPENAI_API_KEY")
		os.Setenv("OPENAI_API_KEY", "test-key")
		defer func() {
			if originalKey != "" {
				os.Setenv("OPENAI_API_KEY", originalKey)
			} else {
				os.Unsetenv("OPENAI_API_KEY")
			}
		}()

		provider, err := factory.GetProviderWithDefaults("openai")
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "openai", provider.Name())
	})

	t.Run("polly with defaults", func(t *testing.T) {
		provider, err := factory.GetProviderWithDefaults("polly")
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "polly", provider.Name())
	})

	t.Run("gcp with defaults", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping GCP provider test in short mode - requires network access")
		}
		provider, err := factory.GetProviderWithDefaults("gcp")
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "gcp", provider.Name())
	})

	t.Run("elevenlabs with defaults", func(t *testing.T) {
		originalKey := os.Getenv("ELEVENLABS_API_KEY")
		os.Setenv("ELEVENLABS_API_KEY", "test-key")
		defer func() {
			if originalKey != "" {
				os.Setenv("ELEVENLABS_API_KEY", originalKey)
			} else {
				os.Unsetenv("ELEVENLABS_API_KEY")
			}
		}()

		provider, err := factory.GetProviderWithDefaults("elevenlabs")
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "elevenlabs", provider.Name())
	})

	t.Run("unknown provider returns error", func(t *testing.T) {
		_, err := factory.GetProviderWithDefaults("unknown")
		assert.Error(t, err)
	})
}

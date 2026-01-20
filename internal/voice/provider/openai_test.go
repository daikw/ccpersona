package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewOpenAIProvider(t *testing.T) {
	provider := NewOpenAIProvider("test-api-key")

	assert.NotNil(t, provider)
	assert.Equal(t, "test-api-key", provider.apiKey)
	assert.Equal(t, OpenAIBaseURL, provider.baseURL)
	assert.NotNil(t, provider.httpClient)
}

func TestOpenAIProvider_Name(t *testing.T) {
	provider := NewOpenAIProvider("test-api-key")
	assert.Equal(t, "openai", provider.Name())
}

func TestOpenAIProvider_ListVoices(t *testing.T) {
	provider := NewOpenAIProvider("test-api-key")
	ctx := context.Background()

	voices, err := provider.ListVoices(ctx)
	assert.NoError(t, err)
	assert.Len(t, voices, 6)

	// Check all expected voices
	voiceIDs := make(map[string]bool)
	for _, v := range voices {
		voiceIDs[v.ID] = true
	}

	assert.True(t, voiceIDs["alloy"])
	assert.True(t, voiceIDs["echo"])
	assert.True(t, voiceIDs["fable"])
	assert.True(t, voiceIDs["onyx"])
	assert.True(t, voiceIDs["nova"])
	assert.True(t, voiceIDs["shimmer"])
}

func TestOpenAIProvider_Synthesize(t *testing.T) {
	t.Run("returns error for empty text", func(t *testing.T) {
		provider := NewOpenAIProvider("test-api-key")
		ctx := context.Background()

		_, err := provider.Synthesize(ctx, "", SynthesizeOptions{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "text cannot be empty")
	})

	t.Run("successful synthesis", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			assert.Equal(t, "POST", r.Method)
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer test-api-key")
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			// Return mock audio data
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("mock audio data"))
		}))
		defer server.Close()

		provider := NewOpenAIProvider("test-api-key")
		provider.baseURL = server.URL

		ctx := context.Background()
		reader, err := provider.Synthesize(ctx, "Hello world", SynthesizeOptions{
			Voice:  "alloy",
			Model:  "tts-1",
			Format: "mp3",
			Speed:  1.0,
		})

		assert.NoError(t, err)
		assert.NotNil(t, reader)

		data, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, "mock audio data", string(data))
		reader.Close()
	})

	t.Run("uses defaults when options not provided", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("audio"))
		}))
		defer server.Close()

		provider := NewOpenAIProvider("test-api-key")
		provider.baseURL = server.URL

		ctx := context.Background()
		reader, err := provider.Synthesize(ctx, "Test", SynthesizeOptions{})

		assert.NoError(t, err)
		assert.NotNil(t, reader)
		reader.Close()
	})

	t.Run("clamps speed to valid range", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("audio"))
		}))
		defer server.Close()

		provider := NewOpenAIProvider("test-api-key")
		provider.baseURL = server.URL

		ctx := context.Background()

		// Test speed below minimum
		reader, err := provider.Synthesize(ctx, "Test", SynthesizeOptions{Speed: 0.1})
		assert.NoError(t, err)
		reader.Close()

		// Test speed above maximum
		reader, err = provider.Synthesize(ctx, "Test", SynthesizeOptions{Speed: 10.0})
		assert.NoError(t, err)
		reader.Close()
	})

	t.Run("handles API error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
		}))
		defer server.Close()

		provider := NewOpenAIProvider("test-api-key")
		provider.baseURL = server.URL

		ctx := context.Background()
		_, err := provider.Synthesize(ctx, "Test", SynthesizeOptions{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "OpenAI API error")
	})
}

func TestOpenAIProvider_IsAvailable(t *testing.T) {
	t.Run("returns false with empty API key", func(t *testing.T) {
		provider := NewOpenAIProvider("")
		ctx := context.Background()

		assert.False(t, provider.IsAvailable(ctx))
	})

	t.Run("returns true when API responds OK", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		provider := NewOpenAIProvider("test-api-key")
		provider.baseURL = server.URL

		ctx := context.Background()
		assert.True(t, provider.IsAvailable(ctx))
	})

	t.Run("returns false when API responds with error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		provider := NewOpenAIProvider("test-api-key")
		provider.baseURL = server.URL

		ctx := context.Background()
		assert.False(t, provider.IsAvailable(ctx))
	})
}

func TestOpenAIProviderFromConfig(t *testing.T) {
	t.Run("fails without API key", func(t *testing.T) {
		config := map[string]interface{}{}
		_, err := OpenAIProviderFromConfig(config)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "api_key is required")
	})

	t.Run("creates provider with API key", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key": "test-key",
		}
		provider, err := OpenAIProviderFromConfig(config)

		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "test-key", provider.apiKey)
		assert.Equal(t, OpenAIBaseURL, provider.baseURL)
	})

	t.Run("applies custom base URL", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key":  "test-key",
			"base_url": "https://custom.openai.com/v1/",
		}
		provider, err := OpenAIProviderFromConfig(config)

		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "https://custom.openai.com/v1", provider.baseURL)
	})
}

func TestOpenAIError_String(t *testing.T) {
	err := OpenAIError{}
	err.Error.Message = "Test error message"
	err.Error.Type = "invalid_request_error"
	err.Error.Code = "invalid_api_key"

	result := err.String()

	assert.Contains(t, result, "Test error message")
	assert.Contains(t, result, "invalid_request_error")
	assert.Contains(t, result, "invalid_api_key")
}

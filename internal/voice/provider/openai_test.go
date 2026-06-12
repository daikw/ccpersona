package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
			_, _ = w.Write([]byte("mock audio data"))
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
		_ = reader.Close()
	})

	t.Run("uses defaults when options not provided", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("audio"))
		}))
		defer server.Close()

		provider := NewOpenAIProvider("test-api-key")
		provider.baseURL = server.URL

		ctx := context.Background()
		reader, err := provider.Synthesize(ctx, "Test", SynthesizeOptions{})

		assert.NoError(t, err)
		assert.NotNil(t, reader)
		_ = reader.Close()
	})

	t.Run("clamps speed to valid range", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("audio"))
		}))
		defer server.Close()

		provider := NewOpenAIProvider("test-api-key")
		provider.baseURL = server.URL

		ctx := context.Background()

		// Test speed below minimum
		reader, err := provider.Synthesize(ctx, "Test", SynthesizeOptions{Speed: 0.1})
		assert.NoError(t, err)
		_ = reader.Close()

		// Test speed above maximum
		reader, err = provider.Synthesize(ctx, "Test", SynthesizeOptions{Speed: 10.0})
		assert.NoError(t, err)
		_ = reader.Close()
	})

	t.Run("handles API error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
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

	t.Run("uses non-billed GET /models for the check", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Must not hit the billed synthesis endpoint.
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, OpenAIModelsEndpoint, r.URL.Path)
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer test-api-key")
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

	t.Run("probes {base_url}/models for local server without API key", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, OpenAIModelsEndpoint, r.URL.Path)
			// No auth header expected for keyless local servers.
			assert.Empty(t, r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		provider, err := OpenAIProviderFromConfig(map[string]interface{}{
			"base_url": server.URL + "/",
		})
		assert.NoError(t, err)

		ctx := context.Background()
		assert.True(t, provider.IsAvailable(ctx))
	})

	t.Run("returns false when local server /models is unavailable (503)", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		provider, err := OpenAIProviderFromConfig(map[string]interface{}{
			"base_url": server.URL,
		})
		assert.NoError(t, err)

		ctx := context.Background()
		assert.False(t, provider.IsAvailable(ctx))
	})
}

func TestOpenAIProvider_SynthesizeReflectsConfig(t *testing.T) {
	t.Run("uses config model/voice and omits auth for local server", func(t *testing.T) {
		var gotBody map[string]interface{}
		var gotAuth string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("audio"))
		}))
		defer server.Close()

		provider, err := OpenAIProviderFromConfig(map[string]interface{}{
			"base_url": server.URL,
		})
		assert.NoError(t, err)

		ctx := context.Background()
		reader, err := provider.Synthesize(ctx, "こんにちは", SynthesizeOptions{
			Model: "irodori-tts",
			Voice: "none",
		})
		assert.NoError(t, err)
		_ = reader.Close()

		assert.Equal(t, "irodori-tts", gotBody["model"])
		assert.Equal(t, "none", gotBody["voice"])
		assert.Empty(t, gotAuth, "no Authorization header for keyless local server")
	})

	t.Run("timeout_seconds is applied to the HTTP client", func(t *testing.T) {
		provider, err := OpenAIProviderFromConfig(map[string]interface{}{
			"base_url":        "http://localhost:8088/v1",
			"timeout_seconds": 120,
		})
		assert.NoError(t, err)
		assert.Equal(t, 120*time.Second, provider.httpClient.Timeout)
	})

	t.Run("timeout defaults to 30s when unset", func(t *testing.T) {
		provider, err := OpenAIProviderFromConfig(map[string]interface{}{
			"api_key": "test-key",
		})
		assert.NoError(t, err)
		assert.Equal(t, 30*time.Second, provider.httpClient.Timeout)
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

	t.Run("succeeds with base_url and no API key", func(t *testing.T) {
		config := map[string]interface{}{
			"base_url": "http://localhost:8088/v1",
		}
		provider, err := OpenAIProviderFromConfig(config)

		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Empty(t, provider.apiKey)
		assert.Equal(t, "http://localhost:8088/v1", provider.baseURL)
	})

	t.Run("coerces JSON-decoded float64 timeout_seconds", func(t *testing.T) {
		// encoding/json decodes numbers as float64; ensure that path works too.
		config := map[string]interface{}{
			"base_url":        "http://localhost:8088/v1",
			"timeout_seconds": float64(90),
		}
		provider, err := OpenAIProviderFromConfig(config)

		assert.NoError(t, err)
		assert.Equal(t, 90*time.Second, provider.httpClient.Timeout)
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

package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewElevenLabsProvider(t *testing.T) {
	provider := NewElevenLabsProvider("test-api-key")

	assert.NotNil(t, provider)
	assert.Equal(t, "test-api-key", provider.apiKey)
	assert.Equal(t, ElevenLabsBaseURL, provider.baseURL)
	assert.NotNil(t, provider.httpClient)
}

func TestElevenLabsProvider_Name(t *testing.T) {
	provider := NewElevenLabsProvider("test-api-key")
	assert.Equal(t, "elevenlabs", provider.Name())
}

func TestElevenLabsProvider_ListVoices(t *testing.T) {
	t.Run("successful voice listing", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "test-api-key", r.Header.Get("xi-api-key"))

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"voices": [
					{
						"voice_id": "voice1",
						"name": "Test Voice",
						"category": "premade",
						"description": "A test voice",
						"labels": {"language": "en"},
						"available_for_tts": true
					}
				]
			}`))
		}))
		defer server.Close()

		provider := NewElevenLabsProvider("test-api-key")
		provider.baseURL = server.URL

		ctx := context.Background()
		voices, err := provider.ListVoices(ctx)

		assert.NoError(t, err)
		assert.Len(t, voices, 1)
		assert.Equal(t, "voice1", voices[0].ID)
		assert.Equal(t, "Test Voice", voices[0].Name)
	})

	t.Run("handles API error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"detail": {"status": "invalid_api_key"}}`))
		}))
		defer server.Close()

		provider := NewElevenLabsProvider("test-api-key")
		provider.baseURL = server.URL

		ctx := context.Background()
		_, err := provider.ListVoices(ctx)

		assert.Error(t, err)
	})
}

func TestElevenLabsProvider_Synthesize(t *testing.T) {
	t.Run("returns error for empty text", func(t *testing.T) {
		provider := NewElevenLabsProvider("test-api-key")
		ctx := context.Background()

		_, err := provider.Synthesize(ctx, "", SynthesizeOptions{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "text cannot be empty")
	})

	t.Run("successful synthesis", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Contains(t, r.URL.Path, "/text-to-speech/")
			assert.Equal(t, "test-api-key", r.Header.Get("xi-api-key"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("mock audio data"))
		}))
		defer server.Close()

		provider := NewElevenLabsProvider("test-api-key")
		provider.baseURL = server.URL

		ctx := context.Background()
		reader, err := provider.Synthesize(ctx, "Hello world", SynthesizeOptions{
			Voice:           "test-voice-id",
			Model:           "eleven_multilingual_v2",
			Format:          "mp3_44100_128",
			Stability:       0.5,
			SimilarityBoost: 0.5,
		})

		assert.NoError(t, err)
		assert.NotNil(t, reader)

		data, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, "mock audio data", string(data))
		reader.Close()
	})

	t.Run("uses default values", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("audio"))
		}))
		defer server.Close()

		provider := NewElevenLabsProvider("test-api-key")
		provider.baseURL = server.URL

		ctx := context.Background()
		reader, err := provider.Synthesize(ctx, "Test", SynthesizeOptions{})

		assert.NoError(t, err)
		assert.NotNil(t, reader)
		reader.Close()
	})

	t.Run("handles API error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"detail": {"message": "Bad request"}}`))
		}))
		defer server.Close()

		provider := NewElevenLabsProvider("test-api-key")
		provider.baseURL = server.URL

		ctx := context.Background()
		_, err := provider.Synthesize(ctx, "Test", SynthesizeOptions{})

		assert.Error(t, err)
	})
}

func TestElevenLabsProvider_IsAvailable(t *testing.T) {
	t.Run("returns false with empty API key", func(t *testing.T) {
		provider := NewElevenLabsProvider("")
		ctx := context.Background()

		assert.False(t, provider.IsAvailable(ctx))
	})

	t.Run("returns true when API responds OK", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"voices": []}`))
		}))
		defer server.Close()

		provider := NewElevenLabsProvider("test-api-key")
		provider.baseURL = server.URL

		ctx := context.Background()
		assert.True(t, provider.IsAvailable(ctx))
	})

	t.Run("returns false when API responds with error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		provider := NewElevenLabsProvider("test-api-key")
		provider.baseURL = server.URL

		ctx := context.Background()
		assert.False(t, provider.IsAvailable(ctx))
	})
}

func TestElevenLabsProviderFromConfig(t *testing.T) {
	t.Run("fails without API key", func(t *testing.T) {
		config := map[string]interface{}{}
		_, err := ElevenLabsProviderFromConfig(config)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "api_key is required")
	})

	t.Run("creates provider with API key", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key": "test-key",
		}
		provider, err := ElevenLabsProviderFromConfig(config)

		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "test-key", provider.apiKey)
	})

	t.Run("applies custom base URL", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key":  "test-key",
			"base_url": "https://custom.elevenlabs.io/v1/",
		}
		provider, err := ElevenLabsProviderFromConfig(config)

		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "https://custom.elevenlabs.io/v1", provider.baseURL)
	})
}

func TestConvertToElevenLabsFormat(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"mp3", "mp3_44100_128"},
		{"wav", "pcm_44100"},
		{"ogg", "ulaw_8000"},     // maps to ulaw_8000
		{"", "mp3_44100_128"},   // default
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := convertToElevenLabsFormat(tt.format)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestElevenLabsError_String(t *testing.T) {
	t.Run("with string detail", func(t *testing.T) {
		err := ElevenLabsError{
			Detail: "Test error message",
		}

		result := err.String()
		assert.Contains(t, result, "Test error message")
	})

	t.Run("with map detail", func(t *testing.T) {
		err := ElevenLabsError{
			Detail: map[string]interface{}{
				"message": "Error from map",
				"status":  "error_status",
			},
		}

		result := err.String()
		assert.Contains(t, result, "Error from map")
	})

	t.Run("with nil detail", func(t *testing.T) {
		err := ElevenLabsError{
			Detail: nil,
		}

		result := err.String()
		assert.Contains(t, result, "ElevenLabs API Error")
	})
}

func TestGetPrebuiltVoices(t *testing.T) {
	voices := GetPrebuiltVoices()

	assert.NotEmpty(t, voices)

	// Check that each voice has required fields
	for _, v := range voices {
		assert.NotEmpty(t, v.ID)
		assert.NotEmpty(t, v.Name)
		assert.NotEmpty(t, v.Description)
	}

	// Check for some known prebuilt voices
	voiceNames := make(map[string]bool)
	for _, v := range voices {
		voiceNames[v.Name] = true
	}

	assert.True(t, voiceNames["Rachel"])
	assert.True(t, voiceNames["Adam"])
}

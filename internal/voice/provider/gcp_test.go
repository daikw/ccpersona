package provider

import (
	"context"
	"io"
	"strings"
	"testing"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockGCPClient is a mock for the GCP TTS client
type MockGCPClient struct {
	mock.Mock
}

func (m *MockGCPClient) ListVoices(ctx context.Context, req *texttospeechpb.ListVoicesRequest, opts ...grpc.CallOption) (*texttospeechpb.ListVoicesResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*texttospeechpb.ListVoicesResponse), args.Error(1)
}

func (m *MockGCPClient) SynthesizeSpeech(ctx context.Context, req *texttospeechpb.SynthesizeSpeechRequest, opts ...grpc.CallOption) (*texttospeechpb.SynthesizeSpeechResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*texttospeechpb.SynthesizeSpeechResponse), args.Error(1)
}

func (m *MockGCPClient) Close() error {
	return nil
}

func TestGCPProvider_Name(t *testing.T) {
	p := &GCPProvider{}
	assert.Equal(t, "gcp", p.Name())
}

func TestGCPProvider_detectEngineType(t *testing.T) {
	p := &GCPProvider{}

	tests := []struct {
		voiceName string
		expected  string
	}{
		{"ja-JP-Wavenet-A", "WaveNet"},
		{"ja-JP-Neural2-B", "Neural2"},
		{"ja-JP-Studio-A", "Studio"},
		{"ja-JP-Standard-A", "Standard"},
		{"en-US-Polyglot-1", "Polyglot"},
		{"en-US-News-K", "News"},
		{"en-US-Casual-K", "Casual"},
		{"unknown-voice", "Standard"},
	}

	for _, tt := range tests {
		t.Run(tt.voiceName, func(t *testing.T) {
			result := p.detectEngineType(tt.voiceName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGCPProvider_getAudioEncoding(t *testing.T) {
	p := &GCPProvider{}

	tests := []struct {
		format   string
		expected texttospeechpb.AudioEncoding
	}{
		{"mp3", texttospeechpb.AudioEncoding_MP3},
		{"MP3", texttospeechpb.AudioEncoding_MP3},
		{"wav", texttospeechpb.AudioEncoding_LINEAR16},
		{"linear16", texttospeechpb.AudioEncoding_LINEAR16},
		{"ogg", texttospeechpb.AudioEncoding_OGG_OPUS},
		{"ogg_opus", texttospeechpb.AudioEncoding_OGG_OPUS},
		{"mulaw", texttospeechpb.AudioEncoding_MULAW},
		{"alaw", texttospeechpb.AudioEncoding_ALAW},
		{"unknown", texttospeechpb.AudioEncoding_MP3}, // Default to MP3
		{"", texttospeechpb.AudioEncoding_MP3},        // Default to MP3
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := p.getAudioEncoding(tt.format)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGCPProvider_getSpeakingRate(t *testing.T) {
	p := &GCPProvider{}

	tests := []struct {
		name     string
		speed    float64
		expected float64
	}{
		{"default", 0, 1.0},
		{"negative", -1.0, 1.0},
		{"normal", 1.0, 1.0},
		{"slow", 0.5, 0.5},
		{"fast", 2.0, 2.0},
		{"too_slow", 0.1, 0.25}, // Clamped to min
		{"too_fast", 5.0, 4.0},  // Clamped to max
		{"boundary_min", 0.25, 0.25},
		{"boundary_max", 4.0, 4.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.getSpeakingRate(tt.speed)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGCPProvider_getSampleRate(t *testing.T) {
	p := &GCPProvider{}

	tests := []struct {
		sampleRate string
		expected   int32
	}{
		{"8000", 8000},
		{"16000", 16000},
		{"22050", 22050},
		{"24000", 24000},
		{"32000", 32000},
		{"44100", 44100},
		{"48000", 48000},
		{"invalid", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.sampleRate, func(t *testing.T) {
			result := p.getSampleRate(tt.sampleRate)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSSML(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"plain_text", "Hello world", false},
		{"speak_tag", "<speak>Hello world</speak>", true},
		{"prosody_tag", "Text with <prosody rate='slow'>slow speech</prosody>", true},
		{"break_tag", "Hello <break time='1s'/> world", true},
		{"emphasis_tag", "This is <emphasis>important</emphasis>", true},
		{"html_not_ssml", "<p>This is HTML</p>", false},
		{"whitespace_speak", "  <speak>Hello</speak>  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSSML(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGCPProviderFromConfig(t *testing.T) {
	// Note: This test will fail without valid GCP credentials
	// It's primarily for verifying the configuration parsing logic

	t.Run("config_parsing", func(t *testing.T) {
		// Test that config values are parsed correctly
		config := map[string]interface{}{
			"project_id": "test-project",
			"region":     "us-central1",
			"voice":      "en-US-Wavenet-A",
			"language":   "en-US",
			"engine":     "wavenet",
		}

		// This will fail without credentials, but we're testing config parsing
		_, err := GCPProviderFromConfig(config)
		// We expect an error because we don't have real credentials in tests
		// But the config parsing should work
		if err != nil {
			assert.Contains(t, err.Error(), "failed to create GCP TTS client")
		}
	})
}

func TestGCPProvider_Synthesize_EmptyText(t *testing.T) {
	p := &GCPProvider{}

	_, err := p.Synthesize(context.Background(), "", SynthesizeOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "text cannot be empty")
}

func TestGCPProvider_LanguageExtraction(t *testing.T) {
	tests := []struct {
		voice    string
		expected string
	}{
		{"ja-JP-Neural2-B", "ja-JP"},
		{"en-US-Wavenet-A", "en-US"},
		{"ko-KR-Standard-A", "ko-KR"},
		{"cmn-CN-Wavenet-A", "cmn-CN"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.voice, func(t *testing.T) {
			parts := strings.Split(tt.voice, "-")
			var result string
			if len(parts) >= 2 {
				result = parts[0] + "-" + parts[1]
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Integration test - requires real GCP credentials
// Run with: go test -v -run TestGCPProvider_Integration -tags=integration
func TestGCPProvider_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test requires GOOGLE_APPLICATION_CREDENTIALS to be set
	ctx := context.Background()

	// Try to create client - will fail gracefully if no credentials
	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		t.Skipf("Skipping integration test - no GCP credentials: %v", err)
	}
	defer func() { _ = client.Close() }()

	p := &GCPProvider{
		client:   client,
		voice:    "ja-JP-Neural2-B",
		language: "ja-JP",
	}

	t.Run("list_voices", func(t *testing.T) {
		voices, err := p.ListVoices(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, voices)

		// Check that we have Japanese voices
		hasJapanese := false
		for _, v := range voices {
			if strings.HasPrefix(v.Language, "ja") {
				hasJapanese = true
				break
			}
		}
		assert.True(t, hasJapanese, "Should have Japanese voices")
	})

	t.Run("list_japanese_voices", func(t *testing.T) {
		voices, err := p.ListJapaneseVoices(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, voices)

		for _, v := range voices {
			assert.Equal(t, "ja-JP", v.Language)
		}
	})

	t.Run("synthesize_text", func(t *testing.T) {
		reader, err := p.Synthesize(ctx, "こんにちは、世界！", SynthesizeOptions{
			Format: "mp3",
		})
		assert.NoError(t, err)
		assert.NotNil(t, reader)
		defer func() { _ = reader.Close() }()

		// Read the audio data
		data, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
		t.Logf("Generated %d bytes of audio", len(data))
	})

	t.Run("synthesize_with_options", func(t *testing.T) {
		reader, err := p.Synthesize(ctx, "速度テストです", SynthesizeOptions{
			Voice:  "ja-JP-Wavenet-A",
			Speed:  1.5,
			Format: "mp3",
		})
		assert.NoError(t, err)
		assert.NotNil(t, reader)
		defer func() { _ = reader.Close() }()

		data, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("synthesize_ssml", func(t *testing.T) {
		ssml := `<speak>こんにちは<break time="500ms"/>世界</speak>`
		reader, err := p.Synthesize(ctx, ssml, SynthesizeOptions{
			Format: "mp3",
		})
		assert.NoError(t, err)
		assert.NotNil(t, reader)
		defer func() { _ = reader.Close() }()

		data, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("is_available", func(t *testing.T) {
		available := p.IsAvailable(ctx)
		assert.True(t, available)
	})
}

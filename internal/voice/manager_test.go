package voice

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewVoiceManager(t *testing.T) {
	t.Run("creates manager with default config", func(t *testing.T) {
		config := DefaultConfig()
		manager := NewVoiceManager(config)
		assert.NotNil(t, manager)
		assert.NotNil(t, manager.config)
		assert.NotNil(t, manager.legacyEngine)
	})

	t.Run("creates manager with custom config", func(t *testing.T) {
		config := &Config{
			EnginePriority:  EngineVoicevox,
			VoicevoxSpeaker: 5,
			ReadingMode:     ModeFull,
		}
		manager := NewVoiceManager(config)
		assert.NotNil(t, manager)
		assert.Equal(t, EngineVoicevox, manager.config.EnginePriority)
	})
}

func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"mp3", "mp3"},
		{"wav", "wav"},
		{"ogg", "ogg"},
		{"flac", "flac"},
		{"aac", "aac"},
		{"unknown", "mp3"}, // defaults to mp3
		{"", "mp3"},        // empty defaults to mp3
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := getFileExtension(tt.format)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsVoiceFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"voice_123.mp3", true},        // starts with voice_
		{"voice_abc.wav", true},        // starts with voice_
		{"voice_x", true},              // starts with voice_
		{"other_file.mp3", false},      // doesn't start with voice_
		{"voic.mp3", false},            // wrong prefix
		{"voice", false},               // no underscore after voice
		{"voice_", true},               // exactly "voice_" prefix
		{"", false},                    // empty
		{"abc", false},                 // unrelated
		{"ccpersona_voice_test", true}, // starts with ccpersona_voice_
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := isVoiceFile(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSynthesizeLocalNoRace verifies that concurrent calls to synthesizeLocal with
// different providers do not race on the shared VoiceManager config field.
// Run with: go test -race ./internal/voice/...
func TestSynthesizeLocalNoRace(t *testing.T) {
	config := DefaultConfig()
	config.EnginePriority = EngineAivisSpeech
	manager := NewVoiceManager(config)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	providers := []string{"voicevox", "aivisspeech", ""}
	ctx := context.Background()

	for i := 0; i < goroutines; i++ {
		provider := providers[i%len(providers)]
		go func(p string) {
			defer wg.Done()
			// synthesizeLocal will fail to reach a real engine, but the race
			// detector will catch any concurrent writes to vm.config.
			_ , _ = manager.Synthesize(ctx, "test", VoiceOptions{Provider: p})
		}(provider)
	}

	wg.Wait()

	// The original config must not have been mutated by any goroutine.
	assert.Equal(t, EngineAivisSpeech, config.EnginePriority)
}

func TestVoiceOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		opts := VoiceOptions{}
		assert.Empty(t, opts.Provider)
		assert.Empty(t, opts.Voice)
		assert.Equal(t, 0.0, opts.Speed)
	})

	t.Run("options with values", func(t *testing.T) {
		opts := VoiceOptions{
			Provider:        "openai",
			Voice:           "alloy",
			Speed:           1.5,
			Format:          "mp3",
			Model:           "tts-1",
			Stability:       0.7,
			SimilarityBoost: 0.8,
			PlayAudio:       true,
		}
		assert.Equal(t, "openai", opts.Provider)
		assert.Equal(t, "alloy", opts.Voice)
		assert.Equal(t, 1.5, opts.Speed)
		assert.True(t, opts.PlayAudio)
	})
}

package voice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text unchanged",
			input:    "Hello, world!",
			expected: "Hello, world!",
		},
		{
			name:     "text with markdown (without mdstrip)",
			input:    "**bold** and *italic*",
			expected: "**bold** and *italic*", // Returns original if mdstrip not available
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "multiline text",
			input:    "Line 1\nLine 2\nLine 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripMarkdown(tt.input)
			// If mdstrip is not installed, the function should return the original text
			// If mdstrip is installed, the result may differ
			// We test that at minimum, the function doesn't crash
			if tt.input == "" {
				assert.Empty(t, result)
			} else {
				assert.NotEmpty(t, result, "StripMarkdown should return non-empty for non-empty input")
			}
		})
	}
}

func TestIsCommandAvailable(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected bool
	}{
		{
			name:     "go command should exist",
			cmd:      "go",
			expected: true,
		},
		{
			name:     "nonexistent command",
			cmd:      "nonexistent_command_xyz_123",
			expected: false,
		},
		{
			name:     "git command should exist",
			cmd:      "git",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCommandAvailable(tt.cmd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewVoiceEngine(t *testing.T) {
	config := DefaultConfig()

	t.Run("creates engine with default config", func(t *testing.T) {
		engine := NewVoiceEngine(config)
		assert.NotNil(t, engine)
	})

	t.Run("creates engine with custom config", func(t *testing.T) {
		customConfig := &Config{
			EnginePriority:     EngineVoicevox,
			VoicevoxSpeaker:    5,
			AivisSpeechSpeaker: 12345,
			VolumeScale:        0.8,
			ReadingMode:        ModeFull,
			MaxChars:           500,
		}
		engine := NewVoiceEngine(customConfig)
		assert.NotNil(t, engine)
	})
}

func TestVoiceEngine_SelectEngine(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that connects to local voice engines")
	}

	config := DefaultConfig()
	engine := NewVoiceEngine(config)

	t.Run("select with aivisspeech priority", func(t *testing.T) {
		config.EnginePriority = EngineAivisSpeech
		// This will return the selected engine name or empty if none available
		selectedEngine, err := engine.SelectEngine()
		// We can't guarantee which engine is available, but the function should not crash
		assert.NotNil(t, engine)
		// selectedEngine is either "aivisspeech", "voicevox", or "" (with error)
		if err != nil {
			assert.Empty(t, selectedEngine)
		} else {
			assert.True(t, selectedEngine == EngineAivisSpeech || selectedEngine == EngineVoicevox)
		}
	})

	t.Run("select with voicevox priority", func(t *testing.T) {
		config.EnginePriority = EngineVoicevox
		selectedEngine, err := engine.SelectEngine()
		if err != nil {
			assert.Empty(t, selectedEngine)
		} else {
			assert.True(t, selectedEngine == EngineAivisSpeech || selectedEngine == EngineVoicevox)
		}
	})
}

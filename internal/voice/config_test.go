package voice

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandEnvVars(t *testing.T) {
	// Set test environment variable
	os.Setenv("TEST_API_KEY", "sk-test-12345")
	defer os.Unsetenv("TEST_API_KEY")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "expand ${VAR} pattern",
			input:    `{"apiKey": "${TEST_API_KEY}"}`,
			expected: `{"apiKey": "sk-test-12345"}`,
		},
		{
			name:     "missing env var returns empty",
			input:    `{"apiKey": "${NONEXISTENT_VAR}"}`,
			expected: `{"apiKey": ""}`,
		},
		{
			name:     "no variables to expand",
			input:    `{"apiKey": "literal-value"}`,
			expected: `{"apiKey": "literal-value"}`,
		},
		{
			name:     "multiple variables",
			input:    `{"key1": "${TEST_API_KEY}", "key2": "${TEST_API_KEY}"}`,
			expected: `{"key1": "sk-test-12345", "key2": "sk-test-12345"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandEnvVars(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVoiceConfigLoader_LoadConfig(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "voice-config-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create .claude directory
	claudeDir := filepath.Join(tmpDir, ".claude")
	err = os.MkdirAll(claudeDir, 0755)
	require.NoError(t, err)

	t.Run("load project config", func(t *testing.T) {
		// Create project config
		configContent := `{
			"defaultProvider": "openai",
			"providers": {
				"openai": {
					"apiKey": "test-key",
					"voice": "nova"
				}
			}
		}`
		configPath := filepath.Join(claudeDir, "voice.json")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		loader := NewVoiceConfigLoader()
		config, err := loader.LoadConfig(tmpDir)

		require.NoError(t, err)
		require.NotNil(t, config)
		assert.Equal(t, "openai", config.DefaultProvider)
		assert.Equal(t, "test-key", config.Providers["openai"].APIKey)
		assert.Equal(t, "nova", config.Providers["openai"].Voice)
	})

	t.Run("no config returns nil", func(t *testing.T) {
		emptyDir, err := os.MkdirTemp("", "empty-*")
		require.NoError(t, err)
		defer os.RemoveAll(emptyDir)

		loader := NewVoiceConfigLoader()
		config, err := loader.LoadConfig(emptyDir)

		require.NoError(t, err)
		assert.Nil(t, config)
	})
}

func TestVoiceConfigFile_GetProviderConfig(t *testing.T) {
	config := &VoiceConfigFile{
		DefaultProvider: "openai",
		Providers: map[string]ProviderConfig{
			"openai": {
				APIKey: "test-key",
				Voice:  "nova",
			},
		},
	}

	t.Run("existing provider", func(t *testing.T) {
		providerConfig := config.GetProviderConfig("openai")
		require.NotNil(t, providerConfig)
		assert.Equal(t, "test-key", providerConfig.APIKey)
	})

	t.Run("non-existing provider", func(t *testing.T) {
		providerConfig := config.GetProviderConfig("nonexistent")
		assert.Nil(t, providerConfig)
	})

	t.Run("nil config", func(t *testing.T) {
		var nilConfig *VoiceConfigFile
		providerConfig := nilConfig.GetProviderConfig("openai")
		assert.Nil(t, providerConfig)
	})
}

func TestVoiceConfigFile_GetEffectiveProvider(t *testing.T) {
	config := &VoiceConfigFile{
		DefaultProvider: "openai",
	}

	t.Run("explicit provider takes priority", func(t *testing.T) {
		result := config.GetEffectiveProvider("elevenlabs")
		assert.Equal(t, "elevenlabs", result)
	})

	t.Run("default provider when no explicit", func(t *testing.T) {
		result := config.GetEffectiveProvider("")
		assert.Equal(t, "openai", result)
	})

	t.Run("empty when no config and no explicit", func(t *testing.T) {
		var nilConfig *VoiceConfigFile
		result := nilConfig.GetEffectiveProvider("")
		assert.Equal(t, "", result)
	})
}

func TestVoiceConfigFile_Validate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := &VoiceConfigFile{
			DefaultProvider: "aivisspeech",
			Providers: map[string]ProviderConfig{
				"aivisspeech": {
					Speaker: 888753760,
					Volume:  1.0,
				},
			},
		}
		errors := config.Validate()
		assert.Empty(t, errors)
	})

	t.Run("missing openai api key", func(t *testing.T) {
		config := &VoiceConfigFile{
			Providers: map[string]ProviderConfig{
				"openai": {
					Voice: "nova",
				},
			},
		}
		errors := config.Validate()
		assert.Len(t, errors, 1)
		assert.Contains(t, errors[0], "apiKey is required")
	})

	t.Run("invalid speed", func(t *testing.T) {
		config := &VoiceConfigFile{
			Providers: map[string]ProviderConfig{
				"aivisspeech": {
					Speed: 10.0, // Invalid: > 4.0
				},
			},
		}
		errors := config.Validate()
		assert.Len(t, errors, 1)
		assert.Contains(t, errors[0], "speed must be between")
	})

	t.Run("invalid volume", func(t *testing.T) {
		config := &VoiceConfigFile{
			Providers: map[string]ProviderConfig{
				"aivisspeech": {
					Volume: 5.0, // Invalid: > 2.0
				},
			},
		}
		errors := config.Validate()
		assert.Len(t, errors, 1)
		assert.Contains(t, errors[0], "volume must be between")
	})
}

func TestVoiceConfigFile_MaskSecrets(t *testing.T) {
	config := &VoiceConfigFile{
		DefaultProvider: "openai",
		Providers: map[string]ProviderConfig{
			"openai": {
				APIKey: "sk-1234567890abcdef",
				Voice:  "nova",
			},
		},
	}

	masked := config.MaskSecrets()

	require.NotNil(t, masked)
	assert.Equal(t, "openai", masked.DefaultProvider)
	assert.NotEqual(t, config.Providers["openai"].APIKey, masked.Providers["openai"].APIKey)
	// New format shows "[set, N chars]" instead of revealing any characters
	assert.Contains(t, masked.Providers["openai"].APIKey, "[set,")
	assert.Contains(t, masked.Providers["openai"].APIKey, "chars]")
	assert.Equal(t, "nova", masked.Providers["openai"].Voice)
}

func TestValidateConfigPath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		expectErr bool
	}{
		{
			name:      "valid path",
			path:      "/home/user/.claude/voice.json",
			expectErr: false,
		},
		{
			name:      "valid relative path",
			path:      ".claude/voice.json",
			expectErr: false,
		},
		{
			name:      "path traversal attempt",
			path:      "/home/user/../../../etc/passwd",
			expectErr: true,
		},
		{
			name:      "wrong filename",
			path:      "/home/user/.claude/config.json",
			expectErr: true,
		},
		{
			name:      "hidden traversal",
			path:      "/home/user/.claude/../../voice.json",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfigPath(tt.path)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateExampleConfig(t *testing.T) {
	example := GenerateExampleConfig()

	assert.Contains(t, example, "defaultProvider")
	assert.Contains(t, example, "openai")
	assert.Contains(t, example, "elevenlabs")
	assert.Contains(t, example, "polly")
	assert.Contains(t, example, "voicevox")
	assert.Contains(t, example, "aivisspeech")
	assert.Contains(t, example, "${OPENAI_API_KEY}")
}

func TestVoiceConfigLoader_LoadFromPath(t *testing.T) {
	// Create temp directory with config file
	tmpDir, err := os.MkdirTemp("", "voice-config-loadfrompath-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create .claude directory
	claudeDir := filepath.Join(tmpDir, ".claude")
	err = os.MkdirAll(claudeDir, 0755)
	require.NoError(t, err)

	configContent := `{
		"defaultProvider": "aivisspeech",
		"providers": {
			"aivisspeech": {
				"speaker": 888753760
			}
		}
	}`
	configPath := filepath.Join(claudeDir, "voice.json")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	loader := NewVoiceConfigLoader()

	t.Run("load from valid path", func(t *testing.T) {
		config, err := loader.LoadFromPath(configPath)
		require.NoError(t, err)
		require.NotNil(t, config)
		assert.Equal(t, "aivisspeech", config.DefaultProvider)
		assert.Equal(t, 888753760, config.Providers["aivisspeech"].Speaker)
	})

	t.Run("load from path with traversal", func(t *testing.T) {
		_, err := loader.LoadFromPath("/tmp/../etc/passwd")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "path traversal")
	})

	t.Run("load from wrong filename", func(t *testing.T) {
		_, err := loader.LoadFromPath("/tmp/config.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "voice.json")
	})

	t.Run("load from nonexistent path", func(t *testing.T) {
		_, err := loader.LoadFromPath("/nonexistent/path/voice.json")
		assert.Error(t, err)
	})
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "item exists in slice",
			slice:    []string{"a", "b", "c"},
			item:     "b",
			expected: true,
		},
		{
			name:     "item not in slice",
			slice:    []string{"a", "b", "c"},
			item:     "d",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			item:     "a",
			expected: false,
		},
		{
			name:     "item at beginning",
			slice:    []string{"x", "y", "z"},
			item:     "x",
			expected: true,
		},
		{
			name:     "item at end",
			slice:    []string{"x", "y", "z"},
			item:     "z",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVoiceConfigFile_ValidateProviderConfig_ElevenLabs(t *testing.T) {
	t.Run("missing elevenlabs api key", func(t *testing.T) {
		config := &VoiceConfigFile{
			Providers: map[string]ProviderConfig{
				"elevenlabs": {
					Voice: "Rachel",
				},
			},
		}
		errors := config.Validate()
		assert.Len(t, errors, 1)
		assert.Contains(t, errors[0], "apiKey is required")
	})

	t.Run("valid elevenlabs config", func(t *testing.T) {
		config := &VoiceConfigFile{
			Providers: map[string]ProviderConfig{
				"elevenlabs": {
					APIKey: "test-key",
					Voice:  "Rachel",
				},
			},
		}
		errors := config.Validate()
		assert.Empty(t, errors)
	})
}

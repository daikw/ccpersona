package persona

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "ccpersona-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Test case 1: No config file exists
	t.Run("NoConfigFile", func(t *testing.T) {
		config, err := LoadConfig(tmpDir)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if config != nil {
			t.Errorf("Expected nil config, got %v", config)
		}
	})

	// Test case 2: Valid config file
	t.Run("ValidConfigFile", func(t *testing.T) {
		// Create .claude directory
		claudeDir := filepath.Join(tmpDir, ClaudeDir)
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Create test config
		testConfig := &Config{
			Name: "test-persona",
			Voice: &VoiceConfig{
				Engine:    "voicevox",
				SpeakerID: 3,
			},
			OverrideGlobal:     true,
			CustomInstructions: "Test instructions",
		}

		// Write config file
		configPath := filepath.Join(claudeDir, ConfigFileName)
		data, err := json.MarshalIndent(testConfig, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			t.Fatal(err)
		}

		// Load config
		config, err := LoadConfig(tmpDir)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if config == nil {
			t.Fatal("Expected config, got nil")
		}
		if config.Name != testConfig.Name { //nolint:staticcheck // checked for nil above
			t.Errorf("Expected name %s, got %s", testConfig.Name, config.Name)
		}
		if config.Voice == nil || config.Voice.Engine != testConfig.Voice.Engine {
			t.Errorf("Voice config mismatch")
		}
	})

	// Test case 3: Invalid JSON
	t.Run("InvalidJSON", func(t *testing.T) {
		claudeDir := filepath.Join(tmpDir, ClaudeDir)
		configPath := filepath.Join(claudeDir, ConfigFileName)

		// Write invalid JSON
		if err := os.WriteFile(configPath, []byte("invalid json"), 0644); err != nil {
			t.Fatal(err)
		}

		// Try to load config
		config, err := LoadConfig(tmpDir)
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
		if config != nil {
			t.Error("Expected nil config for invalid JSON")
		}
	})
}

func TestSaveConfig(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "ccpersona-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	t.Run("SaveNewConfig", func(t *testing.T) {
		config := &Config{
			Name:           "save-test",
			OverrideGlobal: false,
		}

		// Save config
		if err := SaveConfig(tmpDir, config); err != nil {
			t.Errorf("Failed to save config: %v", err)
		}

		// Verify file exists
		configPath := filepath.Join(tmpDir, ClaudeDir, ConfigFileName)
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error("Config file was not created")
		}

		// Load and verify
		loaded, err := LoadConfig(tmpDir)
		if err != nil {
			t.Errorf("Failed to load saved config: %v", err)
		}
		if loaded.Name != config.Name {
			t.Errorf("Loaded config name mismatch: expected %s, got %s", config.Name, loaded.Name)
		}
	})
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "ValidConfig",
			config: &Config{
				Name: "valid",
			},
			wantErr: false,
		},
		{
			name: "EmptyName",
			config: &Config{
				Name: "",
			},
			wantErr: true,
		},
		{
			name: "ValidVoiceConfig",
			config: &Config{
				Name: "valid",
				Voice: &VoiceConfig{
					Engine:    "voicevox",
					SpeakerID: 1,
				},
			},
			wantErr: false,
		},
		{
			name: "InvalidVoiceEngine",
			config: &Config{
				Name: "valid",
				Voice: &VoiceConfig{
					Engine:    "invalid-engine",
					SpeakerID: 1,
				},
			},
			wantErr: true,
		},
		{
			name: "EmptyVoiceEngine",
			config: &Config{
				Name: "valid",
				Voice: &VoiceConfig{
					Engine:    "",
					SpeakerID: 1,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetDefaultConfig(t *testing.T) {
	config := GetDefaultConfig()

	if config == nil {
		t.Fatal("Expected default config, got nil")
	}

	if config.Name != "default" { //nolint:staticcheck // checked for nil above
		t.Errorf("Expected default name 'default', got %s", config.Name)
	}

	if config.OverrideGlobal != false {
		t.Error("Expected OverrideGlobal to be false")
	}
}

func TestLoadConfigWithFallback(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	t.Run("ProjectConfigExists", func(t *testing.T) {
		// Create temp project directory with config
		tmpDir, err := os.MkdirTemp("", "ccpersona-test-project-*")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = os.RemoveAll(tmpDir)
		}()

		// Create .claude directory and config
		claudeDir := filepath.Join(tmpDir, ClaudeDir)
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatal(err)
		}

		testConfig := &Config{Name: "project-persona"}
		data, _ := json.MarshalIndent(testConfig, "", "  ")
		if err := os.WriteFile(filepath.Join(claudeDir, ConfigFileName), data, 0644); err != nil {
			t.Fatal(err)
		}

		// Change to temp directory
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}

		// Load config with fallback
		config, err := LoadConfigWithFallback()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if config == nil {
			t.Fatal("Expected config, got nil")
		}
		if config.Name != "project-persona" {
			t.Errorf("Expected project-persona, got %s", config.Name)
		}
	})

	t.Run("NoProjectConfigFallsBackToGlobal", func(t *testing.T) {
		// Create temp directory without config
		tmpDir, err := os.MkdirTemp("", "ccpersona-test-empty-*")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = os.RemoveAll(tmpDir)
		}()

		// Change to temp directory (no config here)
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}

		// This will try to load from home directory
		// We can't easily mock home directory, so just verify no error
		config, err := LoadConfigWithFallback()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		// config may be nil if no global config exists, which is OK
		_ = config
	})
}

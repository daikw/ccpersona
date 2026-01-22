package persona

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

const (
	ConfigFileName = "persona.json"
	ClaudeDir      = ".claude"
)

// LoadConfig loads persona configuration from the specified directory's .claude directory
func LoadConfig(projectPath string) (*Config, error) {
	configPath := filepath.Join(projectPath, ClaudeDir, ConfigFileName)

	log.Debug().Str("path", configPath).Msg("Loading persona config")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debug().Msg("No persona config found")
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	log.Debug().Str("persona", config.Name).Msg("Loaded persona config")
	return &config, nil
}

// LoadConfigWithFallback loads persona configuration from the current directory,
// falling back to the home directory if not found.
func LoadConfigWithFallback() (*Config, error) {
	// Try current directory first
	config, err := LoadConfig(".")
	if err != nil {
		return nil, err
	}
	if config != nil {
		log.Debug().Msg("Using project persona config")
		return config, nil
	}

	// Fallback to home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	config, err = LoadConfig(homeDir)
	if err != nil {
		return nil, err
	}
	if config != nil {
		log.Debug().Msg("Using global persona config from home directory")
	}

	return config, nil
}

// SaveConfig saves persona configuration to the project's .claude directory
func SaveConfig(projectPath string, config *Config) error {
	claudeDir := filepath.Join(projectPath, ClaudeDir)

	// Ensure .claude directory exists
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	configPath := filepath.Join(claudeDir, ConfigFileName)

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	log.Debug().Str("path", configPath).Msg("Saved persona config")
	return nil
}

// GetDefaultConfig returns a default persona configuration
func GetDefaultConfig() *Config {
	return &Config{
		Name:           "default",
		OverrideGlobal: false,
	}
}

// ValidateConfig checks if the configuration is valid
func ValidateConfig(config *Config) error {
	if config.Name == "" {
		return fmt.Errorf("persona name cannot be empty")
	}

	if config.Voice != nil {
		if config.Voice.Provider == "" {
			return fmt.Errorf("voice provider cannot be empty when voice is configured")
		}
		if config.Voice.Provider != "voicevox" && config.Voice.Provider != "aivisspeech" {
			return fmt.Errorf("unsupported voice provider: %s", config.Voice.Provider)
		}
	}

	return nil
}

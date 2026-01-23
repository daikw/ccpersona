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

	// File permissions
	DirPermission  = 0755 // Directory permission (rwxr-xr-x)
	FilePermission = 0644 // File permission (rw-r--r--)
)

// Platform identifiers for platform-specific configuration
const (
	PlatformClaudeCode = "claude-code"
	PlatformCodex      = "codex"
	PlatformCursor     = "cursor"
)

// Global config directories for each platform
const (
	GlobalDirClaudeCode = ".claude"
	GlobalDirCodex      = ".codex"
	GlobalDirCursor     = ".cursor"
)

// GetGlobalConfigDir returns the global config directory for a given platform
func GetGlobalConfigDir(platform string) string {
	switch platform {
	case PlatformCodex:
		return GlobalDirCodex
	case PlatformCursor:
		return GlobalDirCursor
	default:
		// Claude Code and unknown platforms use .claude
		return GlobalDirClaudeCode
	}
}

// LoadConfig loads persona configuration from the specified directory's .claude directory
func LoadConfig(projectPath string) (*Config, error) {
	return LoadConfigForPlatform(projectPath, "")
}

// LoadConfigForPlatform loads persona configuration with platform-specific path support.
// For non-Claude platforms (Codex, Cursor), it checks .claude/<platform>/persona.json first.
// Priority: .claude/<platform>/persona.json > .claude/persona.json
// Note: Claude Code does not use a subdirectory; it uses .claude/persona.json directly.
func LoadConfigForPlatform(projectPath, platform string) (*Config, error) {
	var configPath string

	// Try platform-specific path first (only for non-Claude platforms)
	if platform != "" && platform != PlatformClaudeCode {
		configPath = filepath.Join(projectPath, ClaudeDir, platform, ConfigFileName)
		log.Debug().Str("path", configPath).Str("platform", platform).Msg("Trying platform-specific persona config")

		data, err := os.ReadFile(configPath)
		if err == nil {
			var config Config
			if err := json.Unmarshal(data, &config); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
			log.Debug().Str("persona", config.Name).Str("platform", platform).Msg("Loaded platform-specific persona config")
			return &config, nil
		}
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		log.Debug().Str("platform", platform).Msg("No platform-specific config found, trying common config")
	}

	// Fallback to common config (.claude/persona.json)
	configPath = filepath.Join(projectPath, ClaudeDir, ConfigFileName)
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
// This is a convenience wrapper for LoadConfigWithFallbackForPlatform with empty platform.
func LoadConfigWithFallback() (*Config, error) {
	return LoadConfigWithFallbackForPlatform("")
}

// LoadConfigWithFallbackForPlatform loads persona configuration with platform support.
//
// Search order for Claude Code (or empty platform):
//  1. .claude/persona.json (project)
//  2. ~/.claude/persona.json (global)
//
// Search order for Codex:
//  1. .claude/codex/persona.json (project, platform-specific)
//  2. .claude/persona.json (project, common)
//  3. ~/.codex/persona.json (global)
//
// Search order for Cursor:
//  1. .claude/cursor/persona.json (project, platform-specific)
//  2. .claude/persona.json (project, common)
//  3. ~/.cursor/persona.json (global)
func LoadConfigWithFallbackForPlatform(platform string) (*Config, error) {
	// Try current directory first (with platform support)
	config, err := LoadConfigForPlatform(".", platform)
	if err != nil {
		return nil, err
	}
	if config != nil {
		log.Debug().Str("platform", platform).Msg("Using project persona config")
		return config, nil
	}

	// Fallback to platform-specific global directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Get the correct global config directory for this platform
	globalDir := GetGlobalConfigDir(platform)
	globalConfigPath := filepath.Join(homeDir, globalDir, ConfigFileName)

	log.Debug().Str("path", globalConfigPath).Str("platform", platform).Msg("Trying global persona config")

	data, err := os.ReadFile(globalConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debug().Str("platform", platform).Msg("No global persona config found")
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read global config file: %w", err)
	}

	config = &Config{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse global config file: %w", err)
	}

	log.Debug().Str("persona", config.Name).Str("platform", platform).Msg("Using global persona config")
	return config, nil
}

// SaveConfig saves persona configuration to the project's .claude directory
func SaveConfig(projectPath string, config *Config) error {
	claudeDir := filepath.Join(projectPath, ClaudeDir)

	// Ensure .claude directory exists
	if err := os.MkdirAll(claudeDir, DirPermission); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	configPath := filepath.Join(claudeDir, ConfigFileName)

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, FilePermission); err != nil {
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

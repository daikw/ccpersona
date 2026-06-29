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
	AgentsDir      = ".agents"
	CodexDir       = ".codex"
	CursorDir      = ".cursor"

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
	GlobalDirClaudeCode = ClaudeDir
	GlobalDirCodex      = CodexDir
	GlobalDirCursor     = CursorDir
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

// LoadConfig loads persona configuration from the specified directory.
func LoadConfig(projectPath string) (*Config, error) {
	return LoadConfigForPlatform(projectPath, "")
}

// LoadConfigForPlatform loads persona configuration with platform-specific path support.
// Priority:
//   - Claude Code: .claude/persona.json > .agents/persona.json
//   - Codex: .codex/persona.json > .claude/codex/persona.json > .agents/persona.json > .claude/persona.json
//   - Cursor: .cursor/persona.json > .claude/cursor/persona.json > .agents/persona.json > .claude/persona.json
//
// The .claude/<platform>/persona.json and .claude/persona.json paths are kept
// for backward compatibility. New shared agent config should use .agents/persona.json.
func LoadConfigForPlatform(projectPath, platform string) (*Config, error) {
	for _, configPath := range projectConfigPaths(projectPath, platform) {
		config, err := loadConfigFile(configPath)
		if err != nil {
			return nil, err
		}
		if config != nil {
			log.Debug().
				Str("persona", config.Name).
				Str("path", configPath).
				Str("platform", platform).
				Msg("Loaded persona config")
			return config, nil
		}
	}

	log.Debug().Str("platform", platform).Msg("No persona config found")
	return nil, nil
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
//  2. .agents/persona.json (project, shared)
//  3. ~/.claude/persona.json (global)
//  4. ~/.agents/persona.json (global, shared)
//
// Search order for Codex:
//  1. .codex/persona.json (project, platform-specific)
//  2. .claude/codex/persona.json (project, legacy platform-specific)
//  3. .agents/persona.json (project, shared)
//  4. .claude/persona.json (project, legacy shared)
//  5. ~/.codex/persona.json (global)
//  6. ~/.agents/persona.json (global, shared)
//
// Search order for Cursor:
//  1. .cursor/persona.json (project, platform-specific)
//  2. .claude/cursor/persona.json (project, legacy platform-specific)
//  3. .agents/persona.json (project, shared)
//  4. .claude/persona.json (project, legacy shared)
//  5. ~/.cursor/persona.json (global)
//  6. ~/.agents/persona.json (global, shared)
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

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	for _, configPath := range globalConfigPaths(homeDir, platform) {
		config, err := loadConfigFile(configPath)
		if err != nil {
			return nil, err
		}
		if config != nil {
			log.Debug().
				Str("persona", config.Name).
				Str("path", configPath).
				Str("platform", platform).
				Msg("Using global persona config")
			return config, nil
		}
	}

	log.Debug().Str("platform", platform).Msg("No global persona config found")
	return nil, nil
}

func projectConfigPaths(projectPath, platform string) []string {
	switch platform {
	case PlatformCodex:
		return []string{
			filepath.Join(projectPath, CodexDir, ConfigFileName),
			filepath.Join(projectPath, ClaudeDir, PlatformCodex, ConfigFileName),
			filepath.Join(projectPath, AgentsDir, ConfigFileName),
			filepath.Join(projectPath, ClaudeDir, ConfigFileName),
		}
	case PlatformCursor:
		return []string{
			filepath.Join(projectPath, CursorDir, ConfigFileName),
			filepath.Join(projectPath, ClaudeDir, PlatformCursor, ConfigFileName),
			filepath.Join(projectPath, AgentsDir, ConfigFileName),
			filepath.Join(projectPath, ClaudeDir, ConfigFileName),
		}
	default:
		return []string{
			filepath.Join(projectPath, ClaudeDir, ConfigFileName),
			filepath.Join(projectPath, AgentsDir, ConfigFileName),
		}
	}
}

func globalConfigPaths(homeDir, platform string) []string {
	paths := []string{filepath.Join(homeDir, GetGlobalConfigDir(platform), ConfigFileName)}
	paths = append(paths, filepath.Join(homeDir, AgentsDir, ConfigFileName))
	return paths
}

func loadConfigFile(configPath string) (*Config, error) {
	log.Debug().Str("path", configPath).Msg("Trying persona config")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	return &config, nil
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

package persona

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/daikw/ccpersona/internal/voice"
	"github.com/rs/zerolog/log"
)

const (
	ConfigFileName        = "ccpersona.json"
	LegacyPersonaFileName = "persona.json"
	LegacyVoiceConfigName = "config.json"
	ClaudeDir             = ".claude"
	AgentsDir             = ".agents"
	CodexDir              = ".codex"
	CursorDir             = ".cursor"
	DirPermission         = 0755
	FilePermission        = 0600
)

// Platform identifiers for platform-specific configuration.
const (
	PlatformClaudeCode = "claude-code"
	PlatformCodex      = "codex"
	PlatformCursor     = "cursor"
)

var (
	warnedBrokenConfig sync.Map
	warnedLegacyConfig sync.Map
)

// ConfigPath returns the canonical ccpersona config path under baseDir.
func ConfigPath(baseDir string) string {
	return filepath.Join(baseDir, AgentsDir, ConfigFileName)
}

// GetGlobalConfigDir returns the canonical global config directory.
func GetGlobalConfigDir(platform string) string {
	return AgentsDir
}

// LoadConfig loads unified ccpersona configuration from the specified project.
func LoadConfig(projectPath string) (*Config, error) {
	return LoadConfigForPlatform(projectPath, "")
}

// LoadConfigForPlatform loads unified configuration from .agents/ccpersona.json.
// Platform is accepted for call-site compatibility, but no longer changes the
// path: the unified config is shared across Claude Code, Codex, Cursor, and MCP.
func LoadConfigForPlatform(projectPath, platform string) (*Config, error) {
	path := ConfigPath(projectPath)
	config, err := loadConfigFile(path)
	if err != nil {
		warnBrokenConfig(path, err)
		return nil, nil
	}
	if config == nil {
		warnMigrationRequired(projectPath, platform)
		return nil, nil
	}
	log.Debug().Str("persona", config.Name).Str("path", path).Msg("Loaded ccpersona config")
	return config, nil
}

// LoadConfigWithFallback loads configuration from the current directory,
// falling back to ~/.agents/ccpersona.json.
func LoadConfigWithFallback() (*Config, error) {
	return LoadConfigWithFallbackForPlatform("")
}

// LoadConfigWithFallbackForPlatform loads unified project config first, then
// unified global config. Broken files are reported to stderr and ignored so
// runtime paths such as voice synthesis can continue with built-in defaults.
func LoadConfigWithFallbackForPlatform(platform string) (*Config, error) {
	config, err := LoadConfigForPlatform(".", platform)
	if err != nil {
		return nil, err
	}
	if config != nil {
		return config, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	config, err = LoadConfigForPlatform(homeDir, platform)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// LoadConfigFromPath loads a specific unified config file strictly.
func LoadConfigFromPath(path string) (*Config, error) {
	return loadConfigFile(path)
}

func loadConfigFile(path string) (*Config, error) {
	log.Debug().Str("path", path).Msg("Trying ccpersona config")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	expanded := expandConfigEnvVars(string(data))
	var config Config
	if err := json.Unmarshal([]byte(expanded), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}
	return &config, nil
}

func expandConfigEnvVars(input string) string {
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(input, func(match string) string {
		name := match[2 : len(match)-1]
		if value, ok := os.LookupEnv(name); ok {
			return value
		}
		log.Debug().Msg("Referenced environment variable not set in config")
		return ""
	})
}

func warnBrokenConfig(path string, err error) {
	if _, loaded := warnedBrokenConfig.LoadOrStore(path, true); loaded {
		return
	}
	fmt.Fprintf(os.Stderr, "ccpersona: failed to load %s; using built-in defaults: %v\n", path, err)
}

func warnMigrationRequired(baseDir, platform string) {
	paths := existingLegacyPaths(baseDir, platform)
	if len(paths) == 0 {
		return
	}
	key := strings.Join(paths, "\x00")
	if _, loaded := warnedLegacyConfig.LoadOrStore(key, true); loaded {
		return
	}
	fmt.Fprintf(os.Stderr, "ccpersona: legacy configuration detected and ignored; run \"ccpersona config migrate\" to create %s\n", ConfigPath(baseDir))
	for _, path := range paths {
		fmt.Fprintf(os.Stderr, "ccpersona: legacy config: %s\n", path)
	}
}

func existingLegacyPaths(baseDir, platform string) []string {
	var paths []string
	for _, path := range legacyConfigPaths(baseDir, platform) {
		if _, err := os.Stat(path); err == nil {
			paths = append(paths, path)
		}
	}
	return paths
}

func legacyConfigPaths(baseDir, platform string) []string {
	paths := []string{
		filepath.Join(baseDir, AgentsDir, LegacyPersonaFileName),
		filepath.Join(baseDir, ClaudeDir, LegacyPersonaFileName),
		filepath.Join(baseDir, ClaudeDir, LegacyVoiceConfigName),
	}
	switch platform {
	case PlatformCodex:
		paths = append([]string{
			filepath.Join(baseDir, CodexDir, LegacyPersonaFileName),
			filepath.Join(baseDir, ClaudeDir, PlatformCodex, LegacyPersonaFileName),
		}, paths...)
	case PlatformCursor:
		paths = append([]string{
			filepath.Join(baseDir, CursorDir, LegacyPersonaFileName),
			filepath.Join(baseDir, ClaudeDir, PlatformCursor, LegacyPersonaFileName),
		}, paths...)
	}
	return paths
}

// SaveConfig saves unified configuration to .agents/ccpersona.json.
func SaveConfig(projectPath string, config *Config) error {
	agentsDir := filepath.Join(projectPath, AgentsDir)
	if err := os.MkdirAll(agentsDir, DirPermission); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", AgentsDir, err)
	}

	configPath := filepath.Join(agentsDir, ConfigFileName)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(configPath, data, FilePermission); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	log.Debug().Str("path", configPath).Msg("Saved ccpersona config")
	return nil
}

// GetDefaultConfig returns a default persona configuration.
func GetDefaultConfig() *Config {
	return &Config{Name: "default"}
}

// ValidateConfig checks if the unified configuration is valid.
func ValidateConfig(config *Config) error {
	if config == nil {
		return nil
	}
	if config.Name == "" {
		return fmt.Errorf("persona name cannot be empty")
	}
	if config.Voice != nil {
		if config.Voice.Volume < 0 || config.Voice.Volume > 2.0 {
			return fmt.Errorf("voice volume must be between 0.0 and 2.0")
		}
		if config.Voice.Speed < 0 || config.Voice.Speed > 4.0 {
			return fmt.Errorf("voice speed must be between 0.0 and 4.0")
		}
	}
	return nil
}

// MigrateConfig merges legacy persona and voice config files into
// .agents/ccpersona.json under baseDir.
func MigrateConfig(baseDir string, force bool) (string, error) {
	target := ConfigPath(baseDir)
	if _, err := os.Stat(target); err == nil && !force {
		return "", fmt.Errorf("config already exists: %s (use --force to overwrite)", target)
	} else if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("cannot access target config %s: %w", target, err)
	}

	out := GetDefaultConfig()
	found := false

	if legacyPersona, err := loadFirstLegacyPersona(baseDir); err != nil {
		return "", err
	} else if legacyPersona != nil {
		out = legacyPersona
		found = true
	}

	if legacyVoice, err := loadLegacyVoiceConfig(baseDir); err != nil {
		return "", err
	} else if legacyVoice != nil {
		mergeLegacyVoice(out, legacyVoice)
		found = true
	}

	if !found {
		return "", fmt.Errorf("no legacy configuration found under %s", baseDir)
	}
	if out.Name == "" {
		out.Name = "default"
	}

	if err := SaveConfig(baseDir, out); err != nil {
		return "", err
	}
	return target, nil
}

func loadFirstLegacyPersona(baseDir string) (*Config, error) {
	for _, path := range []string{
		filepath.Join(baseDir, AgentsDir, LegacyPersonaFileName),
		filepath.Join(baseDir, ClaudeDir, LegacyPersonaFileName),
		filepath.Join(baseDir, CodexDir, LegacyPersonaFileName),
		filepath.Join(baseDir, CursorDir, LegacyPersonaFileName),
		filepath.Join(baseDir, ClaudeDir, PlatformCodex, LegacyPersonaFileName),
		filepath.Join(baseDir, ClaudeDir, PlatformCursor, LegacyPersonaFileName),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read legacy persona config %s: %w", path, err)
		}
		var cfg Config
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse legacy persona config %s: %w", path, err)
		}
		return &cfg, nil
	}
	return nil, nil
}

func loadLegacyVoiceConfig(baseDir string) (*voice.ConfigFile, error) {
	path := filepath.Join(baseDir, ClaudeDir, LegacyVoiceConfigName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read legacy voice config %s: %w", path, err)
	}
	var cfg voice.ConfigFile
	if err := json.Unmarshal([]byte(expandConfigEnvVars(string(data))), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse legacy voice config %s: %w", path, err)
	}
	return &cfg, nil
}

func mergeLegacyVoice(out *Config, legacy *voice.ConfigFile) {
	if out.Voice == nil {
		out.Voice = &VoiceConfig{}
	}
	if out.Voice.Provider == "" {
		out.Voice.Provider = legacy.DefaultProvider
	}
	if legacy.Defaults != nil {
		if out.Voice.Volume == 0 {
			out.Voice.Volume = legacy.Defaults.Volume
		}
		if out.Voice.Speed == 0 {
			out.Voice.Speed = legacy.Defaults.Speed
		}
	}

	if provider := out.Voice.Provider; provider != "" {
		if providerCfg := legacy.GetProviderConfig(provider); providerCfg != nil {
			mergeProviderConfig(out.Voice, providerCfg)
		}
	}
	if len(legacy.Engines) > 0 {
		out.Engines = legacy.Engines
	}
}

func mergeProviderConfig(dst *VoiceConfig, src *voice.ProviderConfig) {
	if dst.APIKey == "" {
		dst.APIKey = src.APIKey
	}
	if dst.Voice == "" {
		dst.Voice = src.Voice
	}
	if dst.Model == "" {
		dst.Model = src.Model
	}
	if dst.Format == "" {
		dst.Format = src.Format
	}
	if dst.Speed == 0 {
		dst.Speed = src.Speed
	}
	if dst.Host == "" {
		dst.Host = src.Host
	}
	if dst.Port == 0 {
		dst.Port = src.Port
	}
	if dst.Speaker == 0 {
		dst.Speaker = src.Speaker
	}
	if dst.BaseURL == "" {
		dst.BaseURL = src.BaseURL
	}
	if dst.TimeoutSeconds == 0 {
		dst.TimeoutSeconds = src.TimeoutSeconds
	}
	if dst.Stability == 0 {
		dst.Stability = src.Stability
	}
	if dst.SimilarityBoost == 0 {
		dst.SimilarityBoost = src.SimilarityBoost
	}
	if dst.Style == 0 {
		dst.Style = src.Style
	}
	if dst.UseSpeakerBoost == nil {
		dst.UseSpeakerBoost = src.UseSpeakerBoost
	}
	if dst.Region == "" {
		dst.Region = src.Region
	}
	if dst.Engine == "" {
		dst.Engine = src.Engine
	}
	if dst.SampleRate == "" {
		dst.SampleRate = src.SampleRate
	}
	if dst.Volume == 0 {
		dst.Volume = src.Volume
	}
}

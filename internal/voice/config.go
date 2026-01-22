package voice

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

// ConfigFile represents the ccpersona configuration file structure
type ConfigFile struct {
	DefaultProvider string                    `json:"default_provider,omitempty"`
	Providers       map[string]ProviderConfig `json:"providers,omitempty"`
	Defaults        *DefaultsConfig           `json:"defaults,omitempty"`
}

// DefaultsConfig represents default values for voice synthesis
type DefaultsConfig struct {
	Volume float64 `json:"volume,omitempty"`
	Speed  float64 `json:"speed,omitempty"`
}

// ProviderConfig represents provider-specific configuration
type ProviderConfig struct {
	// Common options
	APIKey string  `json:"api_key,omitempty"`
	Voice  string  `json:"voice,omitempty"`
	Model  string  `json:"model,omitempty"`
	Format string  `json:"format,omitempty"`
	Speed  float64 `json:"speed,omitempty"`

	// Local engine options (VOICEVOX/AivisSpeech)
	Host    string `json:"host,omitempty"`
	Port    int    `json:"port,omitempty"`
	Speaker int    `json:"speaker,omitempty"`

	// OpenAI options
	// (uses common options)

	// ElevenLabs options
	Stability       float64 `json:"stability,omitempty"`
	SimilarityBoost float64 `json:"similarity_boost,omitempty"`
	Style           float64 `json:"style,omitempty"`
	UseSpeakerBoost *bool   `json:"use_speaker_boost,omitempty"`

	// Amazon Polly options
	Region     string `json:"region,omitempty"`
	Engine     string `json:"engine,omitempty"`
	SampleRate string `json:"sample_rate,omitempty"`

	// Volume control (provider-specific override)
	Volume float64 `json:"volume,omitempty"`
}

// ConfigLoader handles loading configuration from files
type ConfigLoader struct {
	projectPath string
	globalPath  string
}

// NewConfigLoader creates a new config loader
func NewConfigLoader() *ConfigLoader {
	homeDir, _ := os.UserHomeDir()
	return &ConfigLoader{
		projectPath: ".claude/config.json",
		globalPath:  filepath.Join(homeDir, ".claude", "config.json"),
	}
}

// LoadConfig loads configuration with priority:
// 1. Project-local config (.claude/config.json)
// 2. Global config (~/.claude/config.json)
// Returns nil if no config file found
func (l *ConfigLoader) LoadConfig(workDir string) (*ConfigFile, error) {
	// Try project-local config first
	projectConfigPath := filepath.Join(workDir, l.projectPath)
	if config, err := l.loadFromFile(projectConfigPath); err == nil {
		log.Debug().Str("path", projectConfigPath).Msg("Loaded project config")
		return config, nil
	}

	// Try global config
	if config, err := l.loadFromFile(l.globalPath); err == nil {
		log.Debug().Str("path", l.globalPath).Msg("Loaded global config")
		return config, nil
	}

	log.Debug().Msg("No config file found")
	return nil, nil
}

// LoadFromPath loads configuration from a specific path
// For security, it validates that the path doesn't traverse outside expected directories
func (l *ConfigLoader) LoadFromPath(path string) (*ConfigFile, error) {
	// Validate path to prevent path traversal attacks
	if err := validateConfigPath(path); err != nil {
		return nil, err
	}
	return l.loadFromFile(path)
}

// validateConfigPath checks that the config path is safe to use
func validateConfigPath(path string) error {
	// Check for path traversal attempts BEFORE cleaning
	// This catches both obvious and hidden traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("invalid config path: path traversal not allowed")
	}

	// Clean the path for further validation
	cleanPath := filepath.Clean(path)

	// Ensure path ends with expected filename
	if !strings.HasSuffix(cleanPath, "config.json") {
		return fmt.Errorf("invalid config path: must be a config.json file")
	}

	return nil
}

func (l *ConfigLoader) loadFromFile(path string) (*ConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Expand environment variables
	expanded := expandEnvVars(string(data))

	var config ConfigFile
	if err := json.Unmarshal([]byte(expanded), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	// Check file permissions (warn if too permissive)
	l.checkFilePermissions(path)

	return &config, nil
}

// expandEnvVars replaces ${VAR} patterns with environment variable values
func expandEnvVars(input string) string {
	// Match ${VAR} pattern
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	result := re.ReplaceAllStringFunc(input, func(match string) string {
		varName := match[2 : len(match)-1] // Remove ${ and }
		if value, exists := os.LookupEnv(varName); exists {
			return value
		}
		// Don't log variable names for security reasons
		log.Debug().Msg("Referenced environment variable not set in config")
		return ""
	})

	return result
}

func (l *ConfigLoader) checkFilePermissions(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}

	mode := info.Mode().Perm()
	// Warn if file has any group or world permissions (read/write/execute)
	if mode&0077 != 0 {
		log.Warn().
			Str("permissions", fmt.Sprintf("%04o", mode)).
			Msg("Config file may contain secrets but has permissive permissions. Consider: chmod 600")
	}
}

// GetProviderConfig returns configuration for a specific provider
func (c *ConfigFile) GetProviderConfig(providerName string) *ProviderConfig {
	if c == nil || c.Providers == nil {
		return nil
	}
	if config, exists := c.Providers[providerName]; exists {
		return &config
	}
	return nil
}

// GetEffectiveProvider returns the provider to use (explicit or default)
func (c *ConfigFile) GetEffectiveProvider(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if c != nil && c.DefaultProvider != "" {
		return c.DefaultProvider
	}
	return ""
}

// GetDefaultVolume returns the default volume setting
func (c *ConfigFile) GetDefaultVolume() float64 {
	if c != nil && c.Defaults != nil && c.Defaults.Volume > 0 {
		return c.Defaults.Volume
	}
	return 1.0
}

// GetDefaultSpeed returns the default speed setting
func (c *ConfigFile) GetDefaultSpeed() float64 {
	if c != nil && c.Defaults != nil && c.Defaults.Speed > 0 {
		return c.Defaults.Speed
	}
	return 1.0
}

// Validate validates the configuration
func (c *ConfigFile) Validate() []string {
	var errors []string

	if c == nil {
		return errors
	}

	// Validate provider configurations
	for name, provider := range c.Providers {
		providerErrors := validateProviderConfig(name, &provider)
		errors = append(errors, providerErrors...)
	}

	// Validate defaults
	if c.Defaults != nil {
		if c.Defaults.Volume != 0 && (c.Defaults.Volume < 0 || c.Defaults.Volume > 2.0) {
			errors = append(errors, "defaults: volume must be between 0.0 and 2.0")
		}
		if c.Defaults.Speed != 0 && (c.Defaults.Speed < 0.25 || c.Defaults.Speed > 4.0) {
			errors = append(errors, "defaults: speed must be between 0.25 and 4.0")
		}
	}

	return errors
}

func validateProviderConfig(name string, config *ProviderConfig) []string {
	var errors []string

	switch name {
	case "openai":
		if config.APIKey == "" {
			errors = append(errors, fmt.Sprintf("%s: api_key is required (use ${OPENAI_API_KEY} for env var)", name))
		}
	case "elevenlabs":
		if config.APIKey == "" {
			errors = append(errors, fmt.Sprintf("%s: api_key is required (use ${ELEVENLABS_API_KEY} for env var)", name))
		}
		if config.Stability < 0 || config.Stability > 1 {
			errors = append(errors, fmt.Sprintf("%s: stability must be between 0.0 and 1.0", name))
		}
		if config.SimilarityBoost < 0 || config.SimilarityBoost > 1 {
			errors = append(errors, fmt.Sprintf("%s: similarity_boost must be between 0.0 and 1.0", name))
		}
	case "polly":
		validRegions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-northeast-1", "ap-southeast-1"}
		if config.Region != "" && !contains(validRegions, config.Region) {
			errors = append(errors, fmt.Sprintf("%s: region '%s' may not be valid", name, config.Region))
		}
	case "voicevox", "aivisspeech":
		if config.Port != 0 && (config.Port < 1 || config.Port > 65535) {
			errors = append(errors, fmt.Sprintf("%s: port must be between 1 and 65535", name))
		}
	}

	if config.Speed != 0 && (config.Speed < 0.25 || config.Speed > 4.0) {
		errors = append(errors, fmt.Sprintf("%s: speed must be between 0.25 and 4.0", name))
	}

	if config.Volume != 0 && (config.Volume < 0 || config.Volume > 2.0) {
		errors = append(errors, fmt.Sprintf("%s: volume must be between 0.0 and 2.0", name))
	}

	return errors
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GenerateExampleConfig generates an example configuration
func GenerateExampleConfig() string {
	example := ConfigFile{
		DefaultProvider: "aivisspeech",
		Providers: map[string]ProviderConfig{
			"openai": {
				APIKey: "${OPENAI_API_KEY}",
				Model:  "tts-1",
				Voice:  "nova",
				Speed:  1.0,
				Format: "mp3",
			},
			"elevenlabs": {
				APIKey:          "${ELEVENLABS_API_KEY}",
				Voice:           "Rachel",
				Model:           "eleven_multilingual_v2",
				Stability:       0.5,
				SimilarityBoost: 0.75,
			},
			"polly": {
				Region:     "us-east-1",
				Voice:      "Joanna",
				Engine:     "neural",
				SampleRate: "22050",
			},
			"voicevox": {
				Host:    "localhost",
				Port:    50021,
				Speaker: 3,
			},
			"aivisspeech": {
				Host:    "localhost",
				Port:    10101,
				Speaker: 888753760,
			},
		},
		Defaults: &DefaultsConfig{
			Volume: 1.0,
			Speed:  1.0,
		},
	}

	data, _ := json.MarshalIndent(example, "", "  ")
	return string(data)
}

// MaskSecrets masks sensitive values in config for display
// For security, only shows that a key is present, not its contents
func (c *ConfigFile) MaskSecrets() *ConfigFile {
	if c == nil {
		return nil
	}

	masked := &ConfigFile{
		DefaultProvider: c.DefaultProvider,
		Providers:       make(map[string]ProviderConfig),
		Defaults:        c.Defaults,
	}

	for name, provider := range c.Providers {
		maskedProvider := provider
		if provider.APIKey != "" {
			// Only indicate that a key is set, don't reveal any characters
			maskedProvider.APIKey = fmt.Sprintf("[set, %d chars]", len(provider.APIKey))
		}
		masked.Providers[name] = maskedProvider
	}

	return masked
}

// Deprecated: VoiceConfigFile is deprecated, use ConfigFile instead
type VoiceConfigFile = ConfigFile

// Deprecated: VoiceConfigLoader is deprecated, use ConfigLoader instead
type VoiceConfigLoader = ConfigLoader

// Deprecated: NewVoiceConfigLoader is deprecated, use NewConfigLoader instead
func NewVoiceConfigLoader() *ConfigLoader {
	return NewConfigLoader()
}

package persona

import "github.com/daikw/ccpersona/internal/voice"

// Config represents the persona configuration for a project
type Config struct {
	Name               string                            `json:"name"`
	Voice              *VoiceConfig                      `json:"voice,omitempty"`
	CustomInstructions string                            `json:"custom_instructions,omitempty"`
	Engines            map[string]voice.EngineUserConfig `json:"engines,omitempty"`
}

// VoiceConfig represents the active voice synthesis settings for a persona.
type VoiceConfig struct {
	Provider string  `json:"provider,omitempty"`
	Speaker  int     `json:"speaker,omitempty"`
	Volume   float64 `json:"volume,omitempty"`
	Speed    float64 `json:"speed,omitempty"`

	APIKey string `json:"api_key,omitempty"`
	Voice  string `json:"voice,omitempty"`
	Model  string `json:"model,omitempty"`
	Format string `json:"format,omitempty"`

	Host           string `json:"host,omitempty"`
	Port           int    `json:"port,omitempty"`
	BaseURL        string `json:"base_url,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`

	Stability       float64 `json:"stability,omitempty"`
	SimilarityBoost float64 `json:"similarity_boost,omitempty"`
	Style           float64 `json:"style,omitempty"`
	UseSpeakerBoost *bool   `json:"use_speaker_boost,omitempty"`

	Region     string `json:"region,omitempty"`
	Engine     string `json:"engine,omitempty"`
	SampleRate string `json:"sample_rate,omitempty"`
}

// ToVoiceInput converts the unified config into the small resolver input used
// for persona-level precedence.
func (c *Config) ToVoiceInput() voice.PersonaVoiceInput {
	if c == nil || c.Voice == nil {
		return voice.PersonaVoiceInput{}
	}
	return voice.PersonaVoiceInput{
		Provider: c.Voice.Provider,
		Speaker:  c.Voice.Speaker,
		Volume:   c.Voice.Volume,
		Speed:    c.Voice.Speed,
	}
}

// ToVoiceConfigFile projects the unified config onto the voice resolver's file
// representation. The new schema has one active voice block, so it becomes the
// provider-specific config for voice.provider.
func (c *Config) ToVoiceConfigFile() *voice.ConfigFile {
	if c == nil {
		return nil
	}

	out := &voice.ConfigFile{
		Engines: c.Engines,
	}
	if c.Voice == nil {
		return out
	}

	provider := c.Voice.Provider
	out.DefaultProvider = provider
	out.Defaults = &voice.DefaultsConfig{
		Volume: c.Voice.Volume,
		Speed:  c.Voice.Speed,
	}

	if provider != "" {
		out.Providers = map[string]voice.ProviderConfig{
			provider: c.Voice.ToProviderConfig(),
		}
	}
	return out
}

// ToProviderConfig converts the active voice block into provider-specific TTS
// settings.
func (v *VoiceConfig) ToProviderConfig() voice.ProviderConfig {
	if v == nil {
		return voice.ProviderConfig{}
	}
	return voice.ProviderConfig{
		APIKey:          v.APIKey,
		Voice:           v.Voice,
		Model:           v.Model,
		Format:          v.Format,
		Speed:           v.Speed,
		Host:            v.Host,
		Port:            v.Port,
		Speaker:         v.Speaker,
		BaseURL:         v.BaseURL,
		TimeoutSeconds:  v.TimeoutSeconds,
		Stability:       v.Stability,
		SimilarityBoost: v.SimilarityBoost,
		Style:           v.Style,
		UseSpeakerBoost: v.UseSpeakerBoost,
		Region:          v.Region,
		Engine:          v.Engine,
		SampleRate:      v.SampleRate,
		Volume:          v.Volume,
	}
}

// Definition represents a persona definition
type Definition struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	FilePath      string            `json:"file_path"`
	Tone          string            `json:"tone"`
	Approach      string            `json:"approach"`
	Specialties   []string          `json:"specialties"`
	DialogueStyle string            `json:"dialogue_style"`
	Values        map[string]string `json:"values"`
}

// PersonaFile represents the structure of a persona markdown file
type PersonaFile struct {
	Title          string
	Tone           string
	Approach       string
	Values         string
	CustomSections map[string]string
}

package persona

// Config represents the persona configuration for a project
type Config struct {
	Name               string       `json:"name"`
	Voice              *VoiceConfig `json:"voice,omitempty"`
	OverrideGlobal     bool         `json:"override_global,omitempty"`
	CustomInstructions string       `json:"custom_instructions,omitempty"`
}

// VoiceConfig represents voice synthesis settings for a persona
type VoiceConfig struct {
	Provider string  `json:"provider"`
	Speaker  int     `json:"speaker"`
	Volume   float64 `json:"volume,omitempty"`
	Speed    float64 `json:"speed,omitempty"`
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

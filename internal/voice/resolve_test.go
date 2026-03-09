package voice

import (
	"testing"
)

func TestResolve_Defaults(t *testing.T) {
	opts := Resolve(PersonaVoiceInput{}, nil, "")

	if opts.Volume != 1.0 {
		t.Errorf("expected opts.Volume=1.0, got %v", opts.Volume)
	}
	if opts.Speed != 1.0 {
		t.Errorf("expected opts.Speed=1.0, got %v", opts.Speed)
	}
	if opts.Provider != "" {
		t.Errorf("expected empty provider, got %q", opts.Provider)
	}
}

func TestResolve_FileConfigDefaults(t *testing.T) {
	fileConfig := &ConfigFile{
		Defaults: &DefaultsConfig{
			Volume: 0.8,
			Speed:  1.5,
		},
	}

	opts := Resolve(PersonaVoiceInput{}, fileConfig, "")

	if opts.Volume != 0.8 {
		t.Errorf("expected opts.Volume=0.8, got %v", opts.Volume)
	}
	if opts.Speed != 1.5 {
		t.Errorf("expected opts.Speed=1.5, got %v", opts.Speed)
	}
}

func TestResolve_PersonaOverridesFileDefaults(t *testing.T) {
	fileConfig := &ConfigFile{
		Defaults: &DefaultsConfig{Volume: 0.8, Speed: 1.5},
	}
	personaInput := PersonaVoiceInput{
		Provider: "aivisspeech",
		Speaker:  42,
		Volume:   1.2,
		Speed:    0.9,
	}

	opts := Resolve(personaInput, fileConfig, "")
	cfg := opts.ToConfig(DefaultConfig())

	if opts.Volume != 1.2 {
		t.Errorf("expected opts.Volume=1.2, got %v", opts.Volume)
	}
	if opts.Speed != 0.9 {
		t.Errorf("expected opts.Speed=0.9, got %v", opts.Speed)
	}
	if opts.Provider != "aivisspeech" {
		t.Errorf("expected opts.Provider=aivisspeech, got %q", opts.Provider)
	}
	if opts.AivisSpeechSpeaker != 42 {
		t.Errorf("expected opts.AivisSpeechSpeaker=42, got %v", opts.AivisSpeechSpeaker)
	}
	// ToConfig should propagate all fields
	if cfg.SpeedScale != 0.9 {
		t.Errorf("expected cfg.SpeedScale=0.9, got %v", cfg.SpeedScale)
	}
	if cfg.VolumeScale != 1.2 {
		t.Errorf("expected cfg.VolumeScale=1.2, got %v", cfg.VolumeScale)
	}
	if cfg.EnginePriority != "aivisspeech" {
		t.Errorf("expected cfg.EnginePriority=aivisspeech, got %v", cfg.EnginePriority)
	}
	if cfg.AivisSpeechSpeaker != 42 {
		t.Errorf("expected cfg.AivisSpeechSpeaker=42, got %v", cfg.AivisSpeechSpeaker)
	}
}

func TestResolve_CLIProviderOverridesPersona(t *testing.T) {
	personaInput := PersonaVoiceInput{Provider: "aivisspeech"}

	opts := Resolve(personaInput, nil, "voicevox")
	cfg := opts.ToConfig(DefaultConfig())

	if cfg.EnginePriority != "voicevox" {
		t.Errorf("expected EnginePriority=voicevox, got %q", cfg.EnginePriority)
	}
	if opts.Provider != "voicevox" {
		t.Errorf("expected opts.Provider=voicevox, got %q", opts.Provider)
	}
}

func TestResolve_ProviderConfigOverridesDefaults(t *testing.T) {
	trueVal := true
	fileConfig := &ConfigFile{
		Defaults: &DefaultsConfig{Volume: 0.5, Speed: 1.0},
		Providers: map[string]ProviderConfig{
			"openai": {
				Speed:  1.3,
				Volume: 1.1,
				APIKey: "sk-test",
				Voice:  "nova",
				Model:  "tts-1-hd",
				Format: "mp3",
				// ElevenLabs
				Stability:       0.7,
				SimilarityBoost: 0.8,
				Style:           0.2,
				UseSpeakerBoost: &trueVal,
				// Polly
				Region:     "eu-west-1",
				Engine:     "standard",
				SampleRate: "44100",
			},
		},
	}

	opts := Resolve(PersonaVoiceInput{}, fileConfig, "openai")

	if opts.Speed != 1.3 {
		t.Errorf("expected opts.Speed=1.3, got %v", opts.Speed)
	}
	if opts.Volume != 1.1 {
		t.Errorf("expected opts.Volume=1.1, got %v", opts.Volume)
	}
	if opts.APIKey != "sk-test" {
		t.Errorf("expected APIKey=sk-test, got %q", opts.APIKey)
	}
	if opts.Voice != "nova" {
		t.Errorf("expected Voice=nova, got %q", opts.Voice)
	}
	if opts.Region != "eu-west-1" {
		t.Errorf("expected Region=eu-west-1, got %q", opts.Region)
	}
	if opts.Engine != "standard" {
		t.Errorf("expected Engine=standard, got %q", opts.Engine)
	}
}

// TestResolve_EmptyCLIProviderDoesNotOverridePersona ensures that passing an empty
// cliProvider (= flag not explicitly set) does not shadow persona/file provider.
func TestResolve_EmptyCLIProviderDoesNotOverridePersona(t *testing.T) {
	personaInput := PersonaVoiceInput{Provider: "voicevox"}

	opts := Resolve(personaInput, nil, "") // empty cliProvider
	cfg := opts.ToConfig(DefaultConfig())

	if cfg.EnginePriority != "voicevox" {
		t.Errorf("expected EnginePriority=voicevox, got %q", cfg.EnginePriority)
	}
	if opts.Provider != "voicevox" {
		t.Errorf("expected opts.Provider=voicevox, got %q", opts.Provider)
	}
}

// TestResolve_CLIProviderVoicevox_PersonaSpeakerGoesToVoicevox ensures that when the
// CLI explicitly requests voicevox, a persona speaker is written to VoicevoxSpeaker
// even if persona.Provider is empty (regression guard for the cfg.EnginePriority bug).
func TestResolve_CLIProviderVoicevox_PersonaSpeakerGoesToVoicevox(t *testing.T) {
	personaInput := PersonaVoiceInput{
		Provider: "", // persona doesn't set provider
		Speaker:  99,
	}

	opts := Resolve(personaInput, nil, "voicevox")

	if opts.VoicevoxSpeaker != 99 {
		t.Errorf("expected VoicevoxSpeaker=99, got %d", opts.VoicevoxSpeaker)
	}
	if opts.AivisSpeechSpeaker == 99 {
		t.Errorf("AivisSpeechSpeaker should not be 99 when provider=voicevox")
	}
}

func TestResolve_PersonaOverridesProviderConfig(t *testing.T) {
	fileConfig := &ConfigFile{
		Providers: map[string]ProviderConfig{
			"aivisspeech": {Volume: 0.9, Speed: 1.1},
		},
	}
	personaInput := PersonaVoiceInput{
		Provider: "aivisspeech",
		Volume:   1.4,
		Speed:    0.8,
	}

	opts := Resolve(personaInput, fileConfig, "")

	// Persona should win over provider config
	if opts.Volume != 1.4 {
		t.Errorf("expected opts.Volume=1.4, got %v", opts.Volume)
	}
	if opts.Speed != 0.8 {
		t.Errorf("expected opts.Speed=0.8, got %v", opts.Speed)
	}
}

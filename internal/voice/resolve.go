package voice

import "github.com/rs/zerolog/log"

// PersonaVoiceInput carries voice settings from a persona config,
// avoiding a direct import of the persona package.
type PersonaVoiceInput struct {
	Provider string
	Speaker  int
	Volume   float64
	Speed    float64
}

// Resolve merges all configuration sources into a single VoiceOptions.
// VoiceOptions is the single source of truth for resolved settings.
// Callers that need a *Config (legacy VoiceEngine path) call opts.ToConfig().
//
// Priority (highest → lowest):
//  1. cliProvider argument (provider name only; caller applies CLI speaker/flags after)
//  2. persona (PersonaVoiceInput)
//  3. fileConfig.Providers[effectiveProvider] (per-provider overrides)
//  4. fileConfig.Defaults (global defaults from config file)
//  5. DefaultConfig() hard-coded values
func Resolve(persona PersonaVoiceInput, fileConfig *ConfigFile, cliProvider string) VoiceOptions {
	defaults := DefaultConfig()

	opts := VoiceOptions{
		Speed:           defaults.SpeedScale,
		Volume:          defaults.VolumeScale,
		Format:          "mp3",
		Model:           "tts-1",
		Stability:       0.5,
		SimilarityBoost: 0.5,
		Style:           0.0,
		UseSpeakerBoost: true,
		Region:          "us-east-1",
		Engine:          "neural",
		SampleRate:      "22050",
	}

	// Layer 4: fileConfig.Defaults
	if fileConfig != nil && fileConfig.Defaults != nil {
		if fileConfig.Defaults.Volume > 0 {
			opts.Volume = fileConfig.Defaults.Volume
		}
		if fileConfig.Defaults.Speed > 0 {
			opts.Speed = fileConfig.Defaults.Speed
		}
	}

	// Determine the effective provider before applying provider-specific config.
	// Start with file config default, then persona, then CLI.
	effectiveProvider := ""
	if fileConfig != nil && fileConfig.DefaultProvider != "" {
		effectiveProvider = fileConfig.DefaultProvider
	}
	if persona.Provider != "" {
		effectiveProvider = persona.Provider
	}
	if cliProvider != "" {
		effectiveProvider = cliProvider
	}

	// Layer 3: fileConfig.Providers[effectiveProvider]
	if fileConfig != nil && effectiveProvider != "" {
		if provCfg := fileConfig.GetProviderConfig(effectiveProvider); provCfg != nil {
			if provCfg.Volume > 0 {
				opts.Volume = provCfg.Volume
			}
			if provCfg.Speed > 0 {
				opts.Speed = provCfg.Speed
			}
			if provCfg.Speaker > 0 {
				if effectiveProvider == EngineAivisSpeech {
					opts.AivisSpeechSpeaker = provCfg.Speaker
				} else {
					opts.VoicevoxSpeaker = provCfg.Speaker
				}
			}
			if provCfg.APIKey != "" {
				opts.APIKey = provCfg.APIKey
			}
			if provCfg.Voice != "" {
				opts.Voice = provCfg.Voice
			}
			if provCfg.Model != "" {
				opts.Model = provCfg.Model
			}
			if provCfg.Format != "" {
				opts.Format = provCfg.Format
			}
			if provCfg.Stability > 0 {
				opts.Stability = provCfg.Stability
			}
			if provCfg.SimilarityBoost > 0 {
				opts.SimilarityBoost = provCfg.SimilarityBoost
			}
			if provCfg.Style > 0 {
				opts.Style = provCfg.Style
			}
			if provCfg.UseSpeakerBoost != nil {
				opts.UseSpeakerBoost = *provCfg.UseSpeakerBoost
			}
			if provCfg.Region != "" {
				opts.Region = provCfg.Region
			}
			if provCfg.Engine != "" {
				opts.Engine = provCfg.Engine
			}
			if provCfg.SampleRate != "" {
				opts.SampleRate = provCfg.SampleRate
			}
		}
	}

	// Layer 2: persona
	if persona.Speaker > 0 {
		// Use effectiveProvider so the speaker always lands in the correct field
		// even when persona.Provider is empty.
		if effectiveProvider == EngineAivisSpeech {
			opts.AivisSpeechSpeaker = persona.Speaker
		} else {
			opts.VoicevoxSpeaker = persona.Speaker
		}
	}
	if persona.Volume > 0 {
		opts.Volume = persona.Volume
	}
	if persona.Speed > 0 {
		opts.Speed = persona.Speed
	}

	opts.Provider = effectiveProvider

	log.Debug().
		Str("provider", effectiveProvider).
		Float64("volume", opts.Volume).
		Float64("speed", opts.Speed).
		Msg("Resolved voice config")

	return opts
}

// ToConfig converts VoiceOptions into a *Config for the legacy VoiceEngine path.
// base supplies reading-specific fields (ReadingMode, MaxChars, UUIDMode) that
// are not part of synthesis options.
func (o VoiceOptions) ToConfig(base *Config) *Config {
	cfg := *base
	if o.Provider != "" {
		cfg.EnginePriority = o.Provider
	}
	if o.AivisSpeechSpeaker > 0 {
		cfg.AivisSpeechSpeaker = int64(o.AivisSpeechSpeaker)
	}
	if o.VoicevoxSpeaker > 0 {
		cfg.VoicevoxSpeaker = o.VoicevoxSpeaker
	}
	if o.Speed > 0 {
		cfg.SpeedScale = o.Speed
	}
	if o.Volume > 0 {
		cfg.VolumeScale = o.Volume
	}
	return &cfg
}

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

// Resolve merges all configuration sources into a (*Config, VoiceOptions) pair.
//
// Priority (highest → lowest):
//  1. cliProvider argument (provider name only; caller applies CLI speaker/flags after)
//  2. persona (PersonaVoiceInput)
//  3. fileConfig.Providers[effectiveProvider] (per-provider overrides)
//  4. fileConfig.Defaults (global defaults from config file)
//  5. DefaultConfig() hard-coded values
func Resolve(persona PersonaVoiceInput, fileConfig *ConfigFile, cliProvider string) (*Config, VoiceOptions) {
	cfg := DefaultConfig()

	opts := VoiceOptions{
		Speed:           cfg.SpeedScale,
		Volume:          cfg.VolumeScale,
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
			cfg.VolumeScale = fileConfig.Defaults.Volume
			opts.Volume = fileConfig.Defaults.Volume
		}
		if fileConfig.Defaults.Speed > 0 {
			cfg.SpeedScale = fileConfig.Defaults.Speed
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
				cfg.VolumeScale = provCfg.Volume
				opts.Volume = provCfg.Volume
			}
			if provCfg.Speed > 0 {
				cfg.SpeedScale = provCfg.Speed
				opts.Speed = provCfg.Speed
			}
			if provCfg.Speaker > 0 {
				if effectiveProvider == EngineAivisSpeech {
					cfg.AivisSpeechSpeaker = int64(provCfg.Speaker)
				} else {
					cfg.VoicevoxSpeaker = provCfg.Speaker
				}
			}
			// Cloud-provider-specific fields
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
	if persona.Provider != "" {
		cfg.EnginePriority = persona.Provider
	}
	if persona.Speaker > 0 {
		if cfg.EnginePriority == EngineAivisSpeech {
			cfg.AivisSpeechSpeaker = int64(persona.Speaker)
		} else {
			cfg.VoicevoxSpeaker = persona.Speaker
		}
	}
	if persona.Volume > 0 {
		cfg.VolumeScale = persona.Volume
		opts.Volume = persona.Volume
	}
	if persona.Speed > 0 {
		cfg.SpeedScale = persona.Speed
		opts.Speed = persona.Speed
	}

	// Layer 1: cliProvider (provider name; speaker/flags applied by caller)
	if cliProvider != "" {
		cfg.EnginePriority = cliProvider
	}

	opts.Provider = effectiveProvider

	log.Debug().
		Str("provider", effectiveProvider).
		Float64("volume", opts.Volume).
		Float64("speed", opts.Speed).
		Msg("Resolved voice config")

	return cfg, opts
}

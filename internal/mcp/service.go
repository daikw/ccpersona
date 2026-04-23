package mcp

import (
	"context"
	"fmt"
	"os"

	"github.com/daikw/ccpersona/internal/persona"
	"github.com/daikw/ccpersona/internal/voice"
	"github.com/rs/zerolog/log"
)

// Synthesizer synthesizes text into an audio file.
type Synthesizer interface {
	Synthesize(ctx context.Context, text string, opts voice.VoiceOptions) (string, error)
}

// Player plays an audio file, blocking until playback completes.
type Player interface {
	PlayAudioBlocking(audioPath string) error
}

// SpeakRequest contains parameters for a speak call.
type SpeakRequest struct {
	Text       string
	Provider   string
	Speaker    int
	ProjectDir string
}

// SpeakService handles text-to-speech requests.
type SpeakService struct {
	synthesizer Synthesizer
	player      Player
}

// NewSpeakService creates a new SpeakService with the given dependencies.
func NewSpeakService(synthesizer Synthesizer, player Player) *SpeakService {
	return &SpeakService{
		synthesizer: synthesizer,
		player:      player,
	}
}

// Speak synthesizes the text in req and plays it back.
func (s *SpeakService) Speak(ctx context.Context, req SpeakRequest) error {
	if req.Text == "" {
		return fmt.Errorf("text cannot be empty")
	}

	if voice.IsMuted() {
		log.Debug().Msg("voice synthesis is globally muted, skipping MCP speak")
		return nil
	}

	// Resolve project directory for persona/voice config lookup.
	projectDir := req.ProjectDir
	if projectDir == "" {
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			log.Warn().Err(err).Msg("failed to get cwd; using '.'")
			projectDir = "."
		}
	}

	// Load persona config: project dir first, then global (~/.claude) only.
	// Deliberately avoid LoadConfigWithFallback() which uses cwd and may resolve
	// a persona from a different project when project_dir is explicitly given.
	personaCfg, _ := persona.LoadConfigForPlatform(projectDir, "")
	if personaCfg == nil {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			personaCfg, _ = persona.LoadConfigForPlatform(homeDir, "")
		}
	}

	personaInput := toPersonaVoiceInput(personaCfg)

	// Load voice config file.
	loader := voice.NewConfigLoader()
	fileConfig, _ := loader.LoadConfig(projectDir)

	opts := voice.Resolve(personaInput, fileConfig, req.Provider)

	// Apply request-level speaker override.
	if req.Speaker > 0 {
		opts.VoicevoxSpeaker = req.Speaker
		opts.AivisSpeechSpeaker = req.Speaker
		opts.Voice = "" // clear cloud voice when explicit speaker is given
	}

	opts.PlayAudio = false
	opts.ToStdout = false

	audioPath, err := s.synthesizer.Synthesize(ctx, req.Text, opts)
	if err != nil {
		return fmt.Errorf("synthesis failed: %w", err)
	}

	if audioPath != "" {
		defer os.Remove(audioPath)
	}

	if err := s.player.PlayAudioBlocking(audioPath); err != nil {
		return fmt.Errorf("playback failed: %w", err)
	}

	return nil
}

// toPersonaVoiceInput converts a persona.Config into the voice input struct.
func toPersonaVoiceInput(cfg *persona.Config) voice.PersonaVoiceInput {
	if cfg == nil || cfg.Voice == nil {
		return voice.PersonaVoiceInput{}
	}
	return voice.PersonaVoiceInput{
		Provider: cfg.Voice.Provider,
		Speaker:  cfg.Voice.Speaker,
		Volume:   cfg.Voice.Volume,
		Speed:    cfg.Voice.Speed,
	}
}

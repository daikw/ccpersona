package voice

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// VoiceEngine handles voice synthesis
type VoiceEngine struct {
	config     *Config
	httpClient *http.Client
}

// NewVoiceEngine creates a new voice engine
func NewVoiceEngine(config *Config) *VoiceEngine {
	return &VoiceEngine{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CheckEngines checks which voice engines are available
func (ve *VoiceEngine) CheckEngines() (voicevoxAvailable, aivisSpeechAvailable bool) {
	// Check VOICEVOX
	resp, err := ve.httpClient.Get(VoicevoxURL + "/version")
	if err == nil && resp.StatusCode == http.StatusOK {
		voicevoxAvailable = true
		_ = resp.Body.Close()
		log.Debug().Msg("VOICEVOX ENGINE is available")
	}

	// Check AivisSpeech
	resp, err = ve.httpClient.Get(AivisSpeechURL + "/speakers")
	if err == nil && resp.StatusCode == http.StatusOK {
		aivisSpeechAvailable = true
		_ = resp.Body.Close()
		log.Debug().Msg("AivisSpeech is available")
	}

	return
}

// SelectEngine selects which engine to use based on availability and priority
func (ve *VoiceEngine) SelectEngine() (string, error) {
	voicevox, aivisspeech := ve.CheckEngines()

	if ve.config.EnginePriority == EngineVoicevox {
		if voicevox {
			return EngineVoicevox, nil
		}
		if aivisspeech {
			return EngineAivisSpeech, nil
		}
	} else {
		if aivisspeech {
			return EngineAivisSpeech, nil
		}
		if voicevox {
			return EngineVoicevox, nil
		}
	}

	return "", fmt.Errorf("no voice engine available")
}

// Synthesize generates audio from text
func (ve *VoiceEngine) Synthesize(text string) (string, error) {
	engine, err := ve.SelectEngine()
	if err != nil {
		return "", err
	}

	log.Info().Str("engine", engine).Msg("Using voice engine")

	switch engine {
	case EngineVoicevox:
		return ve.synthesizeVoicevox(text)
	case EngineAivisSpeech:
		return ve.synthesizeAivisSpeech(text)
	default:
		return "", fmt.Errorf("unknown engine: %s", engine)
	}
}

// synthesizeVoicevox uses VOICEVOX ENGINE for synthesis
func (ve *VoiceEngine) synthesizeVoicevox(text string) (string, error) {
	// Create audio query
	queryURL := fmt.Sprintf("%s/audio_query?speaker=%d", VoicevoxURL, ve.config.VoicevoxSpeaker)
	queryURL += "&text=" + url.QueryEscape(text)

	resp, err := ve.httpClient.Post(queryURL, "application/json", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create audio query: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("audio query failed: status %d", resp.StatusCode)
	}

	queryData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read query response: %w", err)
	}

	// Synthesize audio
	synthURL := fmt.Sprintf("%s/synthesis?speaker=%d", VoicevoxURL, ve.config.VoicevoxSpeaker)

	resp, err = ve.httpClient.Post(synthURL, "application/json", bytes.NewReader(queryData))
	if err != nil {
		return "", fmt.Errorf("failed to synthesize: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("synthesis failed: status %d", resp.StatusCode)
	}

	// Save to temporary file
	tmpFile, err := os.CreateTemp("", "voice_*.wav")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		_ = tmpFile.Close()
	}()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to save audio: %w", err)
	}

	return tmpFile.Name(), nil
}

// synthesizeAivisSpeech uses AivisSpeech for synthesis
func (ve *VoiceEngine) synthesizeAivisSpeech(text string) (string, error) {
	// Create audio query (VOICEVOX compatible API)
	queryURL := fmt.Sprintf("%s/audio_query?speaker=%d", AivisSpeechURL, ve.config.AivisSpeechSpeaker)
	queryURL += "&text=" + url.QueryEscape(text)

	resp, err := ve.httpClient.Post(queryURL, "application/json", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create audio query: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("audio query failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	queryData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read query response: %w", err)
	}

	// Synthesize audio
	synthURL := fmt.Sprintf("%s/synthesis?speaker=%d", AivisSpeechURL, ve.config.AivisSpeechSpeaker)

	resp, err = ve.httpClient.Post(synthURL, "application/json", bytes.NewReader(queryData))
	if err != nil {
		return "", fmt.Errorf("failed to synthesize: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("synthesis failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Save to temporary file
	tmpFile, err := os.CreateTemp("", "voice_*.wav")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		_ = tmpFile.Close()
	}()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to save audio: %w", err)
	}

	return tmpFile.Name(), nil
}

// Play plays the audio file
func (ve *VoiceEngine) Play(audioFile string) error {
	// Detect the platform and use appropriate player
	var cmd *exec.Cmd

	switch {
	case isCommandAvailable("afplay"):
		// macOS
		cmd = exec.Command("afplay", audioFile)
	case isCommandAvailable("aplay"):
		// Linux with ALSA
		cmd = exec.Command("aplay", audioFile)
	case isCommandAvailable("paplay"):
		// Linux with PulseAudio
		cmd = exec.Command("paplay", audioFile)
	case isCommandAvailable("ffplay"):
		// Cross-platform with ffmpeg
		cmd = exec.Command("ffplay", "-nodisp", "-autoexit", audioFile)
	default:
		return fmt.Errorf("no audio player found")
	}

	// Start playing in background
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to play audio: %w", err)
	}

	// Clean up the file after a delay
	go func() {
		time.Sleep(10 * time.Second)
		_ = os.Remove(audioFile)
	}()

	return nil
}

// isCommandAvailable checks if a command is available
func isCommandAvailable(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// StripMarkdown removes markdown formatting using mdstrip if available
func StripMarkdown(text string) string {
	if !isCommandAvailable("mdstrip") {
		log.Debug().Msg("mdstrip not found, keeping markdown formatting")
		return text
	}

	cmd := exec.Command("mdstrip")
	cmd.Stdin = strings.NewReader(text)

	output, err := cmd.Output()
	if err != nil {
		log.Warn().Err(err).Msg("mdstrip failed, keeping original text")
		return text
	}

	log.Debug().Msg("Stripped markdown formatting")
	return string(output)
}

package mcp_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	internalmcp "github.com/daikw/ccpersona/internal/mcp"
	"github.com/daikw/ccpersona/internal/voice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSynthesizer records calls to Synthesize.
type mockSynthesizer struct {
	called     bool
	lastOpts   voice.VoiceOptions
	returnPath string
	returnErr  error
}

func (m *mockSynthesizer) Synthesize(_ context.Context, _ string, opts voice.VoiceOptions) (string, error) {
	m.called = true
	m.lastOpts = opts
	return m.returnPath, m.returnErr
}

// mockPlayer records calls to PlayAudioBlocking.
type mockPlayer struct {
	called    bool
	lastPath  string
	returnErr error
}

func (m *mockPlayer) PlayAudioBlocking(audioPath string) error {
	m.called = true
	m.lastPath = audioPath
	return m.returnErr
}

func TestSpeakService_Speak_HappyPath(t *testing.T) {
	synth := &mockSynthesizer{returnPath: "/tmp/voice_test.mp3"}
	player := &mockPlayer{}

	svc := internalmcp.NewSpeakService(synth, player)
	err := svc.Speak(context.Background(), internalmcp.SpeakRequest{
		Text: "こんにちは",
	})

	require.NoError(t, err)
	assert.True(t, synth.called, "Synthesize should be called")
	assert.True(t, player.called, "PlayAudioBlocking should be called")
	assert.Equal(t, "/tmp/voice_test.mp3", player.lastPath)
}

func TestSpeakService_Speak_EmptyText(t *testing.T) {
	synth := &mockSynthesizer{}
	player := &mockPlayer{}

	svc := internalmcp.NewSpeakService(synth, player)
	err := svc.Speak(context.Background(), internalmcp.SpeakRequest{
		Text: "",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "text cannot be empty")
	assert.False(t, synth.called, "Synthesize should NOT be called")
	assert.False(t, player.called, "PlayAudioBlocking should NOT be called")
}

func TestSpeakService_Speak_SynthesizeError(t *testing.T) {
	synthErr := errors.New("engine unavailable")
	synth := &mockSynthesizer{returnErr: synthErr}
	player := &mockPlayer{}

	svc := internalmcp.NewSpeakService(synth, player)
	err := svc.Speak(context.Background(), internalmcp.SpeakRequest{
		Text: "テスト",
	})

	require.Error(t, err)
	assert.ErrorContains(t, err, "synthesis failed")
	assert.True(t, synth.called, "Synthesize should be called")
	assert.False(t, player.called, "PlayAudioBlocking should NOT be called on synthesis error")
}

func TestSpeakService_Speak_PlaybackError(t *testing.T) {
	synth := &mockSynthesizer{returnPath: "/tmp/voice_test.mp3"}
	playErr := errors.New("audio device not found")
	player := &mockPlayer{returnErr: playErr}

	svc := internalmcp.NewSpeakService(synth, player)
	err := svc.Speak(context.Background(), internalmcp.SpeakRequest{
		Text: "テスト",
	})

	require.Error(t, err)
	assert.ErrorContains(t, err, "playback failed")
	assert.True(t, synth.called, "Synthesize should be called")
	assert.True(t, player.called, "PlayAudioBlocking should be called")
}

func TestSpeakService_Speak_WithProjectDir(t *testing.T) {
	synth := &mockSynthesizer{returnPath: "/tmp/voice_test.mp3"}
	player := &mockPlayer{}

	svc := internalmcp.NewSpeakService(synth, player)
	err := svc.Speak(context.Background(), internalmcp.SpeakRequest{
		Text:       "テスト",
		ProjectDir: t.TempDir(), // valid temp dir, no persona config expected
	})

	require.NoError(t, err)
	assert.True(t, synth.called)
	assert.True(t, player.called)
}

func TestSpeakService_Speak_WithProviderAndSpeaker(t *testing.T) {
	synth := &mockSynthesizer{returnPath: "/tmp/voice_test.mp3"}
	player := &mockPlayer{}

	svc := internalmcp.NewSpeakService(synth, player)
	err := svc.Speak(context.Background(), internalmcp.SpeakRequest{
		Text:       "テスト",
		Provider:   "aivisspeech",
		Speaker:    888753760,
		ProjectDir: t.TempDir(),
	})

	require.NoError(t, err)
	assert.True(t, synth.called)
	assert.True(t, player.called)
	// speaker が VoiceOptions に正しく反映されていることを確認
	assert.Equal(t, 888753760, synth.lastOpts.VoicevoxSpeaker)
	assert.Equal(t, 888753760, synth.lastOpts.AivisSpeechSpeaker)
	assert.Equal(t, "", synth.lastOpts.Voice, "cloud voice should be cleared when speaker is set")
}

func TestSpeakService_Speak_WithPersonaVoiceConfig(t *testing.T) {
	// Write a persona config with voice settings to the temp project dir.
	projectDir := t.TempDir()
	claudeDir := filepath.Join(projectDir, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))
	personaJSON := `{"name":"test","voice":{"provider":"aivisspeech","speaker":42}}`
	require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "persona.json"), []byte(personaJSON), 0644))

	synth := &mockSynthesizer{returnPath: "/tmp/voice_test.mp3"}
	player := &mockPlayer{}

	svc := internalmcp.NewSpeakService(synth, player)
	err := svc.Speak(context.Background(), internalmcp.SpeakRequest{
		Text:       "ペルソナテスト",
		ProjectDir: projectDir,
	})

	require.NoError(t, err)
	assert.True(t, synth.called)
	assert.True(t, player.called)
}

func TestSpeakService_Speak_EmptyAudioPath(t *testing.T) {
	// Synthesize returns an empty path (e.g., ToStdout case) — playback should still be attempted.
	synth := &mockSynthesizer{returnPath: ""}
	player := &mockPlayer{}

	svc := internalmcp.NewSpeakService(synth, player)
	err := svc.Speak(context.Background(), internalmcp.SpeakRequest{
		Text:       "テスト",
		ProjectDir: t.TempDir(),
	})

	require.NoError(t, err)
	assert.True(t, synth.called)
	assert.True(t, player.called)
	assert.Equal(t, "", player.lastPath)
}

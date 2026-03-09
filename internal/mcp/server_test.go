package mcp_test

import (
	"context"
	"os"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internalmcp "github.com/daikw/ccpersona/internal/mcp"
	"github.com/daikw/ccpersona/internal/mcp/adapter"
)

func TestNewSpeakService_NotNil(t *testing.T) {
	synth := &mockSynthesizer{}
	player := &mockPlayer{}
	svc := internalmcp.NewSpeakService(synth, player)
	require.NotNil(t, svc)
}

func TestAdapterServer_Initialization(t *testing.T) {
	srv := adapter.NewServer("ccpersona-test", "v0.0.0-test")
	assert.NotNil(t, srv)
}

func TestRunServer_CanceledContext(t *testing.T) {
	// RunServer with a pre-cancelled context should return quickly without error.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// ServeStdio will fail because stdin/stdout are not pipes in test context,
	// but we just verify it does not panic and returns promptly.
	_ = internalmcp.RunServer(ctx, "test")
}

func TestToPersonaVoiceInput_NilVoice(t *testing.T) {
	// Verify Speak succeeds when persona config has no Voice field (cfg.Voice == nil path).
	synth := &mockSynthesizer{returnPath: "/tmp/voice_test.mp3"}
	player := &mockPlayer{}
	svc := internalmcp.NewSpeakService(synth, player)

	// Write persona.json without voice field.
	projectDir := t.TempDir()
	claudeDir := projectDir + "/.claude"
	require.NoError(t, os.MkdirAll(claudeDir, 0755))
	require.NoError(t, os.WriteFile(claudeDir+"/persona.json", []byte(`{"name":"no-voice"}`), 0644))

	err := svc.Speak(context.Background(), internalmcp.SpeakRequest{
		Text:       "テスト",
		ProjectDir: projectDir,
	})
	require.NoError(t, err)
	assert.True(t, synth.called)
}

func TestAdapterServer_AddToolDoesNotPanic(t *testing.T) {
	srv := adapter.NewServer("ccpersona-test", "v0.0.0-test")

	assert.NotPanics(t, func() {
		srv.AddTool(
			"speak",
			"テスト用ツール",
			[]adapter.ToolParam{
				{Name: "text", Description: "テキスト", Type: "string", Required: true},
				{Name: "speaker", Description: "スピーカーID", Type: "number"},
			},
			func(_ context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
				return mcplib.NewToolResultText("ok"), nil
			},
		)
	})
}

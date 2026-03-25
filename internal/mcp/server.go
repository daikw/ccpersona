// Package mcp provides a stdio MCP server for ccpersona voice synthesis.
package mcp

import (
	"context"
	"time"

	"github.com/daikw/ccpersona/internal/mcp/adapter"
	"github.com/daikw/ccpersona/internal/voice"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rs/zerolog/log"
)

// RunServer starts the MCP server and blocks until ctx is cancelled.
func RunServer(ctx context.Context, version string) error {
	srv := adapter.NewServer("ccpersona", version)

	voiceConfig := voice.DefaultConfig()
	manager := voice.NewVoiceManager(voiceConfig)

	speakSvc := NewSpeakService(manager, manager)

	// Periodically clean up stale temp audio files.
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := manager.CleanupTempFiles(24 * time.Hour); err != nil {
					log.Warn().Err(err).Msg("CleanupTempFiles failed")
				}
			}
		}
	}()

	srv.AddTool(
		"speak",
		"Synthesize text to speech. Always pass the current working directory as project_dir (e.g., /Users/you/my-project). This applies per-project persona and voice settings.",
		[]adapter.ToolParam{
			{Name: "text", Description: "Text to synthesize", Type: "string", Required: true},
			{Name: "project_dir", Description: "Absolute path to the current working directory. Used to load voice settings (provider, speaker, speed, volume) from the project's .claude/persona.json.", Type: "string", Required: true},
			{Name: "provider", Description: "TTS provider (voicevox / aivisspeech / openai / elevenlabs, etc.)", Type: "string"},
			{Name: "speaker", Description: "Speaker ID (for local engines)", Type: "number"},
		},
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			text := mcp.ParseString(req, "text", "")
			provider := mcp.ParseString(req, "provider", "")
			speaker := mcp.ParseInt(req, "speaker", 0)
			projectDir := mcp.ParseString(req, "project_dir", "")

			log.Debug().
				Int("text_len", len(text)).
				Str("provider", provider).
				Int("speaker", speaker).
				Str("project_dir", projectDir).
				Msg("speak tool called")

			if err := speakSvc.Speak(ctx, SpeakRequest{
				Text:       text,
				Provider:   provider,
				Speaker:    speaker,
				ProjectDir: projectDir,
			}); err != nil {
				return mcp.NewToolResultText("error: " + err.Error()), err
			}

			return mcp.NewToolResultText("ok"), nil
		},
	)

	log.Info().Str("version", version).Msg("ccpersona MCP server starting (stdio)")
	return srv.ServeStdio(ctx)
}

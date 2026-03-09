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
		"テキストを音声合成して読み上げる",
		[]adapter.ToolParam{
			{Name: "text", Description: "読み上げるテキスト", Type: "string", Required: true},
			{Name: "provider", Description: "TTSプロバイダー (voicevox / aivisspeech / openai / elevenlabs など)", Type: "string"},
			{Name: "speaker", Description: "スピーカーID（ローカルエンジン用）", Type: "number"},
			{Name: "project_dir", Description: "persona/voice 設定を解決する起点ディレクトリ（未指定時は cwd）", Type: "string"},
		},
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			text := mcp.ParseString(req, "text", "")
			provider := mcp.ParseString(req, "provider", "")
			speaker := mcp.ParseInt(req, "speaker", 0)
			projectDir := mcp.ParseString(req, "project_dir", "")

			log.Debug().
				Str("text", text).
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

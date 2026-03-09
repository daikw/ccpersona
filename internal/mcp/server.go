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
		"テキストを音声合成して読み上げる。必ず現在の作業ディレクトリを project_dir に渡すこと（例: /Users/you/my-project）。これによりプロジェクトごとのペルソナ・音声設定が適用される。",
		[]adapter.ToolParam{
			{Name: "text", Description: "読み上げるテキスト", Type: "string", Required: true},
			{Name: "project_dir", Description: "現在の作業ディレクトリの絶対パス。プロジェクトの .claude/persona.json から音声設定（プロバイダー・スピーカー・速度・音量）を読み込むために使用する。", Type: "string", Required: true},
			{Name: "provider", Description: "TTSプロバイダー (voicevox / aivisspeech / openai / elevenlabs など)", Type: "string"},
			{Name: "speaker", Description: "スピーカーID（ローカルエンジン用）", Type: "number"},
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

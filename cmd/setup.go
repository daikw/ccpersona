package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/daikw/ccpersona/internal/persona"
	"github.com/daikw/ccpersona/internal/voice"
	"github.com/urfave/cli/v3"
)

func handleSetup(ctx context.Context, c *cli.Command) error {
	// Deprecated: use 'status' instead (with --diagnose for details)
	fmt.Fprintln(os.Stderr, "⚠️  'setup' is deprecated. Use 'status --diagnose' instead.")
	return handleStatusWithDiagnose(ctx, c, true)
}

func handleStatus(ctx context.Context, c *cli.Command) error {
	forceDiagnose := c.Bool("diagnose")
	return handleStatusWithDiagnose(ctx, c, forceDiagnose)
}

func handleStatusWithDiagnose(ctx context.Context, c *cli.Command, forceDiagnose bool) error {
	issues := 0
	warnings := 0

	// Get current directory
	cwd, _ := os.Getwd()
	fmt.Printf("📍 現在のディレクトリ: %s\n", cwd)

	// Check project persona
	projectConfig, _ := persona.LoadConfig(".")
	if projectConfig != nil {
		fmt.Printf("🎭 プロジェクトペルソナ: %s\n", projectConfig.Name)
		if projectConfig.Voice != nil {
			fmt.Printf("🔊 音声プロバイダー: %s\n", projectConfig.Voice.Provider)
			fmt.Printf("🎤 Speaker: %d\n", projectConfig.Voice.Speaker)
		}
	} else {
		fmt.Println("🎭 プロジェクトペルソナ: (未設定)")
		warnings++
	}

	// Check voice engine status
	voiceConfig := voice.DefaultConfig()
	engine := voice.NewVoiceEngine(voiceConfig)
	voicevoxAvail, aivisAvail := engine.CheckEngines()

	if aivisAvail {
		fmt.Println("🔊 AivisSpeech: 接続OK")
	} else {
		issues++
	}
	if voicevoxAvail {
		fmt.Println("🔊 VOICEVOX: 接続OK")
	}
	if !aivisAvail && !voicevoxAvail {
		fmt.Println("🔊 音声エンジン: 未接続")
	}

	// Check persona manager
	manager, err := persona.NewManager()
	if err != nil {
		issues++
	} else {
		personas, _ := manager.ListPersonas()
		if len(personas) == 0 {
			warnings++
		}
	}

	// Check Claude Code settings
	homeDir, _ := os.UserHomeDir()
	settingsPath := filepath.Join(homeDir, ".claude", "settings.json")
	if _, err := os.Stat(settingsPath); err != nil {
		warnings++
	}

	// Auto-diagnose if there are issues/warnings, or if forced
	if forceDiagnose || issues > 0 || warnings > 0 {
		fmt.Println("")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("🔍 診断情報")
		fmt.Println("")

		// Version info
		fmt.Printf("✅ ccpersona バージョン: %s (%s)\n", version, revision)

		// Personas
		if manager != nil {
			personas, _ := manager.ListPersonas()
			if len(personas) > 0 {
				fmt.Printf("✅ ペルソナ: %d 個\n", len(personas))
			} else {
				fmt.Println("⚠️  ペルソナ: 未作成")
			}
		}

		// Voice engines detail
		if aivisAvail {
			fmt.Println("✅ AivisSpeech: 接続OK (127.0.0.1:10101)")
		} else {
			fmt.Println("❌ AivisSpeech: 接続できません (127.0.0.1:10101)")
		}
		if voicevoxAvail {
			fmt.Println("✅ VOICEVOX: 接続OK (127.0.0.1:50021)")
		} else {
			fmt.Println("⚠️  VOICEVOX: 接続できません (127.0.0.1:50021)")
		}

		// Claude Code settings
		if _, err := os.Stat(settingsPath); err == nil {
			fmt.Println("✅ Claude Code設定: 検出")
		} else {
			fmt.Println("⚠️  Claude Code設定: 見つかりません")
		}

		// Summary and recommendations
		if issues > 0 || warnings > 0 {
			fmt.Println("")
			fmt.Println("推奨アクション:")
			if !aivisAvail && !voicevoxAvail {
				fmt.Println("  - AivisSpeech または VOICEVOX を起動してください")
			}
			if projectConfig == nil {
				fmt.Println("  - 'ccpersona init' でプロジェクトを初期化してください")
			}
			if manager != nil {
				personas, _ := manager.ListPersonas()
				if len(personas) == 0 {
					fmt.Println("  - 'ccpersona edit <name>' でペルソナを作成してください")
				}
			}
		} else {
			fmt.Println("")
			fmt.Println("✅ すべてのチェックに成功しました！")
		}
	}

	return nil
}

func handleDoctor(ctx context.Context, c *cli.Command) error {
	// Deprecated: use 'status --diagnose' instead
	fmt.Fprintln(os.Stderr, "⚠️  'doctor' is deprecated. Use 'status --diagnose' instead.")
	return handleStatusWithDiagnose(ctx, c, true)
}

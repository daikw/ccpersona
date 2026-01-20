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
	fmt.Println("🎭 ccpersona セットアップウィザード")
	fmt.Println("")

	// Step 1: Check voice engines
	fmt.Println("📡 音声エンジンを検出中...")
	voiceConfig := voice.DefaultConfig()
	engine := voice.NewVoiceEngine(voiceConfig)
	voicevoxAvail, aivisAvail := engine.CheckEngines()

	if aivisAvail {
		fmt.Println("  ✅ AivisSpeech が検出されました (127.0.0.1:10101)")
	} else {
		fmt.Println("  ❌ AivisSpeech は検出されませんでした")
	}
	if voicevoxAvail {
		fmt.Println("  ✅ VOICEVOX が検出されました (127.0.0.1:50021)")
	} else {
		fmt.Println("  ❌ VOICEVOX は検出されませんでした")
	}
	fmt.Println("")

	// Step 2: List available personas
	fmt.Println("🎭 利用可能なペルソナ:")
	manager, err := persona.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create persona manager: %w", err)
	}

	personas, err := manager.ListPersonas()
	if err != nil {
		return fmt.Errorf("failed to list personas: %w", err)
	}

	if len(personas) == 0 {
		fmt.Println("  (ペルソナがありません)")
		fmt.Println("  → 'ccpersona create <name>' で作成できます")
	} else {
		for i, p := range personas {
			fmt.Printf("  %d. %s\n", i+1, p)
		}
	}
	fmt.Println("")

	// Step 3: Check current project configuration
	fmt.Println("📁 プロジェクト設定:")
	projectConfig, _ := persona.LoadConfig(".")
	if projectConfig != nil {
		fmt.Printf("  ✅ 設定済み: %s\n", projectConfig.Name)
	} else {
		fmt.Println("  ❌ 未設定")
		fmt.Println("  → 'ccpersona init' でプロジェクトを初期化できます")
	}
	fmt.Println("")

	// Step 4: Check Claude Code hooks (if not skipped)
	if !c.Bool("skip-hooks") {
		fmt.Println("🔗 Claude Code hooks:")
		homeDir, _ := os.UserHomeDir()
		settingsPath := filepath.Join(homeDir, ".claude", "settings.json")

		if _, err := os.Stat(settingsPath); err == nil {
			fmt.Printf("  📄 設定ファイル: %s\n", settingsPath)
			fmt.Println("  → hooks設定は手動で確認してください")
			fmt.Println("")
			fmt.Println("  推奨設定:")
			fmt.Println("  {")
			fmt.Println("    \"hooks\": {")
			fmt.Println("      \"session-start\": [\"ccpersona hook\"],")
			fmt.Println("      \"stop\": [\"ccpersona voice\"]")
			fmt.Println("    }")
			fmt.Println("  }")
		} else {
			fmt.Println("  ❌ Claude Code設定ファイルが見つかりません")
			fmt.Println("  → Claude Codeを起動して設定を作成してください")
		}
		fmt.Println("")
	}

	// Summary
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ セットアップ確認が完了しました！")
	fmt.Println("")
	fmt.Println("次のステップ:")
	if !aivisAvail && !voicevoxAvail {
		fmt.Println("  1. AivisSpeech または VOICEVOX を起動")
	}
	if len(personas) == 0 {
		fmt.Println("  2. 'ccpersona create ずんだもん' でペルソナを作成")
	}
	if projectConfig == nil {
		fmt.Println("  3. 'ccpersona init' でプロジェクトを初期化")
	}
	fmt.Println("  4. Claude Codeを再起動して新しいセッションを開始")

	return nil
}

func handleStatus(ctx context.Context, c *cli.Command) error {
	// Get current directory
	cwd, _ := os.Getwd()
	fmt.Printf("📍 現在のディレクトリ: %s\n", cwd)

	// Check project persona
	projectConfig, _ := persona.LoadConfig(".")
	if projectConfig != nil {
		fmt.Printf("🎭 プロジェクトペルソナ: %s\n", projectConfig.Name)
		if projectConfig.Voice != nil {
			fmt.Printf("🔊 音声エンジン: %s\n", projectConfig.Voice.Engine)
			fmt.Printf("🎤 Speaker ID: %d\n", projectConfig.Voice.SpeakerID)
		}
	} else {
		fmt.Println("🎭 プロジェクトペルソナ: (未設定)")
	}

	// Check voice engine status
	voiceConfig := voice.DefaultConfig()
	engine := voice.NewVoiceEngine(voiceConfig)
	voicevoxAvail, aivisAvail := engine.CheckEngines()

	if aivisAvail {
		fmt.Println("🔊 AivisSpeech: 接続OK")
	}
	if voicevoxAvail {
		fmt.Println("🔊 VOICEVOX: 接続OK")
	}
	if !aivisAvail && !voicevoxAvail {
		fmt.Println("🔊 音声エンジン: 未接続")
	}

	return nil
}

func handleDoctor(ctx context.Context, c *cli.Command) error {
	fmt.Println("🔍 診断を実行中...")
	fmt.Println("")

	issues := 0
	warnings := 0

	// Check version
	fmt.Printf("✅ ccpersona バージョン: %s (%s)\n", version, revision)

	// Check personas directory
	manager, err := persona.NewManager()
	if err != nil {
		fmt.Printf("❌ ペルソナマネージャーの初期化に失敗: %v\n", err)
		issues++
	} else {
		personas, _ := manager.ListPersonas()
		if len(personas) > 0 {
			fmt.Printf("✅ ペルソナディレクトリ: %d 個のペルソナ\n", len(personas))
		} else {
			fmt.Println("⚠️  ペルソナディレクトリ: ペルソナがありません")
			warnings++
		}
	}

	// Check voice engines
	voiceConfig := voice.DefaultConfig()
	engine := voice.NewVoiceEngine(voiceConfig)
	voicevoxAvail, aivisAvail := engine.CheckEngines()

	if aivisAvail {
		fmt.Println("✅ AivisSpeech: 接続OK (127.0.0.1:10101)")
	} else {
		fmt.Println("❌ AivisSpeech: 接続できません (127.0.0.1:10101)")
		issues++
	}

	if voicevoxAvail {
		fmt.Println("✅ VOICEVOX: 接続OK (127.0.0.1:50021)")
	} else {
		fmt.Println("⚠️  VOICEVOX: 接続できません (127.0.0.1:50021)")
		warnings++
	}

	// Check project configuration
	projectConfig, _ := persona.LoadConfig(".")
	if projectConfig != nil {
		fmt.Printf("✅ プロジェクト設定: %s\n", projectConfig.Name)
	} else {
		fmt.Println("⚠️  プロジェクト設定: 未設定")
		warnings++
	}

	// Check Claude Code settings
	homeDir, _ := os.UserHomeDir()
	settingsPath := filepath.Join(homeDir, ".claude", "settings.json")
	if _, err := os.Stat(settingsPath); err == nil {
		fmt.Println("✅ Claude Code設定: 検出")
	} else {
		fmt.Println("⚠️  Claude Code設定: 見つかりません")
		warnings++
	}

	// Summary
	fmt.Println("")
	if issues == 0 && warnings == 0 {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("✅ すべてのチェックに成功しました！")
	} else {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		if issues > 0 {
			fmt.Printf("❌ %d 件の問題があります\n", issues)
		}
		if warnings > 0 {
			fmt.Printf("⚠️  %d 件の警告があります\n", warnings)
		}
		fmt.Println("")
		fmt.Println("推奨アクション:")
		if !aivisAvail && !voicevoxAvail {
			fmt.Println("  - AivisSpeech または VOICEVOX を起動してください")
		}
		if projectConfig == nil {
			fmt.Println("  - 'ccpersona init' でプロジェクトを初期化してください")
		}
	}

	return nil
}

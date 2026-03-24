package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/daikw/ccpersona/internal/engine"
	"github.com/daikw/ccpersona/internal/persona"
	"github.com/daikw/ccpersona/internal/voice"
	"github.com/urfave/cli/v3"
)

func handleSetup(ctx context.Context, c *cli.Command) error {
	fmt.Println("🔧 ccpersona セットアップ")
	fmt.Println("")

	// Run diagnostics first
	if err := handleStatusWithDiagnose(ctx, c, true); err != nil {
		return err
	}

	// Engine service setup
	fmt.Println("")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔊 音声エンジン サービス設定")
	fmt.Println("")

	mgr, err := engine.NewServiceManager()
	if err != nil {
		fmt.Printf("  サービスマネージャ初期化失敗: %v\n", err)
		return nil
	}

	for _, t := range engine.AllEngineTypes() {
		info, discoverErr := engine.DiscoverEngine(t)
		if discoverErr != nil {
			fmt.Printf("  %s: バイナリ未検出 (スキップ)\n", t)
			continue
		}

		status, _ := mgr.Status(t)
		if status != nil && status.Installed {
			runStatus := "停止"
			if status.Running {
				runStatus = fmt.Sprintf("稼働中 (PID: %d)", status.PID)
			}
			fmt.Printf("  %s: インストール済み [%s]\n", t, runStatus)
			continue
		}

		fmt.Printf("  %s: バイナリ検出 → %s\n", t, info.BinaryPath)
		fmt.Printf("  %s: サービス未インストール → 'ccpersona engine install %s' で登録できます\n", t, t)
	}

	return nil
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
	voiceEngine := voice.NewVoiceEngine(voiceConfig)
	voicevoxAvail, aivisAvail := voiceEngine.CheckEngines()

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

		// Engine service status
		if mgr, mgrErr := engine.NewServiceManager(); mgrErr == nil {
			for _, t := range engine.AllEngineTypes() {
				svcStatus, _ := mgr.Status(t)
				if svcStatus == nil {
					continue
				}
				if svcStatus.Installed {
					if svcStatus.Running {
						fmt.Printf("✅ %s サービス: 稼働中 (PID: %d)\n", t, svcStatus.PID)
					} else {
						fmt.Printf("⚠️  %s サービス: インストール済み・停止中\n", t)
					}
				} else {
					if _, err := engine.DiscoverEngine(t); err == nil {
						fmt.Printf("⚠️  %s サービス: 未インストール (バイナリ検出済み)\n", t)
					}
				}
			}
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
				fmt.Println("  - 'ccpersona engine install all' でサービスをインストールしてください")
				fmt.Println("  - または AivisSpeech / VOICEVOX を手動で起動してください")
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

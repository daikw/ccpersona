package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/daikw/ccpersona/internal/cliui"
	"github.com/daikw/ccpersona/internal/engine"
	"github.com/daikw/ccpersona/internal/persona"
	"github.com/daikw/ccpersona/internal/voice"
	"github.com/urfave/cli/v3"
)

func handleSetup(ctx context.Context, c *cli.Command) error {
	fmt.Println(cliui.Header("ccpersona setup"))
	fmt.Println("")

	// Run diagnostics first
	if err := handleStatusWithDiagnose(ctx, c, true); err != nil {
		return err
	}

	// Engine service setup
	fmt.Println("")
	fmt.Println("----------------------------------------")
	fmt.Println(cliui.Header("Voice Engine Services"))
	fmt.Println("")

	mgr, err := engine.NewServiceManager()
	if err != nil {
		fmt.Printf("  %s: %v\n", cliui.Failure("failed to init service manager"), err)
		return nil
	}

	for _, t := range engine.AllEngineTypes() {
		info, discoverErr := engine.DiscoverEngine(t)
		if discoverErr != nil {
			fmt.Printf("  %s: %s\n", cliui.Label(t), cliui.Failure("binary not found"))
			fmt.Printf("  %s: install the app first (https://github.com/daikw/ccpersona#voice-engines)\n", cliui.Label(t))
			continue
		}

		status, _ := mgr.Status(t)
		if status != nil && status.Installed {
			runStatus := cliui.Warn("stopped")
			if status.Running {
				runStatus = fmt.Sprintf("%s (PID: %d)", cliui.Success("running"), status.PID)
			}
			fmt.Printf("  %s: %s [%s]\n", cliui.Label(t), cliui.Success("installed"), runStatus)
			continue
		}

		fmt.Printf("  %s: binary found -> %s\n", cliui.Label(t), cliui.Muted(info.BinaryPath))
		fmt.Printf("  %s: %s -> run 'ccpersona engine install %s'\n", cliui.Label(t), cliui.Warn("not installed"), t)
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
	fmt.Printf("%s %s\n", cliui.Label("Directory:"), cwd)

	// Check project persona
	projectConfig, _ := persona.LoadConfig(".")
	if projectConfig != nil {
		fmt.Printf("%s %s\n", cliui.Label("Persona:"), projectConfig.Name)
		if projectConfig.Voice != nil {
			fmt.Printf("%s %s\n", cliui.Label("Voice provider:"), projectConfig.Voice.Provider)
			fmt.Printf("%s %d\n", cliui.Label("Speaker:"), projectConfig.Voice.Speaker)
		}
	} else {
		fmt.Printf("%s %s\n", cliui.Label("Persona:"), cliui.Warn("(not configured)"))
		warnings++
	}

	// Check voice engine status
	voiceConfig := voice.DefaultConfig()
	voiceEngine := voice.NewVoiceEngine(voiceConfig)
	voicevoxAvail, aivisAvail := voiceEngine.CheckEngines()

	if aivisAvail {
		fmt.Printf("%s %s\n", cliui.Label("AivisSpeech:"), cliui.Success("connected"))
	} else {
		issues++
	}
	if voicevoxAvail {
		fmt.Printf("%s %s\n", cliui.Label("VOICEVOX:"), cliui.Success("connected"))
	}
	if !aivisAvail && !voicevoxAvail {
		fmt.Printf("%s %s\n", cliui.Label("Voice engine:"), cliui.Failure("not connected"))
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
		fmt.Println("----------------------------------------")
		fmt.Println(cliui.Header("Diagnostics"))
		fmt.Println("")

		// Version info
		fmt.Printf("  %s %s (%s)\n", cliui.Label("ccpersona version:"), version, revision)

		// Personas
		if manager != nil {
			personas, _ := manager.ListPersonas()
			if len(personas) > 0 {
				fmt.Printf("  %s %s\n", cliui.Label("personas:"), cliui.Success(fmt.Sprintf("%d found", len(personas))))
			} else {
				fmt.Printf("  %s %s\n", cliui.Label("personas:"), cliui.Warn("none"))
			}
		}

		// Voice engines detail
		if aivisAvail {
			fmt.Printf("  %s %s\n", cliui.Label("AivisSpeech:"), cliui.Success("connected (127.0.0.1:10101)"))
		} else {
			fmt.Printf("  %s %s\n", cliui.Label("AivisSpeech:"), cliui.Failure("not reachable (127.0.0.1:10101)"))
		}
		if voicevoxAvail {
			fmt.Printf("  %s %s\n", cliui.Label("VOICEVOX:"), cliui.Success("connected (127.0.0.1:50021)"))
		} else {
			fmt.Printf("  %s %s\n", cliui.Label("VOICEVOX:"), cliui.Failure("not reachable (127.0.0.1:50021)"))
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
						fmt.Printf("  %s %s (PID: %d)\n", cliui.Label(fmt.Sprintf("%s service:", t)), cliui.Success("running"), svcStatus.PID)
					} else {
						fmt.Printf("  %s %s\n", cliui.Label(fmt.Sprintf("%s service:", t)), cliui.Warn("installed, stopped"))
					}
				} else {
					if _, err := engine.DiscoverEngine(t); err == nil {
						fmt.Printf("  %s %s\n", cliui.Label(fmt.Sprintf("%s service:", t)), cliui.Warn("not installed (binary available)"))
					}
				}
			}
		}

		// Claude Code settings
		if _, err := os.Stat(settingsPath); err == nil {
			fmt.Printf("  %s %s\n", cliui.Label("Claude Code settings:"), cliui.Success("found"))
		} else {
			fmt.Printf("  %s %s\n", cliui.Label("Claude Code settings:"), cliui.Warn("not found"))
		}

		// Summary and recommendations
		if issues > 0 || warnings > 0 {
			fmt.Println("")
			fmt.Println(cliui.Header("Recommended actions:"))
			if !aivisAvail && !voicevoxAvail {
				// Check if engine binaries exist at all
				anyBinaryFound := false
				for _, t := range engine.AllEngineTypes() {
					if _, err := engine.DiscoverEngine(t); err == nil {
						anyBinaryFound = true
						break
					}
				}
				if anyBinaryFound {
					fmt.Println("  - run 'ccpersona engine install all' to install engine services")
					fmt.Println("  - or start AivisSpeech / VOICEVOX manually")
				} else {
					fmt.Println("  - install VOICEVOX or AivisSpeech app first:")
					fmt.Printf("      VOICEVOX:     %s\n", cliui.Muted("https://voicevox.hiroshiba.jp/"))
					fmt.Printf("      AivisSpeech:  %s\n", cliui.Muted("https://aivis-project.com/"))
					fmt.Println("  - then run 'ccpersona engine install all'")
				}
			}
			if projectConfig == nil {
				fmt.Println("  - run 'ccpersona init' to initialize project")
			}
			if manager != nil {
				personas, _ := manager.ListPersonas()
				if len(personas) == 0 {
					fmt.Println("  - run 'ccpersona edit <name>' to create a persona")
				}
			}
		} else {
			fmt.Println("")
			fmt.Println(cliui.Success("All checks passed."))
		}
	}

	return nil
}

func handleDoctor(ctx context.Context, c *cli.Command) error {
	// Deprecated: use 'status --diagnose' instead
	fmt.Fprintln(os.Stderr, "'doctor' is deprecated. Use 'status --diagnose' instead.")
	return handleStatusWithDiagnose(ctx, c, true)
}

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
	fmt.Println("ccpersona setup")
	fmt.Println("")

	// Run diagnostics first
	if err := handleStatusWithDiagnose(ctx, c, true); err != nil {
		return err
	}

	// Engine service setup
	fmt.Println("")
	fmt.Println("----------------------------------------")
	fmt.Println("Voice Engine Services")
	fmt.Println("")

	mgr, err := engine.NewServiceManager()
	if err != nil {
		fmt.Printf("  failed to init service manager: %v\n", err)
		return nil
	}

	for _, t := range engine.AllEngineTypes() {
		info, discoverErr := engine.DiscoverEngine(t)
		if discoverErr != nil {
			fmt.Printf("  %s: binary not found (skipped)\n", t)
			continue
		}

		status, _ := mgr.Status(t)
		if status != nil && status.Installed {
			runStatus := "stopped"
			if status.Running {
				runStatus = fmt.Sprintf("running (PID: %d)", status.PID)
			}
			fmt.Printf("  %s: installed [%s]\n", t, runStatus)
			continue
		}

		fmt.Printf("  %s: binary found -> %s\n", t, info.BinaryPath)
		fmt.Printf("  %s: not installed -> run 'ccpersona engine install %s'\n", t, t)
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
	fmt.Printf("Directory: %s\n", cwd)

	// Check project persona
	projectConfig, _ := persona.LoadConfig(".")
	if projectConfig != nil {
		fmt.Printf("Persona: %s\n", projectConfig.Name)
		if projectConfig.Voice != nil {
			fmt.Printf("Voice provider: %s\n", projectConfig.Voice.Provider)
			fmt.Printf("Speaker: %d\n", projectConfig.Voice.Speaker)
		}
	} else {
		fmt.Println("Persona: (not configured)")
		warnings++
	}

	// Check voice engine status
	voiceConfig := voice.DefaultConfig()
	voiceEngine := voice.NewVoiceEngine(voiceConfig)
	voicevoxAvail, aivisAvail := voiceEngine.CheckEngines()

	if aivisAvail {
		fmt.Println("AivisSpeech: connected")
	} else {
		issues++
	}
	if voicevoxAvail {
		fmt.Println("VOICEVOX: connected")
	}
	if !aivisAvail && !voicevoxAvail {
		fmt.Println("Voice engine: not connected")
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
		fmt.Println("Diagnostics")
		fmt.Println("")

		// Version info
		fmt.Printf("  ccpersona version: %s (%s)\n", version, revision)

		// Personas
		if manager != nil {
			personas, _ := manager.ListPersonas()
			if len(personas) > 0 {
				fmt.Printf("  personas: %d found\n", len(personas))
			} else {
				fmt.Println("  personas: none")
			}
		}

		// Voice engines detail
		if aivisAvail {
			fmt.Println("  AivisSpeech: connected (127.0.0.1:10101)")
		} else {
			fmt.Println("  AivisSpeech: not reachable (127.0.0.1:10101)")
		}
		if voicevoxAvail {
			fmt.Println("  VOICEVOX: connected (127.0.0.1:50021)")
		} else {
			fmt.Println("  VOICEVOX: not reachable (127.0.0.1:50021)")
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
						fmt.Printf("  %s service: running (PID: %d)\n", t, svcStatus.PID)
					} else {
						fmt.Printf("  %s service: installed, stopped\n", t)
					}
				} else {
					if _, err := engine.DiscoverEngine(t); err == nil {
						fmt.Printf("  %s service: not installed (binary available)\n", t)
					}
				}
			}
		}

		// Claude Code settings
		if _, err := os.Stat(settingsPath); err == nil {
			fmt.Println("  Claude Code settings: found")
		} else {
			fmt.Println("  Claude Code settings: not found")
		}

		// Summary and recommendations
		if issues > 0 || warnings > 0 {
			fmt.Println("")
			fmt.Println("Recommended actions:")
			if !aivisAvail && !voicevoxAvail {
				fmt.Println("  - run 'ccpersona engine install all' to install engine services")
				fmt.Println("  - or start AivisSpeech / VOICEVOX manually")
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
			fmt.Println("All checks passed.")
		}
	}

	return nil
}

func handleDoctor(ctx context.Context, c *cli.Command) error {
	// Deprecated: use 'status --diagnose' instead
	fmt.Fprintln(os.Stderr, "'doctor' is deprecated. Use 'status --diagnose' instead.")
	return handleStatusWithDiagnose(ctx, c, true)
}

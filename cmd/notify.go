package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/daikw/ccpersona/internal/hook"
	"github.com/daikw/ccpersona/internal/persona"
	"github.com/daikw/ccpersona/internal/voice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

func handleNotify(ctx context.Context, c *cli.Command) error {
	// Check for debug mode
	debug := os.Getenv("CCPERSONA_DEBUG") != ""
	if !debug {
		// Suppress normal output when running as hook
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}

	var unifiedEvent *hook.UnifiedHookEvent
	var err error

	// Codex passes JSON as command line argument, Claude Code uses stdin
	// Check for JSON argument first (Codex style)
	args := c.Args().Slice()
	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] args: %v\n", args)
	}
	if len(args) > 0 && strings.HasPrefix(strings.TrimSpace(args[0]), "{") {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Parsing from argument (Codex style)\n")
		}
		unifiedEvent, err = hook.DetectAndParse(bytes.NewReader([]byte(args[0])))
	} else {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Parsing from stdin (Claude Code style)\n")
		}
		unifiedEvent, err = hook.DetectAndParse(os.Stdin)
	}

	if err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Parse error: %v\n", err)
		}
		return handleLegacyNotification(ctx, c)
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Event: source=%s, type=%s\n", unifiedEvent.Source, unifiedEvent.EventType)
	}

	log.Debug().
		Str("source", unifiedEvent.Source).
		Str("session_id", unifiedEvent.SessionID).
		Str("event_type", unifiedEvent.EventType).
		Msg("Received hook event")

	// Handle based on event source and type
	if unifiedEvent.IsCodex() {
		// Codex notify hook - triggered on agent-turn-complete
		return handleCodexAgentTurnComplete(ctx, c, unifiedEvent)
	} else if unifiedEvent.IsClaudeCode() {
		// Claude Code events - route to appropriate handler
		switch unifiedEvent.EventType {
		case "UserPromptSubmit":
			// Apply persona at session start
			if err := persona.HandleSessionStart(); err != nil {
				log.Error().Err(err).Msg("Failed to handle session start")
			}
			return nil
		case "Stop", "SubagentStop":
			// Voice synthesis for assistant response
			return handleVoiceSynthesisForEvent(ctx, c, unifiedEvent)
		case "Notification":
			// Desktop and voice notification
			return handleNotificationEvent(ctx, c, unifiedEvent)
		default:
			log.Debug().Str("event_type", unifiedEvent.EventType).Msg("Unhandled event type")
			return nil
		}
	}

	return nil
}

func handleLegacyNotification(ctx context.Context, c *cli.Command) error {
	// Legacy notification format (simple JSON with message field)
	// This is kept for backward compatibility
	log.Debug().Msg("Using legacy notification format")

	// For now, just log that we couldn't process the event
	// In practice, this path should rarely be hit
	log.Warn().Msg("Could not parse notification event in any supported format")
	return nil
}

func handleCodexAgentTurnComplete(ctx context.Context, c *cli.Command, event *hook.UnifiedHookEvent) error {
	debug := os.Getenv("CCPERSONA_DEBUG") != ""

	codexEvent, ok := event.GetCodexEvent()
	if !ok {
		return fmt.Errorf("failed to get Codex event")
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] LastAssistantMessage: %q\n", codexEvent.LastAssistantMessage)
		fmt.Fprintf(os.Stderr, "[DEBUG] voice flag: %v\n", c.Bool("voice"))
	}

	log.Info().
		Str("thread_id", codexEvent.ThreadID).
		Str("turn_id", codexEvent.TurnID).
		Str("cwd", codexEvent.CWD).
		Msg("Codex agent turn complete")

	// Desktop notification (if enabled)
	if c.Bool("desktop") {
		message := fmt.Sprintf("Turn %s completed", codexEvent.TurnID)
		if err := showDesktopNotification(message, "normal"); err != nil {
			log.Warn().Err(err).Msg("Failed to show desktop notification")
		}
	}

	// Voice notification (if enabled)
	if c.Bool("voice") && codexEvent.LastAssistantMessage != "" {
		voiceConfig := voice.DefaultConfig()

		// Load persona config for voice settings
		config, _ := persona.LoadConfigWithFallback()
		if config != nil && config.Voice != nil {
			if config.Voice.Provider != "" {
				voiceConfig.EnginePriority = config.Voice.Provider
			}
			if config.Voice.Speaker > 0 {
				// Apply speaker ID to the appropriate engine based on priority
				if voiceConfig.EnginePriority == voice.EngineAivisSpeech {
					voiceConfig.AivisSpeechSpeaker = int64(config.Voice.Speaker)
				} else {
					voiceConfig.VoicevoxSpeaker = config.Voice.Speaker
				}
			}
		}

		// Process text according to reading mode
		reader := voice.NewTranscriptReader(voiceConfig)
		text := reader.ProcessText(codexEvent.LastAssistantMessage)
		text = voice.StripMarkdown(text)

		// Synthesize and play
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Text to synthesize: %q\n", text)
		}
		engine := voice.NewVoiceEngine(voiceConfig)
		audioFile, err := engine.Synthesize(text)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to synthesize voice")
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] Synthesize error: %v\n", err)
			}
		} else {
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] Audio file: %s\n", audioFile)
			}
			// Use PlayWithOptions with wait=true for hooks
			if err := engine.PlayWithOptions(audioFile, true); err != nil {
				log.Warn().Err(err).Msg("Failed to play audio")
				if debug {
					fmt.Fprintf(os.Stderr, "[DEBUG] Play error: %v\n", err)
				}
			}
		}
	}

	return nil
}

func handleVoiceSynthesisForEvent(ctx context.Context, c *cli.Command, event *hook.UnifiedHookEvent) error {
	if !c.Bool("voice") {
		return nil
	}

	// This would be similar to handleVoice command
	// but using the event's transcript path
	log.Debug().Msg("Voice synthesis for event not yet implemented")
	return nil
}

func handleNotificationEvent(ctx context.Context, c *cli.Command, event *hook.UnifiedHookEvent) error {
	message := event.AIResponse

	// Desktop notification
	if c.Bool("desktop") {
		urgency := "normal"
		if strings.Contains(strings.ToLower(message), "permission") {
			urgency = "critical"
		} else if strings.Contains(strings.ToLower(message), "idle") {
			urgency = "low"
		} else if strings.Contains(strings.ToLower(message), "error") {
			urgency = "high"
		}

		if err := showDesktopNotification(message, urgency); err != nil {
			log.Warn().Err(err).Msg("Failed to show desktop notification")
		}
	}

	// Voice notification
	if c.Bool("voice") {
		voiceConfig := voice.DefaultConfig()
		config, _ := persona.LoadConfigWithFallback()
		if config != nil && config.Voice != nil {
			if config.Voice.Provider != "" {
				voiceConfig.EnginePriority = config.Voice.Provider
			}
			if config.Voice.Speaker > 0 {
				// Apply speaker ID to the appropriate engine based on priority
				if voiceConfig.EnginePriority == voice.EngineAivisSpeech {
					voiceConfig.AivisSpeechSpeaker = int64(config.Voice.Speaker)
				} else {
					voiceConfig.VoicevoxSpeaker = config.Voice.Speaker
				}
			}
		}

		engine := voice.NewVoiceEngine(voiceConfig)
		audioFile, err := engine.Synthesize(message)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to synthesize voice")
		} else {
			// Use PlayWithOptions with wait=true for hooks
			if err := engine.PlayWithOptions(audioFile, true); err != nil {
				log.Warn().Err(err).Msg("Failed to play audio")
			}
		}
	}

	return nil
}

func showDesktopNotification(message, urgency string) error {
	title := "Claude Code"

	switch runtime.GOOS {
	case "darwin":
		// macOS notification using osascript
		script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)
		cmd := exec.Command("osascript", "-e", script)
		return cmd.Run()

	case "linux":
		// Linux notification using notify-send
		cmd := exec.Command("notify-send", "-u", urgency, title, message)
		return cmd.Run()

	case "windows":
		// Windows notification using PowerShell
		script := fmt.Sprintf(`
			[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
			[Windows.UI.Notifications.ToastNotification, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
			[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null

			$template = @"
			<toast>
				<visual>
					<binding template="ToastText02">
						<text id="1">%s</text>
						<text id="2">%s</text>
					</binding>
				</visual>
			</toast>
"@

			$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
			$xml.LoadXml($template)
			$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
			[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("Claude Code").Show($toast)
		`, title, message)
		cmd := exec.Command("powershell", "-Command", script)
		return cmd.Run()

	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

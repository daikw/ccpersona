package main

import (
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
	// Read notification event from stdin
	event, err := hook.ReadNotificationEvent()
	if err != nil {
		return fmt.Errorf("failed to read notification event: %w", err)
	}

	log.Info().
		Str("session_id", event.SessionID).
		Str("message", event.Message).
		Msg("Received notification")

	// Determine notification urgency based on message content
	urgency := "normal"

	// Analyze message content for urgency level only
	switch {
	case strings.Contains(strings.ToLower(event.Message), "permission"):
		urgency = "critical"

	case strings.Contains(strings.ToLower(event.Message), "idle"):
		urgency = "low"

	case strings.Contains(strings.ToLower(event.Message), "error"):
		urgency = "high"
	}

	// Desktop notification (if enabled)
	if c.Bool("desktop") {
		if err := showDesktopNotification(event.Message, urgency); err != nil {
			log.Warn().Err(err).Msg("Failed to show desktop notification")
		}
	}

	// Voice notification (if enabled)
	if c.Bool("voice") {
		// Load persona config to get voice settings
		config, _ := persona.LoadConfig(".")
		voiceConfig := voice.DefaultConfig()

		if config != nil && config.Voice != nil {
			if config.Voice.Engine != "" {
				voiceConfig.EnginePriority = config.Voice.Engine
			}
			if config.Voice.SpeakerID > 0 {
				// Apply speaker ID to the appropriate engine based on priority
				if voiceConfig.EnginePriority == voice.EngineAivisSpeech {
					voiceConfig.AivisSpeechSpeaker = int64(config.Voice.SpeakerID)
				} else {
					voiceConfig.VoicevoxSpeaker = config.Voice.SpeakerID
				}
			}
		}

		// Synthesize and play voice with original message
		engine := voice.NewVoiceEngine(voiceConfig)
		audioFile, err := engine.Synthesize(event.Message)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to synthesize voice")
		} else {
			if err := engine.Play(audioFile); err != nil {
				log.Warn().Err(err).Msg("Failed to play audio")
			}
		}
	}

	return nil
}

func handleCodexNotify(ctx context.Context, c *cli.Command) error {
	// Suppress normal output when running as hook
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	// Use unified hook interface to automatically detect Claude Code or Codex
	event, err := hook.DetectAndParse(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to parse hook event: %w", err)
	}

	log.Debug().
		Str("source", event.Source).
		Str("session_id", event.SessionID).
		Str("event_type", event.EventType).
		Msg("Received hook event")

	// Handle based on event source and type
	if event.IsCodex() {
		// Codex notify hook - triggered on agent-turn-complete
		return handleCodexAgentTurnComplete(ctx, c, event)
	} else if event.IsClaudeCode() {
		// Claude Code events - route to appropriate handler
		switch event.EventType {
		case "UserPromptSubmit":
			// Apply persona at session start
			if err := persona.HandleSessionStart(); err != nil {
				log.Error().Err(err).Msg("Failed to handle session start")
			}
			return nil
		case "Stop", "SubagentStop":
			// Voice synthesis for assistant response
			return handleVoiceSynthesisForEvent(ctx, c, event)
		case "Notification":
			// Desktop and voice notification
			return handleNotificationEvent(ctx, c, event)
		default:
			log.Debug().Str("event_type", event.EventType).Msg("Unhandled event type")
			return nil
		}
	}

	return nil
}

func handleCodexAgentTurnComplete(ctx context.Context, c *cli.Command, event *hook.UnifiedHookEvent) error {
	codexEvent, ok := event.GetCodexEvent()
	if !ok {
		return fmt.Errorf("failed to get Codex event")
	}

	log.Info().
		Str("thread_id", codexEvent.ThreadID).
		Int("turn_id", codexEvent.TurnID).
		Str("cwd", codexEvent.CWD).
		Msg("Codex agent turn complete")

	// Desktop notification (if enabled)
	if c.Bool("desktop") {
		message := fmt.Sprintf("Turn %d completed", codexEvent.TurnID)
		if err := showDesktopNotification(message, "normal"); err != nil {
			log.Warn().Err(err).Msg("Failed to show desktop notification")
		}
	}

	// Voice notification (if enabled)
	if c.Bool("voice") && codexEvent.LastAssistantMessage != "" {
		voiceConfig := voice.DefaultConfig()

		// Load persona config for voice settings
		config, _ := persona.LoadConfig(".")
		if config != nil && config.Voice != nil {
			if config.Voice.Engine != "" {
				voiceConfig.EnginePriority = config.Voice.Engine
			}
			if config.Voice.SpeakerID > 0 {
				// Apply speaker ID to the appropriate engine based on priority
				if voiceConfig.EnginePriority == voice.EngineAivisSpeech {
					voiceConfig.AivisSpeechSpeaker = int64(config.Voice.SpeakerID)
				} else {
					voiceConfig.VoicevoxSpeaker = config.Voice.SpeakerID
				}
			}
		}

		// Process text according to reading mode
		reader := voice.NewTranscriptReader(voiceConfig)
		text := reader.ProcessText(codexEvent.LastAssistantMessage)
		text = voice.StripMarkdown(text)

		// Synthesize and play
		engine := voice.NewVoiceEngine(voiceConfig)
		audioFile, err := engine.Synthesize(text)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to synthesize voice")
		} else {
			if err := engine.Play(audioFile); err != nil {
				log.Warn().Err(err).Msg("Failed to play audio")
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
		config, _ := persona.LoadConfig(".")
		if config != nil && config.Voice != nil {
			if config.Voice.Engine != "" {
				voiceConfig.EnginePriority = config.Voice.Engine
			}
			if config.Voice.SpeakerID > 0 {
				// Apply speaker ID to the appropriate engine based on priority
				if voiceConfig.EnginePriority == voice.EngineAivisSpeech {
					voiceConfig.AivisSpeechSpeaker = int64(config.Voice.SpeakerID)
				} else {
					voiceConfig.VoicevoxSpeaker = config.Voice.SpeakerID
				}
			}
		}

		engine := voice.NewVoiceEngine(voiceConfig)
		audioFile, err := engine.Synthesize(message)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to synthesize voice")
		} else {
			if err := engine.Play(audioFile); err != nil {
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

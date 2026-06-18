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
	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Routing: IsCodex=%v, IsCursor=%v, IsClaudeCode=%v\n",
			unifiedEvent.IsCodex(), unifiedEvent.IsCursor(), unifiedEvent.IsClaudeCode())
	}
	if unifiedEvent.IsCodex() {
		// Codex notify hook - triggered on agent-turn-complete
		return handleCodexAgentTurnComplete(ctx, c, unifiedEvent)
	} else if unifiedEvent.IsCursor() {
		// Cursor events - route to appropriate handler
		switch unifiedEvent.EventType {
		case "sessionStart":
			// Apply persona at session start (platform-aware)
			if err := persona.HandleSessionStartForPlatform(unifiedEvent.Source); err != nil {
				log.Error().Err(err).Msg("Failed to handle session start")
			}
			return nil
		case "beforeSubmitPrompt":
			// Could validate prompts here if needed
			return nil
		case "afterAgentResponse":
			// Voice synthesis using direct AI response text
			return handleDirectResponseVoice(ctx, c, unifiedEvent)
		case "stop":
			// Stop event doesn't contain AI response, skip voice synthesis
			log.Debug().Msg("Cursor stop event received (no voice synthesis)")
			return nil
		default:
			log.Debug().Str("event_type", unifiedEvent.EventType).Msg("Unhandled Cursor event type")
			return nil
		}
	} else if unifiedEvent.IsClaudeCode() {
		// Claude Code events - route to appropriate handler
		switch unifiedEvent.EventType {
		case "UserPromptSubmit":
			// Apply persona at session start (platform-aware)
			if err := persona.HandleSessionStartForPlatform(unifiedEvent.Source); err != nil {
				log.Error().Err(err).Msg("Failed to handle session start")
			}
			return nil
		case "Stop":
			// Voice synthesis for assistant response
			return handleStopEventVoice(ctx, c, unifiedEvent)
		case "SubagentStop":
			log.Debug().Msg("SubagentStop event ignored for voice synthesis")
			return nil
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

// toPersonaVoiceInput converts a persona.Config's Voice field to voice.PersonaVoiceInput.
// Returns a zero-value input if config or its Voice field is nil.
func toPersonaVoiceInput(config *persona.Config) voice.PersonaVoiceInput {
	if config == nil || config.Voice == nil {
		return voice.PersonaVoiceInput{}
	}
	return voice.PersonaVoiceInput{
		Provider: config.Voice.Provider,
		Speaker:  config.Voice.Speaker,
		Volume:   config.Voice.Volume,
		Speed:    config.Voice.Speed,
	}
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
		if voice.IsMuted() {
			log.Debug().Msg("voice synthesis is globally muted, skipping Codex turn voice")
			return nil
		}
		fileConfig := loadVoiceConfig(c)
		config, _ := persona.LoadConfigWithFallbackForPlatform(event.Source)
		opts := voice.Resolve(toPersonaVoiceInput(config), fileConfig, "")
		voiceConfig := opts.ToConfig(voice.DefaultConfig())

		// Process text according to reading mode
		reader := voice.NewTranscriptReader(voiceConfig)
		text := reader.ProcessText(codexEvent.LastAssistantMessage)
		text = voice.StripMarkdown(text)
		text = strings.TrimSpace(text)

		if text == "" {
			log.Debug().Msg("No text to synthesize after processing, skipping")
			return nil
		}

		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Text to synthesize: %q\n", text)
		}

		manager := voice.NewVoiceManager(voiceConfig)
		audioFile, err := manager.Synthesize(ctx, text, opts)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to synthesize voice")
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] Synthesize error: %v\n", err)
			}
		} else {
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] Audio file: %s\n", audioFile)
			}
			engine := voice.NewVoiceEngine(voiceConfig)
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

func handleStopEventVoice(ctx context.Context, c *cli.Command, event *hook.UnifiedHookEvent) error {
	debug := os.Getenv("CCPERSONA_DEBUG") != ""

	if !c.Bool("voice") {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Voice flag not set, skipping voice synthesis\n")
		}
		return nil
	}

	if voice.IsMuted() {
		log.Debug().Msg("voice synthesis is globally muted, skipping Stop event voice")
		return nil
	}

	// Get transcript path from the event
	var transcriptPath string
	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] RawEvent type: %T\n", event.RawEvent)
	}
	switch e := event.RawEvent.(type) {
	case *hook.StopEvent:
		transcriptPath = e.TranscriptPath
	case *hook.CursorStopEvent:
		transcriptPath = e.TranscriptPath
	default:
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Unknown event type for stop event: %T\n", event.RawEvent)
		}
		return nil
	}

	if transcriptPath == "" {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] No transcript path in event\n")
		}
		return nil
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Transcript path: %s\n", transcriptPath)
	}

	fileConfig := loadVoiceConfig(c)
	config, _ := persona.LoadConfigWithFallbackForPlatform(event.Source)
	opts := voice.Resolve(toPersonaVoiceInput(config), fileConfig, "")
	voiceConfig := opts.ToConfig(voice.DefaultConfig())

	// Read latest assistant message from transcript
	reader := voice.NewTranscriptReader(voiceConfig)
	text, err := reader.GetLatestAssistantMessage(transcriptPath)
	if err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Failed to get assistant message: %v\n", err)
		}
		log.Warn().Err(err).Msg("Failed to get assistant message from transcript")
		return nil
	}

	// Process text according to reading mode
	text = reader.ProcessText(text)
	text = voice.StripMarkdown(text)

	if text == "" {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] No text to synthesize after processing\n")
		}
		return nil
	}

	// Skip duplicate messages
	dedup := voice.NewDedupTracker(event.SessionID)
	if dedup.IsDuplicate(text) {
		log.Debug().Msg("Skipping duplicate voice synthesis")
		return nil
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Text to synthesize: %q\n", text)
	}

	manager := voice.NewVoiceManager(voiceConfig)
	audioFile, err := manager.Synthesize(ctx, text, opts)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to synthesize voice")
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Synthesize error: %v\n", err)
		}
		return nil
	}

	dedup.Record(text)
	go dedup.Cleanup()

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Audio file: %s\n", audioFile)
	}

	engine := voice.NewVoiceEngine(voiceConfig)
	if err := engine.PlayWithOptions(audioFile, true); err != nil {
		log.Warn().Err(err).Msg("Failed to play audio")
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Play error: %v\n", err)
		}
	}

	return nil
}

// handleDirectResponseVoice synthesizes voice from the AIResponse field directly
// Used for Cursor's afterAgentResponse event which provides the AI response in the event payload
func handleDirectResponseVoice(ctx context.Context, c *cli.Command, event *hook.UnifiedHookEvent) error {
	debug := os.Getenv("CCPERSONA_DEBUG") != ""

	if !c.Bool("voice") {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Voice flag not set, skipping voice synthesis\n")
		}
		return nil
	}

	if voice.IsMuted() {
		log.Debug().Msg("voice synthesis is globally muted, skipping direct-response voice")
		return nil
	}

	text := event.AIResponse
	if text == "" {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] No AI response in event\n")
		}
		return nil
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] AI response length: %d\n", len(text))
	}

	fileConfig := loadVoiceConfig(c)
	config, _ := persona.LoadConfigWithFallbackForPlatform(event.Source)
	opts := voice.Resolve(toPersonaVoiceInput(config), fileConfig, "")
	voiceConfig := opts.ToConfig(voice.DefaultConfig())

	// Process text according to reading mode
	reader := voice.NewTranscriptReader(voiceConfig)
	text = reader.ProcessText(text)
	text = voice.StripMarkdown(text)

	if text == "" {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] No text to synthesize after processing\n")
		}
		return nil
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Text to synthesize: %q\n", text)
	}

	manager := voice.NewVoiceManager(voiceConfig)
	audioFile, err := manager.Synthesize(ctx, text, opts)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to synthesize voice")
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Synthesize error: %v\n", err)
		}
		return nil
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Audio file: %s\n", audioFile)
	}

	engine := voice.NewVoiceEngine(voiceConfig)
	if err := engine.PlayWithOptions(audioFile, true); err != nil {
		log.Warn().Err(err).Msg("Failed to play audio")
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Play error: %v\n", err)
		}
	}

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
		if voice.IsMuted() {
			log.Debug().Msg("voice synthesis is globally muted, skipping notification voice")
			return nil
		}
		fileConfig := loadVoiceConfig(c)
		config, _ := persona.LoadConfigWithFallbackForPlatform(event.Source)
		opts := voice.Resolve(toPersonaVoiceInput(config), fileConfig, "")
		voiceConfig := opts.ToConfig(voice.DefaultConfig())

		manager := voice.NewVoiceManager(voiceConfig)
		audioFile, err := manager.Synthesize(ctx, message, opts)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to synthesize voice")
		} else {
			engine := voice.NewVoiceEngine(voiceConfig)
			if err := engine.PlayWithOptions(audioFile, true); err != nil {
				log.Warn().Err(err).Msg("Failed to play audio")
			}
		}
	}

	return nil
}

const notificationTitle = "Claude Code"

func showDesktopNotification(message, urgency string) error {
	cmd, err := buildNotificationCommand(runtime.GOOS, message, urgency)
	if err != nil {
		return err
	}
	return cmd.Run()
}

// buildNotificationCommand assembles the platform-specific desktop-notification
// command. Message and title are always passed out-of-band (argv / env var) so
// that they are never interpreted as script source by the shell or scripting host.
func buildNotificationCommand(goos, message, urgency string) (*exec.Cmd, error) {
	switch goos {
	case "darwin":
		args := osascriptNotifyArgs(message, notificationTitle)
		return exec.Command("osascript", args...), nil

	case "linux":
		args := notifySendArgs(message, urgency, notificationTitle)
		return exec.Command("notify-send", args...), nil

	case "windows":
		cmd := exec.Command("powershell", "-NoProfile", "-Command", windowsToastScript())
		cmd.Env = append(os.Environ(),
			"CCPERSONA_NOTIFY_TITLE="+notificationTitle,
			"CCPERSONA_NOTIFY_MESSAGE="+message,
		)
		return cmd, nil

	default:
		return nil, fmt.Errorf("unsupported platform: %s", goos)
	}
}

// osascriptNotifyArgs builds osascript args that take the message and title via
// `on run argv`, so the values are referenced as data rather than spliced into
// the AppleScript source. The `--` terminator keeps a message starting with
// `-e` from being parsed as an additional statement.
func osascriptNotifyArgs(message, title string) []string {
	script := `on run argv
	display notification (item 1 of argv) with title (item 2 of argv)
end run`
	return []string{"-e", script, "--", message, title}
}

// normalizeUrgency maps internal urgency labels to the values notify-send(1)
// accepts (low|normal|critical); an invalid value makes notify-send fail silently.
func normalizeUrgency(urgency string) string {
	switch urgency {
	case "low", "normal", "critical":
		return urgency
	case "high":
		return "critical"
	default:
		return "normal"
	}
}

// notifySendArgs builds notify-send args with a `--` separator so a message that
// starts with '-' is not mistaken for an option.
func notifySendArgs(message, urgency, title string) []string {
	return []string{"-u", normalizeUrgency(urgency), "--", title, message}
}

// windowsToastScript returns a PowerShell toast script that reads the title and
// message from environment variables, avoiding string interpolation into source.
func windowsToastScript() string {
	return `
		[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
		[Windows.UI.Notifications.ToastNotification, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
		[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null

		$title = $env:CCPERSONA_NOTIFY_TITLE
		$message = $env:CCPERSONA_NOTIFY_MESSAGE
		$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
		$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
		$texts = $template.GetElementsByTagName("text")
		$texts.Item(0).AppendChild($template.CreateTextNode($title)) | Out-Null
		$texts.Item(1).AppendChild($template.CreateTextNode($message)) | Out-Null
		$toast = [Windows.UI.Notifications.ToastNotification]::new($template)
		[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("Claude Code").Show($toast)
	`
}

package main

import (
	"context"
	"os"

	"github.com/daikw/ccpersona/internal/hook"
	"github.com/daikw/ccpersona/internal/persona"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

func handleHook(ctx context.Context, c *cli.Command) error {
	// Suppress normal output when running as hook
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	// Try to detect and parse hook event using unified interface
	unifiedEvent, err := hook.DetectAndParse(os.Stdin)
	if err != nil {
		// Fallback to legacy behavior if no stdin data or parse error
		log.Debug().Err(err).Msg("No hook event data from stdin, using legacy mode")
		// Still try to apply persona in legacy mode (no platform info available)
		if err := persona.HandleSessionStart(); err != nil {
			log.Error().Err(err).Msg("Failed to handle session start")
		}
		return nil
	}

	// Platform is available from the unified event
	platform := unifiedEvent.Source

	// Set session ID from hook event
	if unifiedEvent.SessionID != "" {
		_ = os.Setenv("CLAUDE_SESSION_ID", unifiedEvent.SessionID)
	}

	log.Debug().
		Str("source", unifiedEvent.Source).
		Str("event_type", unifiedEvent.EventType).
		Str("session_id", unifiedEvent.SessionID).
		Str("cwd", unifiedEvent.CWD).
		Msg("Received hook event")

	// Handle different event types (platform-aware)
	switch unifiedEvent.EventType {
	case "SessionStart":
		log.Debug().Str("platform", platform).Msg("Processing SessionStart hook")
		if err := persona.HandleSessionStartForPlatform(platform); err != nil {
			log.Error().Err(err).Msg("Failed to handle session start")
		}

	case "UserPromptSubmit":
		log.Debug().Str("platform", platform).Msg("Processing UserPromptSubmit hook (legacy)")
		if err := persona.HandleSessionStartForPlatform(platform); err != nil {
			log.Error().Err(err).Msg("Failed to handle session start")
		}

	case "SessionEnd":
		log.Debug().Msg("Processing SessionEnd hook")
		// Future: Add farewell voice synthesis or session summary

	default:
		log.Debug().Str("event_type", unifiedEvent.EventType).Msg("Unhandled hook event type")
	}

	return nil
}

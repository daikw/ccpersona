package main

import (
	"context"
	"os"
	"strings"

	"github.com/daikw/ccpersona/internal/hook"
	"github.com/daikw/ccpersona/internal/persona"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

func handleHook(ctx context.Context, c *cli.Command) error {
	// Suppress normal output when running as hook
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	platformHint := hookPlatformHint(c)

	// Try to detect and parse hook event using unified interface
	unifiedEvent, err := hook.DetectAndParseForSource(os.Stdin, platformHint)
	if err != nil {
		// Fallback to legacy behavior if no stdin data or parse error
		log.Debug().Err(err).Msg("No hook event data from stdin, using legacy mode")
		// Still try to apply persona in legacy mode. If the invoking platform is
		// known from flags or env, keep platform-specific persona lookup.
		if platformHint != "" {
			if err := persona.HandleSessionStartForPlatform(platformHint); err != nil {
				log.Error().Err(err).Msg("Failed to handle session start")
			}
		} else if err := persona.HandleSessionStart(); err != nil {
			log.Error().Err(err).Msg("Failed to handle session start")
		}
		return nil
	}

	// Platform is available from the unified event
	platform := unifiedEvent.Source

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

func hookPlatformHint(c *cli.Command) string {
	platform := strings.TrimSpace(c.String("platform"))
	if platform == "" {
		platform = strings.TrimSpace(os.Getenv("CCPERSONA_PLATFORM"))
	}

	switch strings.ToLower(platform) {
	case "", "auto":
		return ""
	case "codex":
		return "codex"
	case "claude", "claude-code":
		return "claude-code"
	case "cursor":
		return "cursor"
	default:
		log.Warn().Str("platform", platform).Msg("Ignoring unknown hook platform hint")
		return ""
	}
}

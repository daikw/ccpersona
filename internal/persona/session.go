package persona

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// HandleSessionStart is the main entry point for hook functionality
// This is a convenience wrapper for HandleSessionStartForPlatform with empty platform.
func HandleSessionStart() error {
	return HandleSessionStartForPlatform("")
}

// HandleSessionStartForPlatform is the main entry point for hook functionality with platform support
func HandleSessionStartForPlatform(platform string) error {
	// Load persona configuration (project or global fallback, platform-aware)
	config, err := LoadConfigWithFallbackForPlatform(platform)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if config == nil {
		log.Debug().Msg("No persona configuration found")
		return nil
	}

	log.Info().Str("persona", config.Name).Msg("Found persona configuration")

	// Read persona content
	manager, err := NewManager()
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}

	content, err := manager.ReadPersona(config.Name)
	if err != nil {
		return fmt.Errorf("failed to read persona: %w", err)
	}

	// Output persona content to stdout
	fmt.Print(content)

	// Append speak instruction if voice is configured
	if config.Voice != nil {
		fmt.Print("\n## speak ツールの利用\nユーザーへの確認・許可を求める際、作業完了の報告、または自発的に話しかけたい場面では、\nspeak MCP ツールを使って発話してください。\n")
	}

	return nil
}

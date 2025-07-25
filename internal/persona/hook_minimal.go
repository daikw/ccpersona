package persona

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

// MinimalHookScript is a tiny wrapper that calls ccpersona
const MinimalHookScript = `#!/bin/sh
exec ccpersona hook
`

// SetupMinimalHook creates a minimal hook script that executes ccpersona
func SetupMinimalHook() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	hooksDir := filepath.Join(homeDir, ".claude", "hooks")
	
	// Create hooks directory if it doesn't exist
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	targetHook := filepath.Join(hooksDir, "user-prompt-submit.sh")
	
	// Check if hook already exists
	if _, err := os.Stat(targetHook); err == nil {
		// Read existing content
		existing, err := os.ReadFile(targetHook)
		if err == nil && string(existing) == MinimalHookScript {
			log.Debug().Msg("Hook already installed")
			return nil
		}
		
		// Backup existing hook
		backupPath := targetHook + ".bak"
		if err := os.Rename(targetHook, backupPath); err != nil {
			return fmt.Errorf("failed to backup existing hook: %w", err)
		}
		log.Info().Str("backup", backupPath).Msg("Backed up existing hook")
	}

	// Write minimal hook script
	if err := os.WriteFile(targetHook, []byte(MinimalHookScript), 0755); err != nil {
		return fmt.Errorf("failed to write hook: %w", err)
	}

	log.Info().Str("path", targetHook).Msg("Installed minimal hook")
	return nil
}
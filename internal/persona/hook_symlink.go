package persona

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

// SetupHookSymlink creates a symlink to the ccpersona binary as the hook
func SetupHookSymlink() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	hooksDir := filepath.Join(homeDir, ".claude", "hooks")
	
	// Create hooks directory if it doesn't exist
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	// Hook name without .sh extension
	targetHook := filepath.Join(hooksDir, "user-prompt-submit")
	
	// Check if hook already exists
	if info, err := os.Lstat(targetHook); err == nil {
		// If it's already a symlink to ccpersona, we're done
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(targetHook)
			if err == nil && filepath.Base(link) == "ccpersona" {
				log.Debug().Msg("Hook already points to ccpersona")
				return nil
			}
		}
		
		// Backup existing hook if it's not our symlink
		backupPath := targetHook + ".bak"
		if err := os.Rename(targetHook, backupPath); err != nil {
			return fmt.Errorf("failed to backup existing hook: %w", err)
		}
		log.Info().Str("backup", backupPath).Msg("Backed up existing hook")
	}

	// Get ccpersona executable path from PATH
	// Since we expect ccpersona to be in PATH (via brew), just use the name
	ccpersonaPath := "ccpersona"
	
	// For development, check if we can find the absolute path
	if execPath, err := os.Executable(); err == nil {
		// If we're running from the built binary, use its path
		if filepath.Base(execPath) == "ccpersona" {
			ccpersonaPath = execPath
		}
	}

	// Create symlink to ccpersona
	if err := os.Symlink(ccpersonaPath, targetHook); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	log.Info().
		Str("from", targetHook).
		Str("to", ccpersonaPath).
		Msg("Created hook symlink")

	return nil
}

// RemoveHookSymlink removes the installed hook
func RemoveHookSymlink() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	targetHook := filepath.Join(homeDir, ".claude", "hooks", "user-prompt-submit")
	
	// Check if hook exists
	if info, err := os.Lstat(targetHook); err == nil {
		// Only remove if it's a symlink
		if info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(targetHook); err != nil {
				return fmt.Errorf("failed to remove hook: %w", err)
			}
			log.Info().Msg("Removed hook symlink")
			return nil
		}
		return fmt.Errorf("hook exists but is not a symlink")
	}

	return fmt.Errorf("hook not found")
}
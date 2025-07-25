package persona

import (
	"fmt"
	"os"
	"path/filepath"
)

// ApplyProjectPersona applies the persona configured for the current project
func ApplyProjectPersona(projectPath string) error {
	// Load project configuration
	config, err := LoadConfig(projectPath)
	if err != nil {
		return fmt.Errorf("failed to load project config: %w", err)
	}

	if config == nil {
		// No configuration found
		return nil
	}

	// Create manager
	manager, err := NewManager()
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}

	// Apply the persona
	if err := manager.ApplyPersona(config.Name); err != nil {
		return fmt.Errorf("failed to apply persona: %w", err)
	}

	return nil
}

// SetupHook creates a symlink or copies the hook script to the appropriate location
func SetupHook(sourceHook string) error {
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
		return fmt.Errorf("hook already exists at %s", targetHook)
	}

	// Copy the hook file
	input, err := os.ReadFile(sourceHook)
	if err != nil {
		return fmt.Errorf("failed to read source hook: %w", err)
	}

	if err := os.WriteFile(targetHook, input, 0755); err != nil {
		return fmt.Errorf("failed to write hook: %w", err)
	}

	return nil
}
package persona

import (
	"fmt"
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
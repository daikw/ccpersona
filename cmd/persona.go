package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/daikw/ccpersona/internal/persona"
	"github.com/urfave/cli/v3"
)

func handlePersonaShow(ctx context.Context, c *cli.Command) error {
	personaName := c.Args().Get(0)
	if personaName == "" {
		return fmt.Errorf("persona name is required (usage: ccpersona persona show <name>)")
	}

	manager, err := persona.NewManager()
	if err != nil {
		return err
	}

	content, err := manager.ReadPersona(personaName)
	if err != nil {
		return err
	}

	fmt.Println(content)
	return nil
}

func handleEdit(ctx context.Context, c *cli.Command) error {
	personaName := c.Args().Get(0)
	if personaName == "" {
		return fmt.Errorf("persona name is required")
	}

	manager, err := persona.NewManager()
	if err != nil {
		return err
	}

	// Create persona if it doesn't exist
	if !manager.PersonaExists(personaName) {
		if err := manager.CreatePersona(personaName); err != nil {
			return err
		}
		fmt.Printf("Created new persona: %s\n", personaName)
	}

	// Get the path to the persona file
	path := manager.GetPersonaPath(personaName)

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	fmt.Printf("Edited persona: %s\n", personaName)
	return nil
}

func handleConfig(ctx context.Context, c *cli.Command) error {
	var configPath string
	var config *persona.Config
	var err error

	if c.Bool("global") {
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return fmt.Errorf("failed to get home directory: %w", homeErr)
		}
		configDir := filepath.Join(homeDir, persona.AgentsDir)
		configPath = filepath.Join(configDir, persona.ConfigFileName)

		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		config, err = persona.LoadConfigFromPath(configPath)
		if err != nil {
			// A parse failure must not be swallowed into a default that would
			// overwrite the existing (broken) config on save.
			return fmt.Errorf("failed to load global config: %w", err)
		}
		if config == nil {
			config = persona.GetDefaultConfig()
			if err := persona.SaveConfig(homeDir, config); err != nil {
				return fmt.Errorf("failed to create global config: %w", err)
			}
		}
	} else {
		configPath = persona.ConfigPath(".")

		config, err = persona.LoadConfigFromPath(configPath)
		if err != nil {
			return err
		}
		if config == nil {
			return fmt.Errorf("no project configuration found. Run 'ccpersona config init' first")
		}
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, configPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	if c.Bool("global") {
		fmt.Println("Edited global configuration")
	} else {
		fmt.Println("Edited project configuration")
	}
	return nil
}

func handleConfigMigrate(ctx context.Context, c *cli.Command) error {
	baseDir := "."
	scope := "project"
	if c.Bool("global") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		baseDir = homeDir
		scope = "global"
	}

	path, err := persona.MigrateConfig(baseDir, c.Bool("force"))
	if err != nil {
		return err
	}
	fmt.Printf("Migrated legacy configuration to %s (%s)\n", path, scope)
	return nil
}

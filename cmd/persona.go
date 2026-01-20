package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/daikw/ccpersona/internal/persona"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

func handleInit(ctx context.Context, c *cli.Command) error {
	log.Info().Msg("Initializing persona configuration...")

	config, err := persona.LoadConfig(".")
	if err != nil {
		return err
	}

	if config != nil {
		log.Warn().Msg("Persona configuration already exists")
		return nil
	}

	defaultConfig := persona.GetDefaultConfig()

	if err := persona.SaveConfig(".", defaultConfig); err != nil {
		return err
	}

	log.Info().Msg("Persona configuration initialized successfully")
	fmt.Println("Created .claude/persona.json with default configuration")
	return nil
}

func handleList(ctx context.Context, c *cli.Command) error {
	manager, err := persona.NewManager()
	if err != nil {
		return err
	}

	personas, err := manager.ListPersonas()
	if err != nil {
		return err
	}

	if len(personas) == 0 {
		fmt.Println("No personas found. Create one with 'ccpersona create <name>'")
		return nil
	}

	fmt.Println("Available personas:")
	for _, p := range personas {
		fmt.Printf("  - %s\n", p)
	}

	return nil
}

func handleCurrent(ctx context.Context, c *cli.Command) error {
	manager, err := persona.NewManager()
	if err != nil {
		return err
	}

	current, err := manager.GetCurrentPersona()
	if err != nil {
		return err
	}

	fmt.Printf("Current persona: %s\n", current)
	return nil
}

func handleSet(ctx context.Context, c *cli.Command) error {
	personaName := c.Args().Get(0)
	if personaName == "" {
		return fmt.Errorf("persona name is required")
	}

	manager, err := persona.NewManager()
	if err != nil {
		return err
	}

	if !manager.PersonaExists(personaName) {
		return fmt.Errorf("persona '%s' does not exist", personaName)
	}

	config, err := persona.LoadConfig(".")
	if err != nil {
		return err
	}

	if config == nil {
		config = persona.GetDefaultConfig()
	}

	config.Name = personaName

	if err := persona.SaveConfig(".", config); err != nil {
		return err
	}

	if err := manager.ApplyPersona(personaName); err != nil {
		return err
	}

	fmt.Printf("Set active persona to: %s\n", personaName)
	return nil
}

func handleShow(ctx context.Context, c *cli.Command) error {
	personaName := c.Args().Get(0)
	if personaName == "" {
		return fmt.Errorf("persona name is required")
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

func handleCreate(ctx context.Context, c *cli.Command) error {
	name := c.Args().Get(0)
	if name == "" {
		return fmt.Errorf("persona name is required")
	}

	manager, err := persona.NewManager()
	if err != nil {
		return err
	}

	if err := manager.CreatePersona(name); err != nil {
		return err
	}

	fmt.Printf("Created persona: %s\n", name)
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

	if !manager.PersonaExists(personaName) {
		return fmt.Errorf("persona '%s' does not exist", personaName)
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
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir := filepath.Join(homeDir, ".claude")
		configPath = filepath.Join(configDir, "persona.json")

		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		config, err = persona.LoadConfig(homeDir)
		if err != nil || config == nil {
			config = persona.GetDefaultConfig()
			if err := persona.SaveConfig(homeDir, config); err != nil {
				return fmt.Errorf("failed to create global config: %w", err)
			}
		}
	} else {
		configPath = filepath.Join(".claude", "persona.json")

		config, err = persona.LoadConfig(".")
		if err != nil {
			return err
		}
		if config == nil {
			return fmt.Errorf("no project configuration found. Run 'ccpersona init' first")
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

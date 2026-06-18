package main

import (
	"context"
	"fmt"
	"os"

	"github.com/daikw/ccpersona/internal/cliui"
	"github.com/daikw/ccpersona/internal/persona"
	"github.com/urfave/cli/v3"
)

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
		fmt.Println(cliui.Warn("No personas found."))
		fmt.Println("Create one with 'ccpersona edit <name>'.")
		return nil
	}

	// Resolve the active persona using the normal project -> global order.
	active, _ := manager.GetCurrentPersona()

	for _, p := range personas {
		if p == active {
			fmt.Printf("%s %s\n", cliui.Success("*"), p)
		} else {
			fmt.Printf("  %s\n", p)
		}
	}
	return nil
}

func handleSet(ctx context.Context, c *cli.Command) error {
	name := c.Args().Get(0)
	if name == "" {
		return fmt.Errorf("persona name is required (usage: ccpersona set <name>)")
	}

	manager, err := persona.NewManager()
	if err != nil {
		return err
	}

	if !manager.PersonaExists(name) {
		// main prints returned errors; embed the guidance instead of
		// pre-printing here, which produced a duplicate report.
		return fmt.Errorf("persona '%s' does not exist\nRun 'ccpersona list' to see available personas", name)
	}

	global := c.Bool("global")

	// Determine target directory and load existing config to preserve other fields.
	var targetDir string
	scope := "project"
	if global {
		scope = "global"
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return fmt.Errorf("failed to get home directory: %w", homeErr)
		}
		targetDir = homeDir
	} else {
		targetDir = "."
	}

	config, err := persona.LoadConfig(targetDir)
	if err != nil {
		return fmt.Errorf("failed to load existing config: %w", err)
	}
	if config == nil {
		config = persona.GetDefaultConfig()
	}
	config.Name = name

	if err := persona.SaveConfig(targetDir, config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Persona set to '%s' (%s)\n", name, scope)
	return nil
}

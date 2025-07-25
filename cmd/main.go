package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/daikw/ccpersona/internal/persona"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

var (
	version  = "dev"
	revision = "none"
)

func main() {
	// Setup logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	app := &cli.Command{
		Name:  "ccpersona",
		Usage: "Claude Code Persona System - manage personas for Claude Code sessions",
		Description: `ccpersona helps you manage different personas for Claude Code.
It allows you to switch between different personality settings, voice configurations,
and behavioral patterns for your AI assistant.`,
		Version: fmt.Sprintf("%s (rev: %s)", version, revision),
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"V"},
				Usage:   "Enable verbose logging",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "init",
				Usage:   "Initialize persona configuration in current project",
				Action:  handleInit,
				Aliases: []string{"i"},
			},
			{
				Name:    "list",
				Usage:   "List available personas",
				Action:  handleList,
				Aliases: []string{"ls", "l"},
			},
			{
				Name:    "current",
				Usage:   "Show current active persona",
				Action:  handleCurrent,
				Aliases: []string{"c"},
			},
			{
				Name:      "set",
				Usage:     "Set the active persona for this project",
				Action:    handleSet,
				Aliases:   []string{"s"},
				ArgsUsage: "<persona>",
			},
			{
				Name:      "show",
				Usage:     "Show details of a specific persona",
				Action:    handleShow,
				ArgsUsage: "<persona>",
			},
			{
				Name:      "create",
				Usage:     "Create a new persona",
				Action:    handleCreate,
				ArgsUsage: "<name>",
			},
			{
				Name:      "edit",
				Usage:     "Edit an existing persona",
				Action:    handleEdit,
				ArgsUsage: "<persona>",
			},
			{
				Name:   "config",
				Usage:  "Manage global configuration",
				Action: handleConfig,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "global",
						Aliases: []string{"g"},
						Usage:   "Edit global settings",
					},
				},
			},
			{
				Name:   "apply",
				Usage:  "Apply the configured persona for current project (used by hooks)",
				Action: handleApply,
				Hidden: true, // Hidden because it's mainly for hook usage
			},
			{
				Name:   "install-hook",
				Usage:  "Install the UserPromptSubmit hook",
				Action: handleInstallHook,
			},
		},
		Before: func(ctx context.Context, c *cli.Command) error {
			if c.Bool("verbose") {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			} else {
				zerolog.SetGlobalLevel(zerolog.InfoLevel)
			}
			return nil
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal().Err(err).Msg("Failed to run application")
	}
}

func handleInit(ctx context.Context, c *cli.Command) error {
	log.Info().Msg("Initializing persona configuration...")
	
	// Check if persona.json already exists
	config, err := persona.LoadConfig(".")
	if err != nil {
		return err
	}
	
	if config != nil {
		log.Warn().Msg("Persona configuration already exists")
		return nil
	}

	// Create default configuration
	defaultConfig := persona.GetDefaultConfig()
	
	// Save configuration
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

	// Check if persona exists
	if !manager.PersonaExists(personaName) {
		return fmt.Errorf("persona '%s' does not exist", personaName)
	}

	// Update project configuration
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

	// Apply persona
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

	fmt.Printf("Created new persona: %s\n", name)
	fmt.Printf("Edit it with: ccpersona edit %s\n", name)
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
	
	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Default to vi
	}

	// Open editor
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
	if c.Bool("global") {
		log.Info().Msg("Opening global configuration")
	} else {
		log.Info().Msg("Opening project configuration")
	}
	// TODO: Implement configuration management
	return nil
}

func handleApply(ctx context.Context, c *cli.Command) error {
	// Apply persona based on current project configuration
	if err := persona.ApplyProjectPersona("."); err != nil {
		return err
	}
	return nil
}

func handleInstallHook(ctx context.Context, c *cli.Command) error {
	// Get the path to the hook script
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	
	execDir := filepath.Dir(execPath)
	hookPath := filepath.Join(execDir, "..", "hooks", "persona_router.sh")
	
	// Check if hook exists in the expected location
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		// Try relative to current directory (development mode)
		hookPath = filepath.Join(".", "hooks", "persona_router.sh")
		if _, err := os.Stat(hookPath); os.IsNotExist(err) {
			return fmt.Errorf("hook script not found")
		}
	}

	if err := persona.SetupHook(hookPath); err != nil {
		return err
	}

	fmt.Println("UserPromptSubmit hook installed successfully")
	fmt.Println("The hook will automatically apply personas when you start Claude Code sessions")
	return nil
}
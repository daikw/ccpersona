package main

import (
	"testing"

	"github.com/urfave/cli/v3"
)

func findCommand(commands []*cli.Command, name string) *cli.Command {
	for _, cmd := range commands {
		if cmd.Name == name {
			return cmd
		}
	}
	return nil
}

func requireCommand(t *testing.T, commands []*cli.Command, name string) *cli.Command {
	t.Helper()
	cmd := findCommand(commands, name)
	if cmd == nil {
		t.Fatalf("command %q not found", name)
	}
	return cmd
}

func TestCommandHierarchy_ConfigSubcommands(t *testing.T) {
	app := newApp()
	config := requireCommand(t, app.Commands, "config")

	for _, name := range []string{
		"init",
		"edit",
		"show",
		"status",
		"set-persona",
		"migrate",
	} {
		requireCommand(t, config.Commands, name)
	}
}

func TestCommandHierarchy_PersonaSubcommands(t *testing.T) {
	app := newApp()
	persona := requireCommand(t, app.Commands, "persona")

	for _, name := range []string{"list", "show", "edit"} {
		requireCommand(t, persona.Commands, name)
	}
}

func TestCommandHierarchy_RuntimeSubcommands(t *testing.T) {
	app := newApp()
	runtime := requireCommand(t, app.Commands, "runtime")

	for _, name := range []string{"hook", "voice", "notify", "mcp", "engine"} {
		cmd := requireCommand(t, runtime.Commands, name)
		if cmd.Hidden {
			t.Fatalf("runtime %s should be visible", name)
		}
	}
}

func TestCommandHierarchy_RemovesTopLevelBasicCommands(t *testing.T) {
	app := newApp()

	for _, name := range []string{
		"init",
		"show",
		"list",
		"set",
		"edit",
		"status",
	} {
		if cmd := findCommand(app.Commands, name); cmd != nil {
			t.Fatalf("top-level %q should not exist", name)
		}
	}
}

func TestCommandHierarchy_HiddenRuntimeCompatibility(t *testing.T) {
	app := newApp()

	for _, name := range []string{"hook", "voice", "notify", "mcp", "engine"} {
		cmd := requireCommand(t, app.Commands, name)
		if !cmd.Hidden {
			t.Fatalf("top-level %s should be hidden", name)
		}
	}
}

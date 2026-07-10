package main

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/daikw/ccpersona/internal/voice"
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

func TestCommandHierarchy_VoiceMuteSubcommands(t *testing.T) {
	app := newApp()
	runtime := requireCommand(t, app.Commands, "runtime")
	voiceCommand := requireCommand(t, runtime.Commands, "voice")

	for _, name := range []string{"mute", "unmute", "toggle", "status"} {
		requireCommand(t, voiceCommand.Commands, name)
	}
}

func TestVoiceToggleCommand(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := newApp().Run(context.Background(), []string{
		"ccpersona", "runtime", "voice", "toggle",
	}); err != nil {
		t.Fatalf("toggle to muted: %v", err)
	}
	status, err := voice.LoadMuteStatus()
	if err != nil {
		t.Fatalf("load mute status: %v", err)
	}
	if status == nil {
		t.Fatal("toggle should mute voice when initially unmuted")
	}

	if err := newApp().Run(context.Background(), []string{
		"ccpersona", "runtime", "voice", "toggle",
	}); err != nil {
		t.Fatalf("toggle to unmuted: %v", err)
	}
	if voice.IsMuted() {
		t.Fatal("second toggle should unmute voice")
	}
}

func TestVoiceMuteUnmuteCommandOutput(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "mute",
			args: []string{"runtime", "voice", "mute"},
			want: "🔇 Voice synthesis muted.\n",
		},
		{
			name: "already muted",
			args: []string{"runtime", "voice", "mute"},
			want: "🔇 Voice synthesis is already muted; nothing to do.\n",
		},
		{
			name: "unmute",
			args: []string{"runtime", "voice", "unmute"},
			want: "🔊 Voice synthesis unmuted.\n",
		},
		{
			name: "already unmuted",
			args: []string{"runtime", "voice", "unmute"},
			want: "🔊 Voice synthesis is already unmuted; nothing to do.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := captureCommandStdout(t, tt.args...)
			if got != tt.want {
				t.Fatalf("stdout = %q, want %q", got, tt.want)
			}
		})
	}
}

func captureCommandStdout(t *testing.T, args ...string) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = w

	runErr := newApp().Run(context.Background(), append([]string{"ccpersona"}, args...))
	if err := w.Close(); err != nil {
		os.Stdout = original
		t.Fatalf("close stdout pipe: %v", err)
	}
	os.Stdout = original
	output, readErr := io.ReadAll(r)
	if closeErr := r.Close(); closeErr != nil {
		t.Fatalf("close stdout reader: %v", closeErr)
	}
	if readErr != nil {
		t.Fatalf("read stdout: %v", readErr)
	}
	if runErr != nil {
		t.Fatalf("run command: %v", runErr)
	}
	return string(output)
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

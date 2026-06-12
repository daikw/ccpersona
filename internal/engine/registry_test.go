package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHealthURL(t *testing.T) {
	tests := []struct {
		name       string
		baseURL    string
		healthType string
		want       string
	}{
		{"voicevox", "http://127.0.0.1:50021", HealthVoicevox, "http://127.0.0.1:50021/version"},
		{"openai", "http://127.0.0.1:8088", HealthOpenAI, "http://127.0.0.1:8088/v1/models"},
		{"openai trailing slash", "http://127.0.0.1:8088/", HealthOpenAI, "http://127.0.0.1:8088/v1/models"},
		{"empty health defaults openai", "http://127.0.0.1:9000", "", "http://127.0.0.1:9000/v1/models"},
		{"unknown health defaults openai", "http://127.0.0.1:9000", "weird", "http://127.0.0.1:9000/v1/models"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HealthURL(tt.baseURL, tt.healthType); got != tt.want {
				t.Errorf("HealthURL(%q, %q) = %q, want %q", tt.baseURL, tt.healthType, got, tt.want)
			}
		})
	}
}

func TestBuildRegistry_BuiltinsOnly(t *testing.T) {
	reg, err := BuildRegistry(nil)
	if err != nil {
		t.Fatalf("BuildRegistry(nil) error = %v", err)
	}
	names := reg.Names()
	if len(names) != 2 || names[0] != "voicevox" || names[1] != "aivisspeech" {
		t.Errorf("Names() = %v, want [voicevox aivisspeech]", names)
	}
	for _, n := range names {
		def, ok := reg.Get(n)
		if !ok {
			t.Fatalf("Get(%q) not found", n)
		}
		if !def.Builtin() {
			t.Errorf("%s: Builtin() = false, want true", n)
		}
		if def.HealthType != HealthVoicevox {
			t.Errorf("%s: HealthType = %q, want voicevox", n, def.HealthType)
		}
	}
	if def, _ := reg.Get("voicevox"); def.BaseURL != "http://127.0.0.1:50021" {
		t.Errorf("voicevox BaseURL = %q", def.BaseURL)
	}
	if def, _ := reg.Get("aivisspeech"); def.BaseURL != "http://127.0.0.1:10101" {
		t.Errorf("aivisspeech BaseURL = %q", def.BaseURL)
	}
}

func TestBuildRegistry_UserEngine(t *testing.T) {
	user := map[string]UserEngineConfig{
		"irodori": {
			BaseURL: "http://127.0.0.1:8088",
			Health:  "openai",
			Command: "uv",
			Args:    []string{"run", "irodori-tts-server"},
			Dir:     "~/src/Irodori-TTS-Server",
		},
	}
	reg, err := BuildRegistry(user)
	if err != nil {
		t.Fatalf("BuildRegistry error = %v", err)
	}
	def, ok := reg.Get("irodori")
	if !ok {
		t.Fatal("irodori not found")
	}
	if !def.Managed() {
		t.Error("irodori Managed() = false, want true")
	}
	if def.Builtin() {
		t.Error("irodori Builtin() = true, want false")
	}
	if def.HealthURL() != "http://127.0.0.1:8088/v1/models" {
		t.Errorf("irodori HealthURL = %q", def.HealthURL())
	}
	if def.ServiceLabel() != "com.ccpersona.engine.irodori" {
		t.Errorf("irodori ServiceLabel = %q", def.ServiceLabel())
	}
	if def.SystemdUnitName() != "ccpersona-engine-irodori.service" {
		t.Errorf("irodori SystemdUnitName = %q", def.SystemdUnitName())
	}
	if strings.HasPrefix(def.Dir, "~") {
		t.Errorf("irodori Dir not expanded: %q", def.Dir)
	}
}

func TestBuildRegistry_HealthDefaultsOpenAI(t *testing.T) {
	reg, err := BuildRegistry(map[string]UserEngineConfig{
		"kani": {BaseURL: "http://127.0.0.1:8000", Command: "kani-tts"},
	})
	if err != nil {
		t.Fatalf("BuildRegistry error = %v", err)
	}
	def, _ := reg.Get("kani")
	if def.HealthType != HealthOpenAI {
		t.Errorf("kani HealthType = %q, want openai", def.HealthType)
	}
}

func TestBuildRegistry_ExternalEngine(t *testing.T) {
	reg, err := BuildRegistry(map[string]UserEngineConfig{
		"remote": {BaseURL: "http://10.0.0.5:8088", Health: "openai"},
	})
	if err != nil {
		t.Fatalf("BuildRegistry error = %v", err)
	}
	def, _ := reg.Get("remote")
	if def.Managed() {
		t.Error("remote Managed() = true, want false (no command)")
	}
}

func TestBuildRegistry_NameCollision(t *testing.T) {
	_, err := BuildRegistry(map[string]UserEngineConfig{
		"voicevox": {BaseURL: "http://127.0.0.1:50021", Command: "x"},
	})
	if err == nil {
		t.Fatal("expected error on name collision with built-in")
	}
	if !strings.Contains(err.Error(), "conflicts") {
		t.Errorf("error = %v, want collision message", err)
	}
}

func TestBuildRegistry_InvalidHealth(t *testing.T) {
	_, err := BuildRegistry(map[string]UserEngineConfig{
		"bad": {BaseURL: "http://x", Health: "bogus", Command: "y"},
	})
	if err == nil {
		t.Fatal("expected error on invalid health type")
	}
}

func TestRegistry_Resolve(t *testing.T) {
	reg, _ := BuildRegistry(map[string]UserEngineConfig{
		"irodori": {BaseURL: "http://127.0.0.1:8088", Command: "uv"},
	})

	// "all" expands to built-ins only
	all, err := reg.Resolve("all")
	if err != nil {
		t.Fatalf("Resolve(all) error = %v", err)
	}
	if len(all) != 2 {
		t.Errorf("Resolve(all) returned %d engines, want 2 (built-ins only)", len(all))
	}

	// empty == all
	empty, _ := reg.Resolve("")
	if len(empty) != 2 {
		t.Errorf("Resolve(\"\") returned %d engines, want 2", len(empty))
	}

	// user-defined name resolves to single engine
	one, err := reg.Resolve("irodori")
	if err != nil {
		t.Fatalf("Resolve(irodori) error = %v", err)
	}
	if len(one) != 1 || one[0].Name != "irodori" {
		t.Errorf("Resolve(irodori) = %v", one)
	}

	// unknown name errors with available list
	_, err = reg.Resolve("nope")
	if err == nil {
		t.Fatal("expected error for unknown engine")
	}
	if !strings.Contains(err.Error(), "available:") {
		t.Errorf("error = %v, want available list", err)
	}
}

func TestBuildRegistry_InvalidName(t *testing.T) {
	bad := []string{
		"../../etc/cron.d/x",
		"foo/bar",
		"foo bar",
		"-leadingdash",
		"_leadingunderscore",
		"",
		"name.with.dots",
		strings.Repeat("a", 65),
	}
	for _, name := range bad {
		t.Run(name, func(t *testing.T) {
			_, err := BuildRegistry(map[string]UserEngineConfig{
				name: {BaseURL: "http://x", Command: "y"},
			})
			if err == nil {
				t.Fatalf("BuildRegistry(%q) expected error, got nil", name)
			}
		})
	}
}

func TestBuildRegistry_ValidNames(t *testing.T) {
	good := []string{"irodori", "kani-tts", "tts_2", "A", strings.Repeat("a", 64)}
	for _, name := range good {
		t.Run(name, func(t *testing.T) {
			if _, err := BuildRegistry(map[string]UserEngineConfig{
				name: {BaseURL: "http://x", Command: "y"},
			}); err != nil {
				t.Fatalf("BuildRegistry(%q) unexpected error: %v", name, err)
			}
		})
	}
}

func TestBuildRegistry_ControlCharsRejected(t *testing.T) {
	tests := []struct {
		name string
		uc   UserEngineConfig
	}{
		{"command newline", UserEngineConfig{BaseURL: "http://x", Command: "uv\nExecStart=/bin/sh"}},
		{"command CR", UserEngineConfig{BaseURL: "http://x", Command: "uv\rfoo"}},
		{"command NUL", UserEngineConfig{BaseURL: "http://x", Command: "uv\x00foo"}},
		{"dir newline", UserEngineConfig{BaseURL: "http://x", Command: "uv", Dir: "/tmp\nWorkingDirectory=/etc"}},
		{"arg newline", UserEngineConfig{BaseURL: "http://x", Command: "uv", Args: []string{"ok", "bad\nExecStart=evil"}}},
		{"env value newline", UserEngineConfig{BaseURL: "http://x", Command: "uv", Env: map[string]string{"X": "ok\nExecStart=/bin/sh -c evil"}}},
		{"env key newline", UserEngineConfig{BaseURL: "http://x", Command: "uv", Env: map[string]string{"X\nExecStart=y": "z"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildRegistry(map[string]UserEngineConfig{"svc": tt.uc})
			if err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
			if !strings.Contains(err.Error(), "control character") {
				t.Errorf("error = %v, want control-character message", err)
			}
		})
	}
}

func TestExpandHome_Traversal(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}

	// Escaping paths must be rejected.
	for _, p := range []string{"~/../../etc", "~/.."} {
		if _, err := expandHome(p); err == nil {
			t.Errorf("expandHome(%q) expected error, got nil", p)
		}
	}

	// In-home paths expand fine.
	got, err := expandHome("~/src/x")
	if err != nil {
		t.Fatalf("expandHome(~/src/x) error = %v", err)
	}
	if got != filepath.Join(home, "src", "x") {
		t.Errorf("expandHome(~/src/x) = %q", got)
	}

	// Non-tilde paths pass through unchanged (operator may point at /opt etc).
	if got, _ := expandHome("/opt/engine"); got != "/opt/engine" {
		t.Errorf("expandHome(/opt/engine) = %q, want /opt/engine", got)
	}
	if got, _ := expandHome(""); got != "" {
		t.Errorf("expandHome(\"\") = %q, want empty", got)
	}
}

func TestBuildRegistry_DirTraversalRejected(t *testing.T) {
	_, err := BuildRegistry(map[string]UserEngineConfig{
		"svc": {BaseURL: "http://x", Command: "uv", Dir: "~/../../etc"},
	})
	if err == nil {
		t.Fatal("expected error for traversing dir")
	}
}

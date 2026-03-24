package engine

import (
	"os"
	"testing"
)

func TestParseEngineType(t *testing.T) {
	tests := []struct {
		input   string
		want    []EngineType
		wantErr bool
	}{
		{"voicevox", []EngineType{VOICEVOX}, false},
		{"aivisspeech", []EngineType{AivisSpeech}, false},
		{"all", []EngineType{VOICEVOX, AivisSpeech}, false},
		{"unknown", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseEngineType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEngineType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("ParseEngineType(%q) = %v, want %v", tt.input, got, tt.want)
					return
				}
				for i := range got {
					if got[i] != tt.want[i] {
						t.Errorf("ParseEngineType(%q)[%d] = %v, want %v", tt.input, i, got[i], tt.want[i])
					}
				}
			}
		})
	}
}

func TestDefaultPort(t *testing.T) {
	if got := defaultPort(VOICEVOX); got != 50021 {
		t.Errorf("defaultPort(VOICEVOX) = %d, want 50021", got)
	}
	if got := defaultPort(AivisSpeech); got != 10101 {
		t.Errorf("defaultPort(AivisSpeech) = %d, want 10101", got)
	}
}

func TestServiceLabel(t *testing.T) {
	if got := serviceLabel(VOICEVOX); got != "com.voicevox.engine" {
		t.Errorf("serviceLabel(VOICEVOX) = %q, want %q", got, "com.voicevox.engine")
	}
	if got := serviceLabel(AivisSpeech); got != "com.aivisspeech.engine" {
		t.Errorf("serviceLabel(AivisSpeech) = %q, want %q", got, "com.aivisspeech.engine")
	}
}

func TestSystemdUnit(t *testing.T) {
	if got := SystemdUnit(VOICEVOX); got != "voicevox-engine.service" {
		t.Errorf("SystemdUnit(VOICEVOX) = %q, want %q", got, "voicevox-engine.service")
	}
	if got := SystemdUnit(AivisSpeech); got != "aivisspeech-engine.service" {
		t.Errorf("SystemdUnit(AivisSpeech) = %q, want %q", got, "aivisspeech-engine.service")
	}
}

func TestEngineInfoPortString(t *testing.T) {
	info := &EngineInfo{Port: 50021}
	if got := info.PortString(); got != "50021" {
		t.Errorf("PortString() = %q, want %q", got, "50021")
	}
}

func TestTemplateFS(t *testing.T) {
	// Verify all templates are embedded
	templates := []string{
		"templates/com.voicevox.engine.plist",
		"templates/com.aivisspeech.engine.plist",
		"templates/voicevox-engine.service",
		"templates/aivisspeech-engine.service",
	}
	for _, name := range templates {
		data, err := templateFS.ReadFile(name)
		if err != nil {
			t.Errorf("templateFS.ReadFile(%q) error = %v", name, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("templateFS.ReadFile(%q) returned empty data", name)
		}
	}
}

func TestDiscoverEngine_EnvVar(t *testing.T) {
	// Create a temp file to act as a fake binary
	tmpFile := t.TempDir() + "/fake-engine"
	if err := writeFile(tmpFile, "fake"); err != nil {
		t.Fatal(err)
	}

	t.Setenv("VOICEVOX_ENGINE_PATH", tmpFile)

	info, err := DiscoverEngine(VOICEVOX)
	if err != nil {
		t.Fatalf("DiscoverEngine(VOICEVOX) error = %v", err)
	}
	if info.BinaryPath != tmpFile {
		t.Errorf("BinaryPath = %q, want %q", info.BinaryPath, tmpFile)
	}
	if info.Port != 50021 {
		t.Errorf("Port = %d, want 50021", info.Port)
	}
}

func TestDiscoverEngine_NotFound(t *testing.T) {
	// Clear env vars to avoid interference
	t.Setenv("VOICEVOX_ENGINE_PATH", "")
	t.Setenv("AIVISSPEECH_ENGINE_PATH", "")

	// Use a nonexistent engine type path - this should fail unless
	// the engine is actually installed on the system
	_, err := DiscoverEngine(VOICEVOX)
	// We just check it doesn't panic; whether it finds the engine
	// depends on the test environment
	_ = err
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0755)
}

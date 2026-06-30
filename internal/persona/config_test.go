package persona

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_UsesUnifiedAgentsConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		Name: "fable",
		Voice: &VoiceConfig{
			Provider:       "openai",
			Model:          "irodori-tts",
			Voice:          "none",
			BaseURL:        "http://127.0.0.1:8088/v1",
			TimeoutSeconds: 300,
		},
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, AgentsDir), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ConfigPath(tmpDir), data, 0600); err != nil {
		t.Fatal(err)
	}

	got, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got == nil || got.Name != "fable" {
		t.Fatalf("LoadConfig() = %#v, want fable config", got)
	}
	if got.Voice == nil || got.Voice.BaseURL != "http://127.0.0.1:8088/v1" {
		t.Fatalf("voice config = %#v, want OpenAI-compatible base_url", got.Voice)
	}
}

func TestLoadConfig_BrokenUnifiedConfigFallsBackToNil(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, AgentsDir), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ConfigPath(tmpDir), []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil runtime error", err)
	}
	if got != nil {
		t.Fatalf("LoadConfig() = %#v, want nil fallback for broken config", got)
	}
}

func TestLoadConfig_IgnoresLegacyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	legacyDir := filepath.Join(tmpDir, ClaudeDir)
	if err := os.MkdirAll(legacyDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, LegacyPersonaFileName), []byte(`{"name":"legacy"}`), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got != nil {
		t.Fatalf("LoadConfig() = %#v, want nil because legacy files are ignored", got)
	}
}

func TestSaveConfig_WritesUnifiedAgentsConfig(t *testing.T) {
	tmpDir := t.TempDir()
	if err := SaveConfig(tmpDir, &Config{Name: "save-test"}); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	if _, err := os.Stat(ConfigPath(tmpDir)); err != nil {
		t.Fatalf("stat unified config: %v", err)
	}

	got, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got == nil || got.Name != "save-test" {
		t.Fatalf("LoadConfig() = %#v, want saved config", got)
	}
}

func TestValidateConfig_AllowsOpenAICompatibleVoice(t *testing.T) {
	err := ValidateConfig(&Config{
		Name: "valid",
		Voice: &VoiceConfig{
			Provider: "openai",
			BaseURL:  "http://127.0.0.1:8088/v1",
			Model:    "irodori-tts",
		},
	})
	if err != nil {
		t.Fatalf("ValidateConfig() error = %v", err)
	}
}

func TestMigrateConfig_MergesLegacyPersonaAndVoice(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ClaudeDir)
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, LegacyPersonaFileName), []byte(`{
		"name": "fable",
		"voice": { "provider": "openai" },
		"custom_instructions": "speak naturally"
	}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, LegacyVoiceConfigName), []byte(`{
		"default_provider": "openai",
		"providers": {
			"openai": {
				"base_url": "http://127.0.0.1:8088/v1",
				"model": "irodori-tts",
				"voice": "none",
				"format": "wav",
				"timeout_seconds": 300
			}
		},
		"engines": {
			"irodori": { "base_url": "http://127.0.0.1:8088", "health": "openai" }
		}
	}`), 0600); err != nil {
		t.Fatal(err)
	}

	path, err := MigrateConfig(tmpDir, false)
	if err != nil {
		t.Fatalf("MigrateConfig() error = %v", err)
	}
	if path != ConfigPath(tmpDir) {
		t.Fatalf("path = %s, want %s", path, ConfigPath(tmpDir))
	}

	got, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got == nil || got.Name != "fable" || got.Voice == nil {
		t.Fatalf("migrated config = %#v, want fable voice config", got)
	}
	if got.Voice.BaseURL != "http://127.0.0.1:8088/v1" || got.Voice.TimeoutSeconds != 300 {
		t.Fatalf("voice = %#v, want migrated provider settings", got.Voice)
	}
	if _, ok := got.Engines["irodori"]; !ok {
		t.Fatalf("engines = %#v, want irodori", got.Engines)
	}
}

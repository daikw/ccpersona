package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// chdir switches to dir for the duration of the test, restoring the original
// working directory on cleanup.
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

// writeProjectConfig writes a project-local .claude/config.json under dir.
func writeProjectConfig(t *testing.T, dir, contents string) {
	t.Helper()
	cdir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(cdir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cdir, "config.json"), []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadEngineConfig_MissingFileIsNotAnError(t *testing.T) {
	td := t.TempDir()
	// Point HOME at the empty temp dir so the global candidate is also absent.
	t.Setenv("HOME", td)
	chdir(t, td)

	cfg, err := loadEngineConfig()
	if err != nil {
		t.Fatalf("loadEngineConfig() error = %v, want nil for missing config", err)
	}
	if cfg != nil {
		t.Errorf("loadEngineConfig() = %v, want nil", cfg)
	}
}

func TestLoadEngineConfig_ParseErrorSurfaces(t *testing.T) {
	td := t.TempDir()
	t.Setenv("HOME", td)
	chdir(t, td)

	writeProjectConfig(t, td, `{ this is not valid json `)

	_, err := loadEngineConfig()
	if err == nil {
		t.Fatal("loadEngineConfig() expected error for malformed JSON, got nil")
	}
	if !strings.Contains(err.Error(), "failed to load config") {
		t.Errorf("error = %v, want load-config wrapper", err)
	}
}

func TestLoadEngineRegistry_ValidConfig(t *testing.T) {
	td := t.TempDir()
	t.Setenv("HOME", td)
	chdir(t, td)

	writeProjectConfig(t, td, `{
      "engines": {
        "irodori": { "base_url": "http://127.0.0.1:8088", "health": "openai", "command": "uv", "args": ["run", "irodori-tts-server"] }
      }
    }`)

	reg, err := loadEngineRegistry()
	if err != nil {
		t.Fatalf("loadEngineRegistry() error = %v", err)
	}
	if _, ok := reg.Get("irodori"); !ok {
		t.Errorf("irodori not present in registry; names = %v", reg.Names())
	}
}

func TestLoadEngineRegistry_InvalidNamePropagates(t *testing.T) {
	td := t.TempDir()
	t.Setenv("HOME", td)
	chdir(t, td)

	writeProjectConfig(t, td, `{
      "engines": {
        "../../evil": { "base_url": "http://x", "command": "y" }
      }
    }`)

	if _, err := loadEngineRegistry(); err == nil {
		t.Fatal("loadEngineRegistry() expected error for invalid engine name, got nil")
	}
}

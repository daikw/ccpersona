package engine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/rs/zerolog/log"
)

// EngineType represents a TTS engine type.
type EngineType string

const (
	VOICEVOX    EngineType = "voicevox"
	AivisSpeech EngineType = "aivisspeech"
)

// AllEngineTypes returns all supported engine types.
func AllEngineTypes() []EngineType {
	return []EngineType{VOICEVOX, AivisSpeech}
}

// ParseEngineType parses a string into an EngineType.
// Returns all engine types for "all".
func ParseEngineType(s string) ([]EngineType, error) {
	switch EngineType(s) {
	case VOICEVOX:
		return []EngineType{VOICEVOX}, nil
	case AivisSpeech:
		return []EngineType{AivisSpeech}, nil
	}
	if s == "all" {
		return AllEngineTypes(), nil
	}
	return nil, fmt.Errorf("unknown engine type: %s (use voicevox, aivisspeech, or all)", s)
}

// EngineInfo holds discovered engine information.
type EngineInfo struct {
	Type       EngineType
	BinaryPath string
	Port       int
	Label      string // service label (e.g. "com.voicevox.engine")
}

func defaultPort(t EngineType) int {
	switch t {
	case VOICEVOX:
		return 50021
	case AivisSpeech:
		return 10101
	default:
		return 0
	}
}

func serviceLabel(t EngineType) string {
	switch t {
	case VOICEVOX:
		return "com.voicevox.engine"
	case AivisSpeech:
		return "com.aivisspeech.engine"
	default:
		return ""
	}
}

// SystemdUnit returns the systemd unit name for the engine.
func SystemdUnit(t EngineType) string {
	switch t {
	case VOICEVOX:
		return "voicevox-engine.service"
	case AivisSpeech:
		return "aivisspeech-engine.service"
	default:
		return ""
	}
}

// DiscoverEngine searches for the engine binary on the system.
func DiscoverEngine(t EngineType) (*EngineInfo, error) {
	info := &EngineInfo{
		Type:  t,
		Port:  defaultPort(t),
		Label: serviceLabel(t),
	}

	// 1. Environment variable override
	envKey := envVarName(t)
	if p := os.Getenv(envKey); p != "" {
		if _, err := os.Stat(p); err == nil {
			info.BinaryPath = p
			log.Debug().Str("engine", string(t)).Str("path", p).Msg("Found via env var")
			return info, nil
		}
		log.Warn().Str("env", envKey).Str("path", p).Msg("Env var path does not exist")
	}

	// 2. exec.LookPath
	for _, name := range lookPathNames(t) {
		if p, err := exec.LookPath(name); err == nil {
			info.BinaryPath = p
			log.Debug().Str("engine", string(t)).Str("path", p).Msg("Found in PATH")
			return info, nil
		}
	}

	// 3. Platform-specific search
	for _, p := range platformSearchPaths(t) {
		if _, err := os.Stat(p); err == nil {
			info.BinaryPath = p
			log.Debug().Str("engine", string(t)).Str("path", p).Msg("Found at known path")
			return info, nil
		}
	}

	return nil, fmt.Errorf("engine %s not found (set %s to specify the path)", t, envKey)
}

func envVarName(t EngineType) string {
	switch t {
	case VOICEVOX:
		return "VOICEVOX_ENGINE_PATH"
	case AivisSpeech:
		return "AIVISSPEECH_ENGINE_PATH"
	default:
		return ""
	}
}

func lookPathNames(t EngineType) []string {
	switch t {
	case VOICEVOX:
		return []string{"voicevox-engine", "voicevox_engine", "run"}
	case AivisSpeech:
		return []string{"aivisspeech-engine", "aivisspeech_engine"}
	default:
		return nil
	}
}

func platformSearchPaths(t EngineType) []string {
	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "darwin":
		return macSearchPaths(t, home)
	case "linux":
		return linuxSearchPaths(t, home)
	default:
		return nil
	}
}

func macSearchPaths(t EngineType, home string) []string {
	switch t {
	case VOICEVOX:
		return []string{
			"/Applications/VOICEVOX.app/Contents/Resources/vv-engine/run",
			filepath.Join(home, "Applications/VOICEVOX.app/Contents/Resources/vv-engine/run"),
		}
	case AivisSpeech:
		return []string{
			"/Applications/AivisSpeech.app/Contents/Resources/AivisSpeech-Engine/run",
			filepath.Join(home, "Applications/AivisSpeech.app/Contents/Resources/AivisSpeech-Engine/run"),
		}
	default:
		return nil
	}
}

func linuxSearchPaths(t EngineType, home string) []string {
	var paths []string
	var names []string
	switch t {
	case VOICEVOX:
		names = []string{"voicevox-engine", "run"}
	case AivisSpeech:
		names = []string{"aivisspeech-engine", "run"}
	}

	dirs := []string{
		"/usr/local/bin",
		"/usr/bin",
		"/opt/voicevox_engine",
		"/opt/aivisspeech_engine",
		filepath.Join(home, ".local/bin"),
	}

	for _, dir := range dirs {
		for _, name := range names {
			paths = append(paths, filepath.Join(dir, name))
		}
	}
	return paths
}

// PortString returns the port as a string.
func (e *EngineInfo) PortString() string {
	return strconv.Itoa(e.Port)
}

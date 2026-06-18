package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

// Health check types.
const (
	HealthVoicevox = "voicevox" // GET /version
	HealthOpenAI   = "openai"   // GET /v1/models
)

// EngineDef is the canonical definition of a manageable TTS engine. It unifies
// built-in engines (VOICEVOX / AivisSpeech) and user-defined engines declared
// in the voice config file.
type EngineDef struct {
	Name        string // unique engine name (the identifier used as a CLI argument)
	DisplayName string
	BaseURL     string // health check base, e.g. http://127.0.0.1:8088
	HealthType  string // "voicevox" (GET /version) | "openai" (GET /v1/models)

	// Service spec. When Command is empty the engine is "external" (not managed
	// by ccpersona): only status (health check) is available, no install/start/stop.
	Command string
	Args    []string
	Dir     string // working directory (~ expanded)
	Env     map[string]string

	// builtinType is non-empty for built-in engines; it preserves the existing
	// launchd label / systemd unit naming and discovery behaviour.
	builtinType EngineType
}

// Managed reports whether ccpersona owns a service identity for the engine and
// can manage lifecycle operations. Built-ins remain manageable even when binary
// discovery fails, so users can stop or uninstall a previously installed
// service after the app binary moved or was removed. Install still requires a
// Command because rendering a new service file needs the executable path.
func (d *EngineDef) Managed() bool {
	return d.Builtin() || d.Command != ""
}

// Builtin reports whether the engine is a built-in (VOICEVOX/AivisSpeech).
func (d *EngineDef) Builtin() bool {
	return d.builtinType != ""
}

// ServiceLabel returns the launchd label / systemd unit base name used to
// identify the managed service. Built-ins keep their historical names.
func (d *EngineDef) ServiceLabel() string {
	if d.builtinType != "" {
		return serviceLabel(d.builtinType)
	}
	return "com.ccpersona.engine." + d.Name
}

// SystemdUnitName returns the systemd unit filename for the engine.
func (d *EngineDef) SystemdUnitName() string {
	if d.builtinType != "" {
		return SystemdUnit(d.builtinType)
	}
	return "ccpersona-engine-" + d.Name + ".service"
}

// HealthURL builds the health-check URL for the engine based on its HealthType.
func (d *EngineDef) HealthURL() string {
	return HealthURL(d.BaseURL, d.HealthType)
}

// HealthURL constructs the health-check URL for the given base URL and health
// type. It is a pure function to keep it independently testable.
func HealthURL(baseURL, healthType string) string {
	base := strings.TrimRight(baseURL, "/")
	switch healthType {
	case HealthOpenAI:
		return base + "/v1/models"
	case HealthVoicevox:
		return base + "/version"
	default:
		return base + "/v1/models"
	}
}

// builtinDef constructs the EngineDef for a built-in engine by discovering its
// binary. Discovery may fail (binary not installed); in that case Command is
// left empty but the def is still returned so status can report "not found".
func builtinDef(t EngineType) EngineDef {
	def := EngineDef{
		Name:        string(t),
		DisplayName: builtinDisplayName(t),
		BaseURL:     fmt.Sprintf("http://127.0.0.1:%d", defaultPort(t)),
		HealthType:  HealthVoicevox,
		builtinType: t,
	}
	if info, err := DiscoverEngine(t); err == nil {
		def.Command = info.BinaryPath
		def.Args = []string{"--host", "127.0.0.1", "--port", info.PortString()}
	}
	return def
}

func builtinDisplayName(t EngineType) string {
	switch t {
	case VOICEVOX:
		return "VOICEVOX"
	case AivisSpeech:
		return "AivisSpeech"
	default:
		return string(t)
	}
}

// Registry holds the set of known engines, keyed by name, preserving a stable
// ordering (built-ins first in declaration order, then user-defined sorted by
// name).
type Registry struct {
	order []string
	defs  map[string]*EngineDef
}

// Names returns engine names in stable order.
func (r *Registry) Names() []string {
	return append([]string(nil), r.order...)
}

// Get returns the engine definition for a name, or false if unknown.
func (r *Registry) Get(name string) (*EngineDef, bool) {
	d, ok := r.defs[name]
	return d, ok
}

// All returns all engine definitions in stable order.
func (r *Registry) All() []*EngineDef {
	out := make([]*EngineDef, 0, len(r.order))
	for _, n := range r.order {
		out = append(out, r.defs[n])
	}
	return out
}

// Resolve maps a CLI target argument to engine definitions. "all" (or empty)
// expands to the built-in engines only, preserving the historical behaviour of
// the `all` keyword. A user-defined name resolves to that single engine.
func (r *Registry) Resolve(target string) ([]*EngineDef, error) {
	if target == "" || target == "all" {
		out := make([]*EngineDef, 0, 2)
		for _, t := range AllEngineTypes() {
			if d, ok := r.defs[string(t)]; ok {
				out = append(out, d)
			}
		}
		return out, nil
	}
	if d, ok := r.defs[target]; ok {
		return []*EngineDef{d}, nil
	}
	return nil, fmt.Errorf("unknown engine: %s (available: %s)", target, strings.Join(r.Names(), ", "))
}

// UserEngineConfig is the engine definition supplied by the user in the voice
// config file. It mirrors voice.EngineUserConfig but lives in this package to
// avoid an engine -> voice import (the caller converts and passes these in).
type UserEngineConfig struct {
	BaseURL string
	Health  string // "voicevox" | "openai"; defaults to "openai" when empty
	Command string
	Args    []string
	Dir     string
	Env     map[string]string
}

// BuildRegistry constructs a registry from the built-in engines plus the given
// user-defined engines. A user engine whose name collides with a built-in is
// rejected (override is not allowed, to avoid confusion).
func BuildRegistry(userEngines map[string]UserEngineConfig) (*Registry, error) {
	r := &Registry{defs: map[string]*EngineDef{}}

	for _, t := range AllEngineTypes() {
		def := builtinDef(t)
		r.defs[def.Name] = &def
		r.order = append(r.order, def.Name)
	}

	names := make([]string, 0, len(userEngines))
	for name := range userEngines {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if _, exists := r.defs[name]; exists {
			return nil, fmt.Errorf("engine %q conflicts with a built-in engine; user-defined engines must use a different name", name)
		}
		if !engineNameRe.MatchString(name) {
			return nil, fmt.Errorf("engine %q: invalid name; must match %s", name, engineNamePattern)
		}
		uc := userEngines[name]

		// Fail closed on control characters in any user-controlled field. This
		// is the root defense against systemd/launchd directive injection via
		// embedded newlines (and NUL truncation).
		if err := rejectControlChars(name, uc); err != nil {
			return nil, err
		}

		health := uc.Health
		if health == "" {
			health = HealthOpenAI
		}
		if health != HealthOpenAI && health != HealthVoicevox {
			return nil, fmt.Errorf("engine %q: invalid health type %q (use %q or %q)", name, uc.Health, HealthVoicevox, HealthOpenAI)
		}

		dir, err := expandHome(uc.Dir)
		if err != nil {
			return nil, fmt.Errorf("engine %q: %w", name, err)
		}

		def := EngineDef{
			Name:        name,
			DisplayName: name,
			BaseURL:     uc.BaseURL,
			HealthType:  health,
			Command:     uc.Command,
			Args:        append([]string(nil), uc.Args...),
			Dir:         dir,
			Env:         uc.Env,
		}
		r.defs[name] = &def
		r.order = append(r.order, name)
	}

	return r, nil
}

const engineNamePattern = `^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`

var engineNameRe = regexp.MustCompile(engineNamePattern)

// hasControlChars reports whether s contains characters that could break out of
// a single line in a systemd unit / launchd plist context, or truncate a path.
func hasControlChars(s string) bool {
	return strings.ContainsAny(s, "\r\n\x00")
}

// rejectControlChars returns an error if any user-controlled field of the engine
// definition contains a control character (CR, LF, NUL).
func rejectControlChars(name string, uc UserEngineConfig) error {
	check := func(field, val string) error {
		if hasControlChars(val) {
			return fmt.Errorf("engine %q: field %s contains a control character (newline/NUL not allowed)", name, field)
		}
		return nil
	}
	if err := check("command", uc.Command); err != nil {
		return err
	}
	if err := check("dir", uc.Dir); err != nil {
		return err
	}
	for i, a := range uc.Args {
		if err := check(fmt.Sprintf("args[%d]", i), a); err != nil {
			return err
		}
	}
	for k, v := range uc.Env {
		if err := check("env key", k); err != nil {
			return err
		}
		if err := check("env["+k+"]", v); err != nil {
			return err
		}
	}
	return nil
}

// expandHome expands a leading ~ to the user's home directory. When the input
// is tilde-rooted, the expanded path is required to stay within the home
// directory: a value like "~/../../etc" that normalizes outside home is
// rejected to prevent traversal. Absolute or relative paths without a ~ prefix
// are returned unchanged (the user may legitimately point at a system path).
func expandHome(p string) (string, error) {
	if p == "" {
		return p, nil
	}
	if p != "~" && !strings.HasPrefix(p, "~/") {
		return p, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot expand ~ in dir %q: %w", p, err)
	}

	var expanded string
	if p == "~" {
		expanded = home
	} else {
		expanded = filepath.Join(home, p[2:])
	}

	rel, err := filepath.Rel(home, expanded)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("dir %q escapes the home directory", p)
	}
	return expanded, nil
}

// SupportsServiceManagement reports whether the current platform has a service
// manager implementation.
func SupportsServiceManagement() bool {
	switch runtime.GOOS {
	case "darwin", "linux":
		return true
	default:
		return false
	}
}

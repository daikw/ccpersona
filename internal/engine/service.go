package engine

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// ServiceStatus represents the status of a managed engine service.
type ServiceStatus struct {
	Name      string
	Installed bool
	Running   bool
	PID       int
	Label     string // launchd label or systemd unit name
}

// ServiceManager manages TTS engine background services.
type ServiceManager interface {
	// Install deploys the service configuration file and enables the service.
	Install(def *EngineDef) error
	// Uninstall stops and removes the service configuration.
	Uninstall(def *EngineDef) error
	// Start starts the service.
	Start(def *EngineDef) error
	// Stop stops the service.
	Stop(def *EngineDef) error
	// Status returns the service status.
	Status(def *EngineDef) (*ServiceStatus, error)
}

// NewServiceManager returns the appropriate platform implementation.
func NewServiceManager() (ServiceManager, error) {
	switch runtime.GOOS {
	case "darwin":
		return newLaunchdManager()
	case "linux":
		return newSystemdManager()
	default:
		return nil, fmt.Errorf("unsupported platform: %s (supported: darwin, linux)", runtime.GOOS)
	}
}

// errNotManaged is returned when an unmanaged (external) engine is asked to
// perform a service-management operation.
func errNotManaged(def *EngineDef) error {
	return fmt.Errorf("engine %q is externally managed; define a command in config to make it manageable", def.Name)
}

// servicePath joins dir and filename and verifies the result stays directly
// within dir. Engine names are already constrained by BuildRegistry, so this is
// defense-in-depth against any future bypass of name validation.
func servicePath(dir, filename string) (string, error) {
	p := filepath.Join(dir, filename)
	rel, err := filepath.Rel(dir, p)
	if err != nil || rel != filename || strings.ContainsRune(rel, filepath.Separator) {
		return "", fmt.Errorf("refusing to write service file outside %s: %q", dir, filename)
	}
	return p, nil
}

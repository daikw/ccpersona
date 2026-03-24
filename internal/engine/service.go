package engine

import (
	"embed"
	"fmt"
	"runtime"
)

//go:embed templates/*
var templateFS embed.FS

// ServiceStatus represents the status of a managed engine service.
type ServiceStatus struct {
	Engine    EngineType
	Installed bool
	Running   bool
	PID       int
	Label     string // launchd label or systemd unit name
}

// ServiceManager manages TTS engine background services.
type ServiceManager interface {
	// Install deploys the service configuration file and enables the service.
	Install(info *EngineInfo) error
	// Uninstall stops and removes the service configuration.
	Uninstall(engineType EngineType) error
	// Start starts the service.
	Start(engineType EngineType) error
	// Stop stops the service.
	Stop(engineType EngineType) error
	// Status returns the service status.
	Status(engineType EngineType) (*ServiceStatus, error)
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

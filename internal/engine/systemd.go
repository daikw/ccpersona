package engine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

type systemdManager struct {
	unitDir string // ~/.config/systemd/user
}

func newSystemdManager() (*systemdManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}
	return &systemdManager{
		unitDir: filepath.Join(home, ".config", "systemd", "user"),
	}, nil
}

func (m *systemdManager) unitPath(def *EngineDef) (string, error) {
	return servicePath(m.unitDir, def.SystemdUnitName())
}

func (m *systemdManager) Install(def *EngineDef) error {
	if !def.Managed() {
		return errNotManaged(def)
	}

	unitPath, err := m.unitPath(def)
	if err != nil {
		return err
	}

	contents := RenderSystemdUnit(def)

	if err := os.MkdirAll(m.unitDir, 0755); err != nil {
		return fmt.Errorf("failed to create systemd user directory: %w", err)
	}

	if err := os.WriteFile(unitPath, []byte(contents), 0644); err != nil {
		return fmt.Errorf("failed to write unit file: %w", err)
	}

	log.Info().Str("path", unitPath).Msg("Installed systemd user unit")

	// Reload and enable
	if out, err := exec.Command("systemctl", "--user", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w: %s", err, string(out))
	}
	unit := def.SystemdUnitName()
	if out, err := exec.Command("systemctl", "--user", "enable", unit).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable unit: %w: %s", err, string(out))
	}

	return nil
}

func (m *systemdManager) Uninstall(def *EngineDef) error {
	if !def.Managed() {
		return errNotManaged(def)
	}
	unitPath, err := m.unitPath(def)
	if err != nil {
		return err
	}
	unit := def.SystemdUnitName()

	// Disable and stop
	_ = exec.Command("systemctl", "--user", "stop", unit).Run()
	_ = exec.Command("systemctl", "--user", "disable", unit).Run()

	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove unit file: %w", err)
	}

	// Reload
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()

	log.Info().Str("path", unitPath).Msg("Uninstalled systemd user unit")
	return nil
}

func (m *systemdManager) Start(def *EngineDef) error {
	if !def.Managed() {
		return errNotManaged(def)
	}
	unit := def.SystemdUnitName()
	out, err := exec.Command("systemctl", "--user", "start", unit).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl start failed: %w: %s", err, string(out))
	}
	return nil
}

func (m *systemdManager) Stop(def *EngineDef) error {
	if !def.Managed() {
		return errNotManaged(def)
	}
	unit := def.SystemdUnitName()
	out, err := exec.Command("systemctl", "--user", "stop", unit).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl stop failed: %w: %s", err, string(out))
	}
	return nil
}

func (m *systemdManager) Status(def *EngineDef) (*ServiceStatus, error) {
	unit := def.SystemdUnitName()
	status := &ServiceStatus{
		Name:  def.Name,
		Label: unit,
	}

	unitPath, err := m.unitPath(def)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(unitPath); err == nil {
		status.Installed = true
	}

	out, err := exec.Command("systemctl", "--user", "is-active", unit).CombinedOutput()
	if err == nil && strings.TrimSpace(string(out)) == "active" {
		status.Running = true
	}

	// Get PID
	out, err = exec.Command("systemctl", "--user", "show", unit, "--property=MainPID").CombinedOutput()
	if err == nil {
		line := strings.TrimSpace(string(out))
		if strings.HasPrefix(line, "MainPID=") {
			pidStr := strings.TrimPrefix(line, "MainPID=")
			var pid int
			if _, err := fmt.Sscanf(pidStr, "%d", &pid); err == nil && pid > 0 {
				status.PID = pid
			}
		}
	}

	return status, nil
}

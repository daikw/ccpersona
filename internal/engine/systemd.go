package engine

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

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

func (m *systemdManager) unitPath(t EngineType) string {
	return filepath.Join(m.unitDir, SystemdUnit(t))
}

func (m *systemdManager) templateName(t EngineType) string {
	return "templates/" + SystemdUnit(t)
}

func (m *systemdManager) Install(info *EngineInfo) error {
	tmplData, err := templateFS.ReadFile(m.templateName(info.Type))
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("unit").Parse(string(tmplData))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	data := struct {
		BinaryPath string
		Port       string
	}{
		BinaryPath: info.BinaryPath,
		Port:       info.PortString(),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	if err := os.MkdirAll(m.unitDir, 0755); err != nil {
		return fmt.Errorf("failed to create systemd user directory: %w", err)
	}

	unitPath := m.unitPath(info.Type)
	if err := os.WriteFile(unitPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write unit file: %w", err)
	}

	log.Info().Str("path", unitPath).Msg("Installed systemd user unit")

	// Reload and enable
	if out, err := exec.Command("systemctl", "--user", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w: %s", err, string(out))
	}
	unit := SystemdUnit(info.Type)
	if out, err := exec.Command("systemctl", "--user", "enable", unit).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable unit: %w: %s", err, string(out))
	}

	return nil
}

func (m *systemdManager) Uninstall(t EngineType) error {
	unit := SystemdUnit(t)

	// Disable and stop
	_ = exec.Command("systemctl", "--user", "stop", unit).Run()
	_ = exec.Command("systemctl", "--user", "disable", unit).Run()

	unitPath := m.unitPath(t)
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove unit file: %w", err)
	}

	// Reload
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()

	log.Info().Str("path", unitPath).Msg("Uninstalled systemd user unit")
	return nil
}

func (m *systemdManager) Start(t EngineType) error {
	unit := SystemdUnit(t)
	out, err := exec.Command("systemctl", "--user", "start", unit).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl start failed: %w: %s", err, string(out))
	}
	return nil
}

func (m *systemdManager) Stop(t EngineType) error {
	unit := SystemdUnit(t)
	out, err := exec.Command("systemctl", "--user", "stop", unit).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl stop failed: %w: %s", err, string(out))
	}
	return nil
}

func (m *systemdManager) Status(t EngineType) (*ServiceStatus, error) {
	unit := SystemdUnit(t)
	status := &ServiceStatus{
		Engine: t,
		Label:  unit,
	}

	if _, err := os.Stat(m.unitPath(t)); err == nil {
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

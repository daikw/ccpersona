package engine

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/rs/zerolog/log"
)

type launchdManager struct {
	agentDir string // ~/Library/LaunchAgents
}

func newLaunchdManager() (*launchdManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}
	return &launchdManager{
		agentDir: filepath.Join(home, "Library", "LaunchAgents"),
	}, nil
}

func (m *launchdManager) plistPath(t EngineType) string {
	return filepath.Join(m.agentDir, serviceLabel(t)+".plist")
}

func (m *launchdManager) templateName(t EngineType) string {
	return "templates/" + serviceLabel(t) + ".plist"
}

func (m *launchdManager) Install(info *EngineInfo) error {
	// Read and render template
	tmplData, err := templateFS.ReadFile(m.templateName(info.Type))
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("plist").Parse(string(tmplData))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, "Library", "Logs", "ccpersona")

	data := struct {
		Label      string
		BinaryPath string
		Port       string
		LogDir     string
	}{
		Label:      info.Label,
		BinaryPath: info.BinaryPath,
		Port:       info.PortString(),
		LogDir:     logDir,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Ensure directories exist
	if err := os.MkdirAll(m.agentDir, 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Write plist
	plistPath := m.plistPath(info.Type)
	if err := os.WriteFile(plistPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write plist: %w", err)
	}

	log.Info().Str("path", plistPath).Msg("Installed LaunchAgent plist")

	// Load the agent
	if err := m.launchctlLoad(plistPath); err != nil {
		log.Warn().Err(err).Msg("Failed to load LaunchAgent (may need manual load)")
	}

	return nil
}

func (m *launchdManager) Uninstall(t EngineType) error {
	plistPath := m.plistPath(t)

	// Unload first (ignore errors if not loaded)
	_ = m.launchctlUnload(plistPath)

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plist: %w", err)
	}

	log.Info().Str("path", plistPath).Msg("Uninstalled LaunchAgent plist")
	return nil
}

func (m *launchdManager) Start(t EngineType) error {
	label := serviceLabel(t)
	out, err := exec.Command("launchctl", "start", label).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl start failed: %w: %s", err, string(out))
	}
	return nil
}

func (m *launchdManager) Stop(t EngineType) error {
	// KeepAlive=true のため stop だけでは再起動される。unload → load で停止状態にする。
	plistPath := m.plistPath(t)
	if _, err := os.Stat(plistPath); err != nil {
		// plist が無ければ stop だけ試みる
		label := serviceLabel(t)
		out, err := exec.Command("launchctl", "stop", label).CombinedOutput()
		if err != nil {
			return fmt.Errorf("launchctl stop failed: %w: %s", err, string(out))
		}
		return nil
	}
	// unload でプロセスを停止
	_ = m.launchctlUnload(plistPath)
	return nil
}

func (m *launchdManager) Status(t EngineType) (*ServiceStatus, error) {
	label := serviceLabel(t)
	status := &ServiceStatus{
		Engine: t,
		Label:  label,
	}

	// Check if plist exists
	if _, err := os.Stat(m.plistPath(t)); err == nil {
		status.Installed = true
	}

	// Check if service is loaded and running
	out, err := exec.Command("launchctl", "list").CombinedOutput()
	if err != nil {
		return status, nil
	}

	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, label) {
			// launchctl list format: PID\tStatus\tLabel
			// PID is "-" when the service is loaded but not running
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				if pid, err := strconv.Atoi(fields[0]); err == nil && pid > 0 {
					status.Running = true
					status.PID = pid
				}
				// PID == "-" means loaded but not running → Running stays false
			}
			break
		}
	}

	return status, nil
}

func (m *launchdManager) launchctlLoad(plistPath string) error {
	out, err := exec.Command("launchctl", "load", plistPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(out))
	}
	return nil
}

func (m *launchdManager) launchctlUnload(plistPath string) error {
	out, err := exec.Command("launchctl", "unload", plistPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(out))
	}
	return nil
}

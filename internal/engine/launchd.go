package engine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

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

func (m *launchdManager) plistPath(def *EngineDef) (string, error) {
	return servicePath(m.agentDir, def.ServiceLabel()+".plist")
}

func (m *launchdManager) Install(def *EngineDef) error {
	if !def.Managed() {
		return errNotManaged(def)
	}

	plistPath, err := m.plistPath(def)
	if err != nil {
		return err
	}

	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, "Library", "Logs", "ccpersona")

	contents := RenderPlist(def, logDir)

	// Ensure directories exist
	if err := os.MkdirAll(m.agentDir, 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Write plist
	if err := os.WriteFile(plistPath, []byte(contents), 0644); err != nil {
		return fmt.Errorf("failed to write plist: %w", err)
	}

	log.Info().Str("path", plistPath).Msg("Installed LaunchAgent plist")

	// Load the agent
	if err := m.launchctlLoad(plistPath); err != nil {
		return fmt.Errorf("failed to load LaunchAgent: %w", err)
	}

	return nil
}

func (m *launchdManager) Uninstall(def *EngineDef) error {
	if !def.Managed() {
		return errNotManaged(def)
	}
	plistPath, err := m.plistPath(def)
	if err != nil {
		return err
	}

	// Unload first (ignore errors if not loaded)
	_ = m.launchctlUnload(plistPath)

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plist: %w", err)
	}

	log.Info().Str("path", plistPath).Msg("Uninstalled LaunchAgent plist")
	return nil
}

func (m *launchdManager) Start(def *EngineDef) error {
	if !def.Managed() {
		return errNotManaged(def)
	}
	// Stop() が unload するため、まず plist を load し直す必要がある
	plistPath, err := m.plistPath(def)
	if err != nil {
		return err
	}
	if _, err := os.Stat(plistPath); err == nil {
		// load は既に loaded でもエラーにならない（warning が出るだけ）
		_ = m.launchctlLoad(plistPath)
	}

	label := def.ServiceLabel()
	out, err := exec.Command("launchctl", "start", label).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl start failed: %w: %s", err, string(out))
	}
	return nil
}

func (m *launchdManager) Stop(def *EngineDef) error {
	if !def.Managed() {
		return errNotManaged(def)
	}
	// KeepAlive=true のため stop だけでは再起動される。unload → load で停止状態にする。
	plistPath, err := m.plistPath(def)
	if err != nil {
		return err
	}
	if _, err := os.Stat(plistPath); err != nil {
		// plist が無ければ stop だけ試みる
		label := def.ServiceLabel()
		out, err := exec.Command("launchctl", "stop", label).CombinedOutput()
		if err != nil {
			return fmt.Errorf("launchctl stop failed: %w: %s", err, string(out))
		}
		return nil
	}
	// unload でプロセスを停止
	if err := m.launchctlUnload(plistPath); err != nil {
		log.Warn().Err(err).Msg("Failed to unload plist during stop")
	}
	return nil
}

func (m *launchdManager) Status(def *EngineDef) (*ServiceStatus, error) {
	label := def.ServiceLabel()
	status := &ServiceStatus{
		Name:  def.Name,
		Label: label,
	}

	// Check if plist exists
	plistPath, err := m.plistPath(def)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(plistPath); err == nil {
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

package persona

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// SessionManager handles Claude Code session tracking
type SessionManager struct {
	homeDir   string
	sessionID string
}

// NewSessionManager creates a new session manager
func NewSessionManager() (*SessionManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Get session ID from environment or use "unknown"
	sessionID := os.Getenv("CLAUDE_SESSION_ID")
	if sessionID == "" {
		sessionID = "unknown"
	}

	return &SessionManager{
		homeDir:   homeDir,
		sessionID: sessionID,
	}, nil
}

// IsNewSession checks if this is a new Claude Code session
func (sm *SessionManager) IsNewSession() bool {
	markerPath := sm.getSessionMarkerPath()
	_, err := os.Stat(markerPath)
	return os.IsNotExist(err)
}

// MarkSessionStarted creates a session marker file
func (sm *SessionManager) MarkSessionStarted() error {
	markerPath := sm.getSessionMarkerPath()

	// Ensure directory exists
	dir := filepath.Dir(markerPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	// Create marker file
	file, err := os.Create(markerPath)
	if err != nil {
		return fmt.Errorf("failed to create session marker: %w", err)
	}
	_ = file.Close()

	log.Debug().Str("session_id", sm.sessionID).Msg("Session marked as started")
	return nil
}

// CleanupOldSessions removes session markers older than 24 hours
func (sm *SessionManager) CleanupOldSessions() error {
	sessionDir := filepath.Join(sm.homeDir, ".claude")

	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read session directory: %w", err)
	}

	cutoff := time.Now().Add(-24 * time.Hour)
	cleanedCount := 0

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".session_") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			path := filepath.Join(sessionDir, entry.Name())
			if err := os.Remove(path); err == nil {
				cleanedCount++
			}
		}
	}

	if cleanedCount > 0 {
		log.Debug().Int("count", cleanedCount).Msg("Cleaned up old session markers")
	}

	return nil
}

// getSessionMarkerPath returns the path to the session marker file
func (sm *SessionManager) getSessionMarkerPath() string {
	return filepath.Join(sm.homeDir, ".claude", fmt.Sprintf(".session_%s", sm.sessionID))
}

// HandleSessionStart is the main entry point for hook functionality
func HandleSessionStart() error {
	// Create session manager
	sessionMgr, err := NewSessionManager()
	if err != nil {
		return fmt.Errorf("failed to create session manager: %w", err)
	}

	// Check if this is a new session
	if !sessionMgr.IsNewSession() {
		log.Debug().Msg("Not a new session, skipping persona application")
		return nil
	}

	// Mark session as started
	if err := sessionMgr.MarkSessionStarted(); err != nil {
		log.Warn().Err(err).Msg("Failed to mark session as started")
	}

	// Clean up old sessions
	if err := sessionMgr.CleanupOldSessions(); err != nil {
		log.Warn().Err(err).Msg("Failed to cleanup old sessions")
	}

	// Check if project has persona configuration
	projectDir, _ := os.Getwd()
	config, err := LoadConfig(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if config == nil {
		log.Debug().Msg("No project persona configuration found")
		return nil
	}

	log.Info().Str("persona", config.Name).Msg("Found project persona configuration")

	// Apply the persona
	manager, err := NewManager()
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}

	if err := manager.ApplyPersona(config.Name); err != nil {
		return fmt.Errorf("failed to apply persona: %w", err)
	}

	// Output success message
	fmt.Printf("🎭 人格を適用したのだ: %s\n", config.Name)

	return nil
}

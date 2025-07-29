package persona

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionManager(t *testing.T) {
	// Set up test environment
	tmpDir, err := os.MkdirTemp("", "ccpersona-session-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() {
		_ = os.Setenv("HOME", originalHome)
	}()

	// Set test session ID
	_ = os.Setenv("CLAUDE_SESSION_ID", "test-session-123")
	defer func() {
		_ = os.Unsetenv("CLAUDE_SESSION_ID")
	}()

	t.Run("NewSessionManager", func(t *testing.T) {
		sm, err := NewSessionManager()
		if err != nil {
			t.Fatalf("Failed to create session manager: %v", err)
		}

		if sm.sessionID != "test-session-123" {
			t.Errorf("Expected session ID 'test-session-123', got '%s'", sm.sessionID)
		}
	})

	t.Run("IsNewSession", func(t *testing.T) {
		sm := &SessionManager{
			homeDir:   tmpDir,
			sessionID: "test-new",
		}

		// First check should return true (new session)
		if !sm.IsNewSession() {
			t.Error("Expected new session")
		}

		// Create marker
		if err := sm.MarkSessionStarted(); err != nil {
			t.Fatalf("Failed to mark session: %v", err)
		}

		// Second check should return false (existing session)
		if sm.IsNewSession() {
			t.Error("Expected existing session")
		}
	})

	t.Run("CleanupOldSessions", func(t *testing.T) {
		sm := &SessionManager{
			homeDir:   tmpDir,
			sessionID: "test-cleanup",
		}

		sessionDir := filepath.Join(tmpDir, ".claude")
		if err := os.MkdirAll(sessionDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Create old session marker (simulate 25 hours old)
		oldMarker := filepath.Join(sessionDir, ".session_old")
		if err := os.WriteFile(oldMarker, []byte("old"), 0644); err != nil {
			t.Fatal(err)
		}

		// Modify time to be 25 hours ago
		oldTime := time.Now().Add(-25 * time.Hour)
		if err := os.Chtimes(oldMarker, oldTime, oldTime); err != nil {
			t.Fatal(err)
		}

		// Create new session marker
		newMarker := filepath.Join(sessionDir, ".session_new")
		if err := os.WriteFile(newMarker, []byte("new"), 0644); err != nil {
			t.Fatal(err)
		}

		// Run cleanup
		if err := sm.CleanupOldSessions(); err != nil {
			t.Fatalf("Cleanup failed: %v", err)
		}

		// Check old marker was removed
		if _, err := os.Stat(oldMarker); !os.IsNotExist(err) {
			t.Error("Old session marker should have been removed")
		}

		// Check new marker still exists
		if _, err := os.Stat(newMarker); os.IsNotExist(err) {
			t.Error("New session marker should still exist")
		}
	})
}

func TestHandleSessionStart(t *testing.T) {
	// Set up test environment
	tmpDir, err := os.MkdirTemp("", "ccpersona-session-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create a test project directory
	projectDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Change to project directory
	originalWd, _ := os.Getwd()
	_ = os.Chdir(projectDir)
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	// Override home directory
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() {
		_ = os.Setenv("HOME", originalHome)
	}()

	// Set unique session ID
	_ = os.Setenv("CLAUDE_SESSION_ID", "test-handle-start")
	defer func() {
		_ = os.Unsetenv("CLAUDE_SESSION_ID")
	}()

	t.Run("NoPersonaConfig", func(t *testing.T) {
		// Should complete without error even without config
		if err := HandleSessionStart(); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("WithPersonaConfig", func(t *testing.T) {
		// Create personas directory and test persona
		personasDir := filepath.Join(tmpDir, ".claude", "personas")
		if err := os.MkdirAll(personasDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Create test persona
		testPersona := filepath.Join(personasDir, "test.md")
		if err := os.WriteFile(testPersona, []byte("# Test Persona"), 0644); err != nil {
			t.Fatal(err)
		}

		// Create project config
		claudeDir := filepath.Join(projectDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatal(err)
		}

		config := &Config{Name: "test"}
		if err := SaveConfig(projectDir, config); err != nil {
			t.Fatal(err)
		}

		// Reset session marker
		_ = os.Setenv("CLAUDE_SESSION_ID", "test-with-config")

		// Handle session start
		if err := HandleSessionStart(); err != nil {
			t.Errorf("Failed to handle session start: %v", err)
		}

		// Verify CLAUDE.md was created
		claudeMd := filepath.Join(tmpDir, ".claude", "CLAUDE.md")
		if _, err := os.Stat(claudeMd); os.IsNotExist(err) {
			t.Error("CLAUDE.md should have been created")
		}
	})
}

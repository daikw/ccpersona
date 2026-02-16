package voice

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const dedupDir = "ccpersona-voice"

// DedupTracker tracks previously synthesized messages to avoid duplicates.
// State is stored per session in the OS temp directory.
type DedupTracker struct {
	sessionID string
	dir       string
}

// NewDedupTracker creates a tracker for the given session.
func NewDedupTracker(sessionID string) *DedupTracker {
	return &DedupTracker{
		sessionID: sessionID,
		dir:       filepath.Join(os.TempDir(), dedupDir),
	}
}

// IsDuplicate returns true if this text was already synthesized in the current session.
func (dt *DedupTracker) IsDuplicate(text string) bool {
	hash := hashText(text)
	stored, err := os.ReadFile(dt.markerPath())
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(stored)) == hash
}

// Record stores the hash of the synthesized text for this session.
func (dt *DedupTracker) Record(text string) {
	if err := os.MkdirAll(dt.dir, 0755); err != nil {
		log.Debug().Err(err).Msg("Failed to create dedup directory")
		return
	}
	hash := hashText(text)
	if err := os.WriteFile(dt.markerPath(), []byte(hash), 0644); err != nil {
		log.Debug().Err(err).Msg("Failed to write dedup marker")
	}
}

// Cleanup removes markers older than 24 hours.
func (dt *DedupTracker) Cleanup() {
	entries, err := os.ReadDir(dt.dir)
	if err != nil {
		return
	}

	cutoff := time.Now().Add(-24 * time.Hour)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dt.dir, entry.Name()))
		}
	}
}

func (dt *DedupTracker) markerPath() string {
	return filepath.Join(dt.dir, dt.sessionID+".lastread")
}

func hashText(text string) string {
	h := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", h)
}

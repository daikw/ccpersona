package voice

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MuteStatus represents the global mute state snapshot.
type MuteStatus struct {
	MutedAt time.Time `json:"muted_at"`
	Reason  string    `json:"reason,omitempty"`
}

// MutePath returns the absolute path to the mute marker file
// (~/.claude/ccpersona/mute). The marker's existence means voice
// synthesis is globally muted.
func MutePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".claude", "ccpersona", "mute"), nil
}

// IsMuted reports whether global voice synthesis is currently muted.
// Errors (e.g. missing HOME) are treated as "not muted" so the gate
// fails open and does not break hook-driven callers.
func IsMuted() bool {
	path, err := MutePath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// Mute enables the global mute. Idempotent: refreshes MutedAt and Reason.
func Mute(reason string) (*MuteStatus, error) {
	path, err := MutePath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create mute dir: %w", err)
	}
	status := &MuteStatus{MutedAt: time.Now().UTC(), Reason: reason}
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal mute status: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, fmt.Errorf("write mute file: %w", err)
	}
	return status, nil
}

// Unmute removes the mute marker. Idempotent when already unmuted.
func Unmute() error {
	path, err := MutePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove mute file: %w", err)
	}
	return nil
}

// LoadMuteStatus returns the current mute status. Returns (nil, nil)
// when not muted. When the marker file exists but is unparseable,
// returns a zero-value status so callers still treat the session as muted.
func LoadMuteStatus() (*MuteStatus, error) {
	path, err := MutePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read mute file: %w", err)
	}
	var status MuteStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return &MuteStatus{}, nil
	}
	return &status, nil
}

package voice

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMute_NotMutedByDefault(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if IsMuted() {
		t.Fatal("expected IsMuted() == false on a fresh home")
	}
}

func TestMute_SetsMarkerFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	status, err := Mute("focusing")
	if err != nil {
		t.Fatalf("Mute returned error: %v", err)
	}
	if status == nil {
		t.Fatal("Mute returned nil status")
	}
	if status.Reason != "focusing" {
		t.Errorf("Reason = %q, want %q", status.Reason, "focusing")
	}
	if status.MutedAt.IsZero() {
		t.Error("MutedAt should be set")
	}

	if !IsMuted() {
		t.Error("IsMuted() should return true after Mute()")
	}

	want := filepath.Join(home, ".claude", "ccpersona", "mute")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("expected mute file at %s: %v", want, err)
	}
}

func TestMute_Idempotent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	first, err := Mute("a")
	if err != nil {
		t.Fatalf("first Mute: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	second, err := Mute("b")
	if err != nil {
		t.Fatalf("second Mute: %v", err)
	}

	if !second.MutedAt.After(first.MutedAt) {
		t.Error("second Mute should refresh MutedAt to a later time")
	}
	if second.Reason != "b" {
		t.Errorf("second Reason = %q, want %q", second.Reason, "b")
	}
}

func TestUnmute_RemovesMarker(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if _, err := Mute(""); err != nil {
		t.Fatalf("Mute: %v", err)
	}
	if !IsMuted() {
		t.Fatal("precondition: should be muted")
	}

	if err := Unmute(); err != nil {
		t.Fatalf("Unmute: %v", err)
	}
	if IsMuted() {
		t.Error("IsMuted() should return false after Unmute()")
	}
}

func TestUnmute_IdempotentWhenNotMuted(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := Unmute(); err != nil {
		t.Errorf("Unmute on non-muted state should succeed, got: %v", err)
	}
}

func TestLoadMuteStatus_ReturnsNilWhenNotMuted(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	status, err := LoadMuteStatus()
	if err != nil {
		t.Fatalf("LoadMuteStatus: %v", err)
	}
	if status != nil {
		t.Errorf("expected nil status when not muted, got %+v", status)
	}
}

func TestLoadMuteStatus_ReturnsStatusWhenMuted(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if _, err := Mute("quiet hours"); err != nil {
		t.Fatalf("Mute: %v", err)
	}

	status, err := LoadMuteStatus()
	if err != nil {
		t.Fatalf("LoadMuteStatus: %v", err)
	}
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.Reason != "quiet hours" {
		t.Errorf("Reason = %q, want %q", status.Reason, "quiet hours")
	}
	if status.MutedAt.IsZero() {
		t.Error("MutedAt should be populated")
	}
}

func TestLoadMuteStatus_HandlesCorruptFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".claude", "ccpersona", "mute")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not-json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !IsMuted() {
		t.Error("IsMuted() should rely on file existence, not content")
	}

	status, err := LoadMuteStatus()
	if err != nil {
		t.Fatalf("LoadMuteStatus on corrupt file should not error, got: %v", err)
	}
	if status == nil {
		t.Error("LoadMuteStatus on corrupt file should return non-nil zero status (still muted)")
	}
}

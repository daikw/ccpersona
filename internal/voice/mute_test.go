package voice

import (
	"os"
	"path/filepath"
	"testing"
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

	changed, err := Mute()
	if err != nil {
		t.Fatalf("Mute returned error: %v", err)
	}
	if !changed {
		t.Fatal("Mute should report a state change")
	}

	if !IsMuted() {
		t.Error("IsMuted() should return true after Mute()")
	}

	want := filepath.Join(home, ".agents", "ccpersona", "mute")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("expected mute file at %s: %v", want, err)
	}
}

func TestMute_ReadsLegacyMarker(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	legacyPath := filepath.Join(home, ".claude", "ccpersona", "mute")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, []byte(`{"muted_at":"2026-07-10T00:00:00Z","reason":"legacy"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if !IsMuted() {
		t.Error("IsMuted() should return true for a legacy mute marker")
	}

	status, err := LoadMuteStatus()
	if err != nil {
		t.Fatalf("LoadMuteStatus: %v", err)
	}
	if status == nil || status.MutedAt.IsZero() {
		t.Fatalf("expected legacy marker status, got %+v", status)
	}
}

func TestMute_Idempotent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	changed, err := Mute()
	if err != nil {
		t.Fatalf("first Mute: %v", err)
	}
	if !changed {
		t.Fatal("first Mute should report a state change")
	}
	first, err := LoadMuteStatus()
	if err != nil {
		t.Fatalf("LoadMuteStatus after first Mute: %v", err)
	}

	changed, err = Mute()
	if err != nil {
		t.Fatalf("second Mute: %v", err)
	}
	if changed {
		t.Fatal("second Mute should not report a state change")
	}
	second, err := LoadMuteStatus()
	if err != nil {
		t.Fatalf("LoadMuteStatus after second Mute: %v", err)
	}
	if second == nil || first == nil || !second.MutedAt.Equal(first.MutedAt) {
		t.Error("second Mute should preserve the original mute timestamp")
	}
}

func TestUnmute_RemovesMarker(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if _, err := Mute(); err != nil {
		t.Fatalf("Mute: %v", err)
	}
	legacyPath := filepath.Join(home, ".claude", "ccpersona", "mute")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, []byte(`{"reason":"legacy"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsMuted() {
		t.Fatal("precondition: should be muted")
	}

	changed, err := Unmute()
	if err != nil {
		t.Fatalf("Unmute: %v", err)
	}
	if !changed {
		t.Fatal("Unmute should report a state change")
	}
	if IsMuted() {
		t.Error("IsMuted() should return false after Unmute()")
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Errorf("legacy marker should be removed, got %v", err)
	}
}

func TestUnmute_IdempotentWhenNotMuted(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	changed, err := Unmute()
	if err != nil {
		t.Errorf("Unmute on non-muted state should succeed, got: %v", err)
	}
	if changed {
		t.Error("Unmute on non-muted state should not report a state change")
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

	if _, err := Mute(); err != nil {
		t.Fatalf("Mute: %v", err)
	}

	status, err := LoadMuteStatus()
	if err != nil {
		t.Fatalf("LoadMuteStatus: %v", err)
	}
	if status == nil {
		t.Fatal("expected non-nil status")
		return
	}
	if status.MutedAt.IsZero() {
		t.Error("MutedAt should be populated")
	}
}

func TestLoadMuteStatus_HandlesCorruptFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".agents", "ccpersona", "mute")
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

package voice

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDedupTracker_IsDuplicate(t *testing.T) {
	dt := NewDedupTracker("test-session-dedup")
	dt.dir = t.TempDir()

	text := "こんにちはなのだ"

	if dt.IsDuplicate(text) {
		t.Error("Should not be duplicate on first call")
	}

	dt.Record(text)

	if !dt.IsDuplicate(text) {
		t.Error("Should be duplicate after recording")
	}

	if dt.IsDuplicate("別のテキスト") {
		t.Error("Different text should not be duplicate")
	}
}

func TestDedupTracker_DifferentSessions(t *testing.T) {
	dir := t.TempDir()

	dt1 := NewDedupTracker("session-1")
	dt1.dir = dir
	dt2 := NewDedupTracker("session-2")
	dt2.dir = dir

	text := "同じテキスト"
	dt1.Record(text)

	if !dt1.IsDuplicate(text) {
		t.Error("Should be duplicate in session-1")
	}
	if dt2.IsDuplicate(text) {
		t.Error("Should not be duplicate in session-2")
	}
}

func TestDedupTracker_OverwritesPrevious(t *testing.T) {
	dt := NewDedupTracker("test-overwrite")
	dt.dir = t.TempDir()

	dt.Record("first message")
	dt.Record("second message")

	if dt.IsDuplicate("first message") {
		t.Error("First message should no longer be tracked after overwrite")
	}
	if !dt.IsDuplicate("second message") {
		t.Error("Second message should be tracked")
	}
}

func TestDedupTracker_Cleanup(t *testing.T) {
	dir := t.TempDir()
	dt := NewDedupTracker("current")
	dt.dir = dir

	// Create an old marker
	oldPath := filepath.Join(dir, "old-session.lastread")
	if err := os.WriteFile(oldPath, []byte("hash"), 0644); err != nil {
		t.Fatal(err)
	}
	// Set mod time to 25 hours ago
	oldTime := time.Now().Add(-25 * time.Hour)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Create a current marker
	dt.Record("current text")

	dt.Cleanup()

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("Old marker should have been cleaned up")
	}
	if !dt.IsDuplicate("current text") {
		t.Error("Current marker should still exist")
	}
}

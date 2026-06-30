package voice

import (
	"os"
	"path/filepath"
	"strings"
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

func TestDedupTracker_PathTraversal(t *testing.T) {
	// dir is the only place markers may be written. The parent holds a sentinel
	// file that a traversal-style sessionID must never overwrite.
	root := t.TempDir()
	dir := filepath.Join(root, "markers")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	maliciousIDs := []string{
		"../../../etc/x",
		"../escape",
		"a/b/c",
		"with space",
		"..",
		"foo\x00bar",
	}

	for _, id := range maliciousIDs {
		t.Run(id, func(t *testing.T) {
			dt := NewDedupTracker(id)
			dt.dir = dir

			dt.Record("payload")

			// The marker must land directly inside dir, not anywhere else.
			abs, err := filepath.Abs(dt.markerPath())
			if err != nil {
				t.Fatal(err)
			}
			if filepath.Dir(abs) != dir {
				t.Errorf("marker path %q escaped dedup dir %q", abs, dir)
			}

			// Round-trip must still work for the normalized name.
			if !dt.IsDuplicate("payload") {
				t.Error("recorded payload should be reported as duplicate")
			}
		})
	}

	// Nothing should have been written outside dir.
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "markers" {
			t.Errorf("unexpected entry written outside dedup dir: %s", e.Name())
		}
	}
}

func TestDedupTracker_LongSessionID(t *testing.T) {
	// Safe charset but over maxSessionIDLen: must fall back to the hashed name
	// to avoid ENAMETOOLONG, and still round-trip.
	longID := strings.Repeat("a", maxSessionIDLen+72)
	dt := NewDedupTracker(longID)
	dt.dir = t.TempDir()

	dt.Record("payload")
	if !dt.IsDuplicate("payload") {
		t.Error("recorded payload should be reported as duplicate")
	}

	name := filepath.Base(dt.markerPath())
	if strings.Contains(name, longID[:maxSessionIDLen+1]) {
		t.Errorf("marker name embeds over-long sessionID verbatim: %s", name)
	}
	if len(name) > maxSessionIDLen+len(".lastread") {
		t.Errorf("marker name too long (%d chars): %s", len(name), name)
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

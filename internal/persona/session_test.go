package persona

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleSessionStart(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ccpersona-session-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	projectDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	originalWd, _ := os.Getwd()
	_ = os.Chdir(projectDir)
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() {
		_ = os.Setenv("HOME", originalHome)
	}()

	t.Run("NoPersonaConfig", func(t *testing.T) {
		if err := HandleSessionStart(); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("WithPersonaConfig", func(t *testing.T) {
		personasDir := filepath.Join(tmpDir, ".claude", "personas")
		if err := os.MkdirAll(personasDir, 0755); err != nil {
			t.Fatal(err)
		}

		testPersonaContent := "# 人格: test\n\n## 口調\nテストなのだ。"
		if err := os.WriteFile(filepath.Join(personasDir, "test.md"), []byte(testPersonaContent), 0644); err != nil {
			t.Fatal(err)
		}

		if err := SaveConfig(projectDir, &Config{Name: "test"}); err != nil {
			t.Fatal(err)
		}

		origStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := HandleSessionStart()

		_ = w.Close()
		os.Stdout = origStdout

		buf := make([]byte, 4096)
		n, _ := r.Read(buf)
		output := string(buf[:n])

		if err != nil {
			t.Errorf("Failed to handle session start: %v", err)
		}
		if !strings.Contains(output, "# 人格: test") {
			t.Errorf("Expected persona content in stdout, got: %q", output)
		}

		// 2回目も同じ出力が得られる（重複防止なし）
		r2, w2, _ := os.Pipe()
		os.Stdout = w2

		err = HandleSessionStart()

		_ = w2.Close()
		os.Stdout = origStdout

		buf2 := make([]byte, 4096)
		n2, _ := r2.Read(buf2)
		output2 := string(buf2[:n2])

		if err != nil {
			t.Errorf("Failed on second call: %v", err)
		}
		if output != output2 {
			t.Errorf("Expected same output on second call, got: %q", output2)
		}
	})

	t.Run("WithVoiceConfig", func(t *testing.T) {
		personasDir := filepath.Join(tmpDir, ".claude", "personas")
		if err := os.MkdirAll(personasDir, 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(filepath.Join(personasDir, "voice-test.md"), []byte("# 人格: voice-test"), 0644); err != nil {
			t.Fatal(err)
		}

		config := &Config{
			Name:  "voice-test",
			Voice: &VoiceConfig{Provider: "aivisspeech", Speaker: 888753760},
		}
		if err := SaveConfig(projectDir, config); err != nil {
			t.Fatal(err)
		}

		origStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := HandleSessionStart()

		_ = w.Close()
		os.Stdout = origStdout

		buf := make([]byte, 4096)
		n, _ := r.Read(buf)
		output := string(buf[:n])

		if err != nil {
			t.Errorf("Failed to handle session start: %v", err)
		}
		if !strings.Contains(output, "speak MCP ツール") {
			t.Errorf("Expected speak instruction in stdout, got: %q", output)
		}
	})
}

package persona

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewManager(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	if manager == nil {
		t.Fatal("Expected manager, got nil")
	}

	if manager.homeDir == "" { //nolint:staticcheck // checked for nil above
		t.Error("Home directory not set")
	}

	if !strings.HasSuffix(manager.personasDir, filepath.Join(".claude", "personas")) { //nolint:staticcheck // checked for nil above
		t.Errorf("Invalid personas directory: %s", manager.personasDir)
	}
}

func TestListPersonas(t *testing.T) {
	// Create test manager with custom paths
	tmpDir, err := os.MkdirTemp("", "ccpersona-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	manager := &Manager{
		homeDir:     tmpDir,
		personasDir: filepath.Join(tmpDir, ".claude", "personas"),
	}

	// Test 1: Empty directory
	t.Run("EmptyDirectory", func(t *testing.T) {
		personas, err := manager.ListPersonas()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(personas) != 0 {
			t.Errorf("Expected empty list, got %v", personas)
		}
	})

	// Test 2: With personas
	t.Run("WithPersonas", func(t *testing.T) {
		// Create personas directory
		if err := os.MkdirAll(manager.personasDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Create test persona files
		testPersonas := []string{"default", "zundamon", "engineer"}
		for _, name := range testPersonas {
			path := filepath.Join(manager.personasDir, name+".md")
			if err := os.WriteFile(path, []byte("# Test persona"), 0644); err != nil {
				t.Fatal(err)
			}
		}

		// Also create a non-.md file that should be ignored
		if err := os.WriteFile(filepath.Join(manager.personasDir, "ignore.txt"), []byte("ignore"), 0644); err != nil {
			t.Fatal(err)
		}

		// List personas
		personas, err := manager.ListPersonas()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(personas) != len(testPersonas) {
			t.Errorf("Expected %d personas, got %d", len(testPersonas), len(personas))
		}

		// Check all expected personas are present
		for _, expected := range testPersonas {
			found := false
			for _, actual := range personas {
				if actual == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected persona %s not found", expected)
			}
		}
	})
}

func TestPersonaExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ccpersona-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	manager := &Manager{
		homeDir:     tmpDir,
		personasDir: filepath.Join(tmpDir, ".claude", "personas"),
	}

	// Create personas directory
	if err := os.MkdirAll(manager.personasDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a test persona
	testPath := filepath.Join(manager.personasDir, "test.md")
	if err := os.WriteFile(testPath, []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test existing persona
	if !manager.PersonaExists("test") {
		t.Error("Expected persona 'test' to exist")
	}

	// Test non-existing persona
	if manager.PersonaExists("nonexistent") {
		t.Error("Expected persona 'nonexistent' to not exist")
	}
}

func TestCreatePersona(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ccpersona-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	manager := &Manager{
		homeDir:     tmpDir,
		personasDir: filepath.Join(tmpDir, ".claude", "personas"),
	}

	// Test creating new persona
	t.Run("CreateNew", func(t *testing.T) {
		err := manager.CreatePersona("newpersona")
		if err != nil {
			t.Errorf("Failed to create persona: %v", err)
		}

		// Verify file was created
		if !manager.PersonaExists("newpersona") {
			t.Error("Persona file was not created")
		}

		// Verify content
		content, err := manager.ReadPersona("newpersona")
		if err != nil {
			t.Errorf("Failed to read created persona: %v", err)
		}

		if !strings.Contains(content, "# 人格: newpersona") {
			t.Error("Persona content does not contain expected header")
		}
	})

	// Test creating existing persona
	t.Run("CreateExisting", func(t *testing.T) {
		err := manager.CreatePersona("newpersona")
		if err == nil {
			t.Error("Expected error when creating existing persona")
		}
	})
}

func TestReadPersona(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ccpersona-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	manager := &Manager{
		homeDir:     tmpDir,
		personasDir: filepath.Join(tmpDir, ".claude", "personas"),
	}

	// Create personas directory
	if err := os.MkdirAll(manager.personasDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test persona
	testContent := "# Test Persona\n\nThis is a test persona."
	testPath := filepath.Join(manager.personasDir, "test.md")
	if err := os.WriteFile(testPath, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test reading existing persona
	t.Run("ReadExisting", func(t *testing.T) {
		content, err := manager.ReadPersona("test")
		if err != nil {
			t.Errorf("Failed to read persona: %v", err)
		}
		if content != testContent {
			t.Errorf("Content mismatch: expected %q, got %q", testContent, content)
		}
	})

	// Test reading non-existing persona
	t.Run("ReadNonExisting", func(t *testing.T) {
		_, err := manager.ReadPersona("nonexistent")
		if err == nil {
			t.Error("Expected error when reading non-existing persona")
		}
	})
}

func TestStripYAMLFrontMatter(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "front matter あり",
			input: "---\nname: zundamon\ndescription: test\n---\n# 人格\n本文",
			want:  "# 人格\n本文",
		},
		{
			name:  "front matter なし",
			input: "# 人格\n本文",
			want:  "# 人格\n本文",
		},
		{
			name:  "閉じ区切りなし（通常Markdownとして扱う）",
			input: "---\nname: zundamon\n# 人格\n本文",
			want:  "---\nname: zundamon\n# 人格\n本文",
		},
		{
			name:  "... で閉じる場合",
			input: "---\nname: test\n...\n# 人格\n本文",
			want:  "# 人格\n本文",
		},
		{
			name:  "空文字列",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripYAMLFrontMatter(tt.input)
			if got != tt.want {
				t.Errorf("stripYAMLFrontMatter() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadPersonaForContext(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ccpersona-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	manager := &Manager{
		homeDir:     tmpDir,
		personasDir: filepath.Join(tmpDir, ".claude", "personas"),
	}
	if err := os.MkdirAll(manager.personasDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Run("front matter が除去される", func(t *testing.T) {
		content := "---\nname: zundamon\ndescription: test\n---\n# 人格: zundamon\n本文"
		path := filepath.Join(manager.personasDir, "zundamon.md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		got, err := manager.ReadPersonaForContext("zundamon")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(got, "name: zundamon") {
			t.Errorf("front matter が残っている: %q", got)
		}
		if !strings.Contains(got, "# 人格: zundamon") {
			t.Errorf("本文が含まれていない: %q", got)
		}
	})

	t.Run("front matter なしのファイルはそのまま返す", func(t *testing.T) {
		content := "# 人格: anneli\n本文"
		path := filepath.Join(manager.personasDir, "anneli.md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		got, err := manager.ReadPersonaForContext("anneli")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != content {
			t.Errorf("content mismatch: got %q, want %q", got, content)
		}
	})
}

func TestGetCurrentPersona(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ccpersona-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Override home directory
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() {
		_ = os.Setenv("HOME", originalHome)
	}()

	manager := &Manager{
		homeDir:     tmpDir,
		personasDir: filepath.Join(tmpDir, ".claude", "personas"),
	}

	// Create .claude directory
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Test 1: No persona.json
	t.Run("NoConfig", func(t *testing.T) {
		// Change to a temp dir with no persona.json
		origWd, _ := os.Getwd()
		tmpProject, _ := os.MkdirTemp("", "ccpersona-proj-*")
		defer func() {
			_ = os.Chdir(origWd)
			_ = os.RemoveAll(tmpProject)
		}()
		_ = os.Chdir(tmpProject)

		current, err := manager.GetCurrentPersona()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if current != "none" {
			t.Errorf("Expected 'none', got %s", current)
		}
	})

	// Test 2: With persona.json
	t.Run("WithConfig", func(t *testing.T) {
		origWd, _ := os.Getwd()
		tmpProject, _ := os.MkdirTemp("", "ccpersona-proj-*")
		defer func() {
			_ = os.Chdir(origWd)
			_ = os.RemoveAll(tmpProject)
		}()
		_ = os.Chdir(tmpProject)

		config := &Config{Name: "ずんだもん"}
		if err := SaveConfig(tmpProject, config); err != nil {
			t.Fatal(err)
		}

		current, err := manager.GetCurrentPersona()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if current != "ずんだもん" {
			t.Errorf("Expected 'ずんだもん', got %s", current)
		}
	})
}

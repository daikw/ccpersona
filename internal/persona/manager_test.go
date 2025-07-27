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
	
	if !strings.HasSuffix(manager.claudeMdPath, filepath.Join(".claude", "CLAUDE.md")) {
		t.Errorf("Invalid CLAUDE.md path: %s", manager.claudeMdPath)
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
		homeDir:      tmpDir,
		personasDir:  filepath.Join(tmpDir, ".claude", "personas"),
		claudeMdPath: filepath.Join(tmpDir, ".claude", "CLAUDE.md"),
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
		homeDir:      tmpDir,
		personasDir:  filepath.Join(tmpDir, ".claude", "personas"),
		claudeMdPath: filepath.Join(tmpDir, ".claude", "CLAUDE.md"),
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
		homeDir:      tmpDir,
		personasDir:  filepath.Join(tmpDir, ".claude", "personas"),
		claudeMdPath: filepath.Join(tmpDir, ".claude", "CLAUDE.md"),
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
		homeDir:      tmpDir,
		personasDir:  filepath.Join(tmpDir, ".claude", "personas"),
		claudeMdPath: filepath.Join(tmpDir, ".claude", "CLAUDE.md"),
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

func TestApplyPersona(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ccpersona-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()
	
	manager := &Manager{
		homeDir:      tmpDir,
		personasDir:  filepath.Join(tmpDir, ".claude", "personas"),
		claudeMdPath: filepath.Join(tmpDir, ".claude", "CLAUDE.md"),
	}
	
	// Create directories
	if err := os.MkdirAll(manager.personasDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(manager.claudeMdPath), 0755); err != nil {
		t.Fatal(err)
	}
	
	// Create test persona
	testContent := "# Test Persona\n\nThis is a test persona."
	testPath := filepath.Join(manager.personasDir, "test.md")
	if err := os.WriteFile(testPath, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}
	
	// Test applying existing persona
	t.Run("ApplyExisting", func(t *testing.T) {
		err := manager.ApplyPersona("test")
		if err != nil {
			t.Errorf("Failed to apply persona: %v", err)
		}
		
		// Verify CLAUDE.md was created with correct content
		content, err := os.ReadFile(manager.claudeMdPath)
		if err != nil {
			t.Errorf("Failed to read CLAUDE.md: %v", err)
		}
		
		if string(content) != testContent {
			t.Errorf("CLAUDE.md content mismatch: expected %q, got %q", testContent, string(content))
		}
	})
	
	// Test applying non-existing persona
	t.Run("ApplyNonExisting", func(t *testing.T) {
		err := manager.ApplyPersona("nonexistent")
		if err == nil {
			t.Error("Expected error when applying non-existing persona")
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
	
	manager := &Manager{
		homeDir:      tmpDir,
		personasDir:  filepath.Join(tmpDir, ".claude", "personas"),
		claudeMdPath: filepath.Join(tmpDir, ".claude", "CLAUDE.md"),
	}
	
	// Create .claude directory
	if err := os.MkdirAll(filepath.Dir(manager.claudeMdPath), 0755); err != nil {
		t.Fatal(err)
	}
	
	// Test 1: No CLAUDE.md file
	t.Run("NoCLAUDEmd", func(t *testing.T) {
		current, err := manager.GetCurrentPersona()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if current != "none" {
			t.Errorf("Expected 'none', got %s", current)
		}
	})
	
	// Test 2: CLAUDE.md with persona name
	t.Run("WithPersonaName", func(t *testing.T) {
		content := "# 人格: ずんだもん\n\n## 口調\nなのだ！"
		if err := os.WriteFile(manager.claudeMdPath, []byte(content), 0644); err != nil {
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
	
	// Test 3: CLAUDE.md without persona name
	t.Run("WithoutPersonaName", func(t *testing.T) {
		content := "# Some other content\n\n## Not a persona"
		if err := os.WriteFile(manager.claudeMdPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		
		current, err := manager.GetCurrentPersona()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if current != "unknown" {
			t.Errorf("Expected 'unknown', got %s", current)
		}
	})
}
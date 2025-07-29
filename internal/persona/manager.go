package persona

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// Manager handles persona operations
type Manager struct {
	homeDir      string
	personasDir  string
	claudeMdPath string
}

// NewManager creates a new persona manager
func NewManager() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	return &Manager{
		homeDir:      homeDir,
		personasDir:  filepath.Join(homeDir, ".claude", "personas"),
		claudeMdPath: filepath.Join(homeDir, ".claude", "CLAUDE.md"),
	}, nil
}

// ListPersonas returns all available personas
func (m *Manager) ListPersonas() ([]string, error) {
	entries, err := os.ReadDir(m.personasDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debug().Msg("Personas directory does not exist")
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read personas directory: %w", err)
	}

	var personas []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			name := strings.TrimSuffix(entry.Name(), ".md")
			personas = append(personas, name)
		}
	}

	return personas, nil
}

// GetPersonaPath returns the full path to a persona file
func (m *Manager) GetPersonaPath(name string) string {
	return filepath.Join(m.personasDir, name+".md")
}

// PersonaExists checks if a persona exists
func (m *Manager) PersonaExists(name string) bool {
	path := m.GetPersonaPath(name)
	_, err := os.Stat(path)
	return err == nil
}

// ApplyPersona copies the specified persona to CLAUDE.md
func (m *Manager) ApplyPersona(name string) error {
	if !m.PersonaExists(name) {
		return fmt.Errorf("persona '%s' does not exist", name)
	}

	sourcePath := m.GetPersonaPath(name)

	// Read persona file
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read persona file: %w", err)
	}

	// Write to CLAUDE.md
	if err := os.WriteFile(m.claudeMdPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write CLAUDE.md: %w", err)
	}

	log.Info().Str("persona", name).Msg("Applied persona")
	return nil
}

// CreatePersona creates a new persona from a template
func (m *Manager) CreatePersona(name string) error {
	if m.PersonaExists(name) {
		return fmt.Errorf("persona '%s' already exists", name)
	}

	// Ensure personas directory exists
	if err := os.MkdirAll(m.personasDir, 0755); err != nil {
		return fmt.Errorf("failed to create personas directory: %w", err)
	}

	template := fmt.Sprintf(`# 人格: %s

## 口調
標準的な口調で話します。

## 考え方
- 論理的に問題を解決します
- 効率性を重視します

## 価値観
- コードの品質を大切にします
- テストの重要性を理解しています

## 専門性
- 一般的なプログラミング知識

## 対話スタイル
- 明確で簡潔な説明
- 必要に応じて詳細を提供
`, name)

	path := m.GetPersonaPath(name)
	if err := os.WriteFile(path, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to create persona file: %w", err)
	}

	log.Info().Str("persona", name).Str("path", path).Msg("Created new persona")
	return nil
}

// ReadPersona reads and returns the content of a persona file
func (m *Manager) ReadPersona(name string) (string, error) {
	if !m.PersonaExists(name) {
		return "", fmt.Errorf("persona '%s' does not exist", name)
	}

	path := m.GetPersonaPath(name)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read persona file: %w", err)
	}

	return string(content), nil
}

// GetCurrentPersona tries to determine the currently active persona
func (m *Manager) GetCurrentPersona() (string, error) {
	// Read CLAUDE.md
	content, err := os.ReadFile(m.claudeMdPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "none", nil
		}
		return "", fmt.Errorf("failed to read CLAUDE.md: %w", err)
	}

	// Try to extract persona name from the content
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# 人格:") || strings.HasPrefix(line, "# Persona:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				name := strings.TrimSpace(parts[1])
				return name, nil
			}
		}
	}

	return "unknown", nil
}

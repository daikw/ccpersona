package hook

import (
	"strings"
	"testing"
)

func TestDetectAndParseCodexEvent(t *testing.T) {
	jsonData := `{
		"type": "agent-turn-complete",
		"thread-id": "12345678-1234-1234-1234-123456789abc",
		"turn-id": 5,
		"cwd": "/home/user/project",
		"input-messages": ["Fix the bug in main.go"],
		"last-assistant-message": "I've fixed the bug by updating the error handling."
	}`

	reader := strings.NewReader(jsonData)
	event, err := DetectAndParse(reader)
	if err != nil {
		t.Fatalf("Failed to parse Codex event: %v", err)
	}

	if event.Source != "codex" {
		t.Errorf("Expected source 'codex', got '%s'", event.Source)
	}

	if event.SessionID != "12345678-1234-1234-1234-123456789abc" {
		t.Errorf("Expected session ID '12345678-1234-1234-1234-123456789abc', got '%s'", event.SessionID)
	}

	if event.EventType != "agent-turn-complete" {
		t.Errorf("Expected event type 'agent-turn-complete', got '%s'", event.EventType)
	}

	if len(event.UserInput) != 1 || event.UserInput[0] != "Fix the bug in main.go" {
		t.Errorf("Unexpected user input: %v", event.UserInput)
	}

	if event.AIResponse != "I've fixed the bug by updating the error handling." {
		t.Errorf("Unexpected AI response: %s", event.AIResponse)
	}

	if !event.IsCodex() {
		t.Error("Expected IsCodex() to return true")
	}

	if event.IsClaudeCode() {
		t.Error("Expected IsClaudeCode() to return false")
	}

	codexEvent, ok := event.GetCodexEvent()
	if !ok {
		t.Error("Expected to get Codex event")
	}

	if codexEvent.TurnID != 5 {
		t.Errorf("Expected turn ID 5, got %d", codexEvent.TurnID)
	}
}

func TestDetectAndParseClaudeCodeUserPromptSubmit(t *testing.T) {
	jsonData := `{
		"session_id": "test-session-123",
		"transcript_path": "/path/to/transcript.jsonl",
		"cwd": "/home/user/project",
		"hook_event_name": "UserPromptSubmit",
		"prompt": "Help me fix this bug"
	}`

	reader := strings.NewReader(jsonData)
	event, err := DetectAndParse(reader)
	if err != nil {
		t.Fatalf("Failed to parse Claude Code event: %v", err)
	}

	if event.Source != "claude-code" {
		t.Errorf("Expected source 'claude-code', got '%s'", event.Source)
	}

	if event.SessionID != "test-session-123" {
		t.Errorf("Expected session ID 'test-session-123', got '%s'", event.SessionID)
	}

	if event.EventType != "UserPromptSubmit" {
		t.Errorf("Expected event type 'UserPromptSubmit', got '%s'", event.EventType)
	}

	if len(event.UserInput) != 1 || event.UserInput[0] != "Help me fix this bug" {
		t.Errorf("Unexpected user input: %v", event.UserInput)
	}

	if !event.IsClaudeCode() {
		t.Error("Expected IsClaudeCode() to return true")
	}

	if event.IsCodex() {
		t.Error("Expected IsCodex() to return false")
	}
}

func TestDetectAndParseClaudeCodeNotification(t *testing.T) {
	jsonData := `{
		"session_id": "test-session-456",
		"transcript_path": "/path/to/transcript.jsonl",
		"cwd": "/home/user/project",
		"hook_event_name": "Notification",
		"message": "Task completed successfully"
	}`

	reader := strings.NewReader(jsonData)
	event, err := DetectAndParse(reader)
	if err != nil {
		t.Fatalf("Failed to parse Claude Code notification: %v", err)
	}

	if event.Source != "claude-code" {
		t.Errorf("Expected source 'claude-code', got '%s'", event.Source)
	}

	if event.EventType != "Notification" {
		t.Errorf("Expected event type 'Notification', got '%s'", event.EventType)
	}

	if event.AIResponse != "Task completed successfully" {
		t.Errorf("Expected AI response 'Task completed successfully', got '%s'", event.AIResponse)
	}
}

func TestDetectAndParseInvalidJSON(t *testing.T) {
	jsonData := `{"invalid json`

	reader := strings.NewReader(jsonData)
	_, err := DetectAndParse(reader)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestDetectAndParseUnknownFormat(t *testing.T) {
	jsonData := `{"unknown_field": "value"}`

	reader := strings.NewReader(jsonData)
	_, err := DetectAndParse(reader)
	if err == nil {
		t.Error("Expected error for unknown format, got nil")
	}
}

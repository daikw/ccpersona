package hook

import (
	"strings"
	"testing"
)

func TestDetectAndParseCodexEvent(t *testing.T) {
	jsonData := `{
		"type": "agent-turn-complete",
		"thread-id": "12345678-1234-1234-1234-123456789abc",
		"turn-id": "5",
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

	if codexEvent.TurnID != "5" {
		t.Errorf("Expected turn ID '5', got '%s'", codexEvent.TurnID)
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

func TestDetectAndParseClaudeCodeSessionStart(t *testing.T) {
	jsonData := `{
		"session_id": "session-start-123",
		"transcript_path": "/path/to/transcript.jsonl",
		"cwd": "/home/user/project",
		"hook_event_name": "SessionStart"
	}`

	reader := strings.NewReader(jsonData)
	event, err := DetectAndParse(reader)
	if err != nil {
		t.Fatalf("Failed to parse SessionStart event: %v", err)
	}

	if event.Source != "claude-code" {
		t.Errorf("Expected source 'claude-code', got '%s'", event.Source)
	}

	if event.EventType != "SessionStart" {
		t.Errorf("Expected event type 'SessionStart', got '%s'", event.EventType)
	}

	if event.SessionID != "session-start-123" {
		t.Errorf("Expected session ID 'session-start-123', got '%s'", event.SessionID)
	}

	if event.CWD != "/home/user/project" {
		t.Errorf("Expected CWD '/home/user/project', got '%s'", event.CWD)
	}
}

func TestDetectAndParseClaudeCodeSessionEnd(t *testing.T) {
	jsonData := `{
		"session_id": "session-end-456",
		"transcript_path": "/path/to/transcript.jsonl",
		"cwd": "/home/user/project",
		"hook_event_name": "SessionEnd"
	}`

	reader := strings.NewReader(jsonData)
	event, err := DetectAndParse(reader)
	if err != nil {
		t.Fatalf("Failed to parse SessionEnd event: %v", err)
	}

	if event.Source != "claude-code" {
		t.Errorf("Expected source 'claude-code', got '%s'", event.Source)
	}

	if event.EventType != "SessionEnd" {
		t.Errorf("Expected event type 'SessionEnd', got '%s'", event.EventType)
	}

	if event.SessionID != "session-end-456" {
		t.Errorf("Expected session ID 'session-end-456', got '%s'", event.SessionID)
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

func TestGetClaudeCodeEvent(t *testing.T) {
	t.Run("returns event for Claude Code source", func(t *testing.T) {
		jsonData := `{
			"session_id": "test-session",
			"transcript_path": "/path/to/transcript.jsonl",
			"hook_event_name": "Stop",
			"stop_hook_active": true
		}`

		reader := strings.NewReader(jsonData)
		event, err := DetectAndParse(reader)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		rawEvent, ok := event.GetClaudeCodeEvent()
		if !ok {
			t.Error("Expected GetClaudeCodeEvent to return true for Claude Code event")
		}
		if rawEvent == nil {
			t.Error("Expected non-nil raw event")
		}
	})

	t.Run("returns false for Codex source", func(t *testing.T) {
		jsonData := `{
			"type": "agent-turn-complete",
			"thread-id": "12345678-1234-1234-1234-123456789abc",
			"turn-id": "1",
			"cwd": "/project",
			"input-messages": [],
			"last-assistant-message": "Done"
		}`

		reader := strings.NewReader(jsonData)
		event, err := DetectAndParse(reader)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		_, ok := event.GetClaudeCodeEvent()
		if ok {
			t.Error("Expected GetClaudeCodeEvent to return false for Codex event")
		}
	})
}

func TestDetectAndParseCursorSessionStart(t *testing.T) {
	jsonData := `{
		"conversation_id": "cursor-conv-123",
		"generation_id": "gen-456",
		"model": "claude-3.5-sonnet",
		"hook_event_name": "sessionStart",
		"cursor_version": "0.45.0",
		"workspace_roots": ["/home/user/project"],
		"user_email": "user@example.com"
	}`

	reader := strings.NewReader(jsonData)
	event, err := DetectAndParse(reader)
	if err != nil {
		t.Fatalf("Failed to parse Cursor event: %v", err)
	}

	if event.Source != "cursor" {
		t.Errorf("Expected source 'cursor', got '%s'", event.Source)
	}

	if event.SessionID != "cursor-conv-123" {
		t.Errorf("Expected session ID 'cursor-conv-123', got '%s'", event.SessionID)
	}

	if event.EventType != "sessionStart" {
		t.Errorf("Expected event type 'sessionStart', got '%s'", event.EventType)
	}

	if event.CWD != "/home/user/project" {
		t.Errorf("Expected CWD '/home/user/project', got '%s'", event.CWD)
	}

	if !event.IsCursor() {
		t.Error("Expected IsCursor() to return true")
	}

	if event.IsClaudeCode() {
		t.Error("Expected IsClaudeCode() to return false")
	}

	if event.IsCodex() {
		t.Error("Expected IsCodex() to return false")
	}
}

func TestDetectAndParseCursorBeforeSubmitPrompt(t *testing.T) {
	jsonData := `{
		"conversation_id": "cursor-conv-789",
		"generation_id": "gen-012",
		"model": "gpt-4o",
		"hook_event_name": "beforeSubmitPrompt",
		"cursor_version": "0.45.0",
		"workspace_roots": ["/home/user/project"],
		"prompt": "Fix the authentication bug"
	}`

	reader := strings.NewReader(jsonData)
	event, err := DetectAndParse(reader)
	if err != nil {
		t.Fatalf("Failed to parse Cursor event: %v", err)
	}

	if event.Source != "cursor" {
		t.Errorf("Expected source 'cursor', got '%s'", event.Source)
	}

	if event.EventType != "beforeSubmitPrompt" {
		t.Errorf("Expected event type 'beforeSubmitPrompt', got '%s'", event.EventType)
	}

	if len(event.UserInput) != 1 || event.UserInput[0] != "Fix the authentication bug" {
		t.Errorf("Unexpected user input: %v", event.UserInput)
	}
}

func TestDetectAndParseCursorStop(t *testing.T) {
	jsonData := `{
		"conversation_id": "cursor-conv-stop",
		"generation_id": "gen-stop",
		"model": "claude-3.5-sonnet",
		"hook_event_name": "stop",
		"cursor_version": "0.45.0",
		"workspace_roots": ["/home/user/project"]
	}`

	reader := strings.NewReader(jsonData)
	event, err := DetectAndParse(reader)
	if err != nil {
		t.Fatalf("Failed to parse Cursor stop event: %v", err)
	}

	if event.Source != "cursor" {
		t.Errorf("Expected source 'cursor', got '%s'", event.Source)
	}

	if event.EventType != "stop" {
		t.Errorf("Expected event type 'stop', got '%s'", event.EventType)
	}
}

func TestGetCursorEvent(t *testing.T) {
	t.Run("returns event for Cursor source", func(t *testing.T) {
		jsonData := `{
			"conversation_id": "cursor-conv-test",
			"generation_id": "gen-test",
			"model": "claude-3.5-sonnet",
			"hook_event_name": "sessionStart",
			"cursor_version": "0.45.0",
			"workspace_roots": ["/project"]
		}`

		reader := strings.NewReader(jsonData)
		event, err := DetectAndParse(reader)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		rawEvent, ok := event.GetCursorEvent()
		if !ok {
			t.Error("Expected GetCursorEvent to return true for Cursor event")
		}
		if rawEvent == nil {
			t.Error("Expected non-nil raw event")
		}
	})

	t.Run("returns false for Claude Code source", func(t *testing.T) {
		jsonData := `{
			"session_id": "claude-session",
			"transcript_path": "/path/transcript.jsonl",
			"hook_event_name": "SessionStart"
		}`

		reader := strings.NewReader(jsonData)
		event, err := DetectAndParse(reader)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		_, ok := event.GetCursorEvent()
		if ok {
			t.Error("Expected GetCursorEvent to return false for Claude Code event")
		}
	})
}

func TestParseClaudeCodeEventTypes(t *testing.T) {
	t.Run("parse PreToolUse event", func(t *testing.T) {
		jsonData := `{
			"session_id": "test-session",
			"transcript_path": "/path/transcript.jsonl",
			"hook_event_name": "PreToolUse",
			"tool_name": "Read"
		}`

		reader := strings.NewReader(jsonData)
		event, err := DetectAndParse(reader)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		if event.EventType != "PreToolUse" {
			t.Errorf("Expected event type 'PreToolUse', got '%s'", event.EventType)
		}
	})

	t.Run("parse PostToolUse event", func(t *testing.T) {
		jsonData := `{
			"session_id": "test-session",
			"transcript_path": "/path/transcript.jsonl",
			"hook_event_name": "PostToolUse",
			"tool_name": "Write"
		}`

		reader := strings.NewReader(jsonData)
		event, err := DetectAndParse(reader)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		if event.EventType != "PostToolUse" {
			t.Errorf("Expected event type 'PostToolUse', got '%s'", event.EventType)
		}
	})

	t.Run("parse SubagentStop event", func(t *testing.T) {
		jsonData := `{
			"session_id": "test-session",
			"transcript_path": "/path/transcript.jsonl",
			"hook_event_name": "SubagentStop"
		}`

		reader := strings.NewReader(jsonData)
		event, err := DetectAndParse(reader)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		if event.EventType != "SubagentStop" {
			t.Errorf("Expected event type 'SubagentStop', got '%s'", event.EventType)
		}
	})
}

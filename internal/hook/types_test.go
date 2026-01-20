package hook

import (
	"strings"
	"testing"
)

func TestParseHookEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		checkFunc func(*testing.T, *HookEvent)
	}{
		{
			name: "valid UserPromptSubmit event",
			input: `{
				"session_id": "test-session-123",
				"transcript_path": "/path/to/transcript.jsonl",
				"cwd": "/home/user/project",
				"hook_event_name": "UserPromptSubmit"
			}`,
			wantErr: false,
			checkFunc: func(t *testing.T, e *HookEvent) {
				if e.SessionID != "test-session-123" {
					t.Errorf("Expected session_id 'test-session-123', got '%s'", e.SessionID)
				}
				if e.TranscriptPath != "/path/to/transcript.jsonl" {
					t.Errorf("Expected transcript_path '/path/to/transcript.jsonl', got '%s'", e.TranscriptPath)
				}
				if e.CWD != "/home/user/project" {
					t.Errorf("Expected cwd '/home/user/project', got '%s'", e.CWD)
				}
				if e.HookEventName != "UserPromptSubmit" {
					t.Errorf("Expected hook_event_name 'UserPromptSubmit', got '%s'", e.HookEventName)
				}
			},
		},
		{
			name: "valid Stop event",
			input: `{
				"session_id": "stop-session",
				"transcript_path": "/transcript.jsonl",
				"hook_event_name": "Stop"
			}`,
			wantErr: false,
			checkFunc: func(t *testing.T, e *HookEvent) {
				if e.HookEventName != "Stop" {
					t.Errorf("Expected hook_event_name 'Stop', got '%s'", e.HookEventName)
				}
			},
		},
		{
			name:    "invalid JSON",
			input:   `{"invalid`,
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   ``,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			event, err := ParseHookEvent(reader)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, event)
			}
		})
	}
}

func TestParseUserPromptSubmitEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		checkFunc func(*testing.T, *UserPromptSubmitEvent)
	}{
		{
			name: "valid event with prompt",
			input: `{
				"session_id": "session-123",
				"transcript_path": "/path/transcript.jsonl",
				"cwd": "/project",
				"hook_event_name": "UserPromptSubmit",
				"prompt": "Help me fix this bug"
			}`,
			wantErr: false,
			checkFunc: func(t *testing.T, e *UserPromptSubmitEvent) {
				if e.Prompt != "Help me fix this bug" {
					t.Errorf("Expected prompt 'Help me fix this bug', got '%s'", e.Prompt)
				}
				if e.SessionID != "session-123" {
					t.Errorf("Expected session_id 'session-123', got '%s'", e.SessionID)
				}
			},
		},
		{
			name: "event without prompt",
			input: `{
				"session_id": "session-456",
				"hook_event_name": "UserPromptSubmit"
			}`,
			wantErr: false,
			checkFunc: func(t *testing.T, e *UserPromptSubmitEvent) {
				if e.Prompt != "" {
					t.Errorf("Expected empty prompt, got '%s'", e.Prompt)
				}
			},
		},
		{
			name:    "invalid JSON",
			input:   `not json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			event, err := ParseUserPromptSubmitEvent(reader)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, event)
			}
		})
	}
}

func TestParseStopEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		checkFunc func(*testing.T, *StopEvent)
	}{
		{
			name: "valid Stop event with stop_hook_active",
			input: `{
				"session_id": "stop-session",
				"transcript_path": "/transcript.jsonl",
				"hook_event_name": "Stop",
				"stop_hook_active": true
			}`,
			wantErr: false,
			checkFunc: func(t *testing.T, e *StopEvent) {
				if !e.StopHookActive {
					t.Error("Expected stop_hook_active to be true")
				}
				if e.HookEventName != "Stop" {
					t.Errorf("Expected hook_event_name 'Stop', got '%s'", e.HookEventName)
				}
			},
		},
		{
			name: "SubagentStop event",
			input: `{
				"session_id": "subagent-stop",
				"transcript_path": "/transcript.jsonl",
				"hook_event_name": "SubagentStop",
				"stop_hook_active": false
			}`,
			wantErr: false,
			checkFunc: func(t *testing.T, e *StopEvent) {
				if e.StopHookActive {
					t.Error("Expected stop_hook_active to be false")
				}
				if e.HookEventName != "SubagentStop" {
					t.Errorf("Expected hook_event_name 'SubagentStop', got '%s'", e.HookEventName)
				}
			},
		},
		{
			name:    "invalid JSON",
			input:   `{broken}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			event, err := ParseStopEvent(reader)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, event)
			}
		})
	}
}

func TestParseNotificationEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		checkFunc func(*testing.T, *NotificationEvent)
	}{
		{
			name: "valid Notification event",
			input: `{
				"session_id": "notify-session",
				"transcript_path": "/transcript.jsonl",
				"hook_event_name": "Notification",
				"message": "Task completed successfully"
			}`,
			wantErr: false,
			checkFunc: func(t *testing.T, e *NotificationEvent) {
				if e.Message != "Task completed successfully" {
					t.Errorf("Expected message 'Task completed successfully', got '%s'", e.Message)
				}
				if e.HookEventName != "Notification" {
					t.Errorf("Expected hook_event_name 'Notification', got '%s'", e.HookEventName)
				}
			},
		},
		{
			name: "Notification with empty message",
			input: `{
				"session_id": "notify-empty",
				"hook_event_name": "Notification",
				"message": ""
			}`,
			wantErr: false,
			checkFunc: func(t *testing.T, e *NotificationEvent) {
				if e.Message != "" {
					t.Errorf("Expected empty message, got '%s'", e.Message)
				}
			},
		},
		{
			name:    "invalid JSON",
			input:   `{"message": `,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			event, err := ParseNotificationEvent(reader)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, event)
			}
		})
	}
}

func TestParseSessionStartEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		checkFunc func(*testing.T, *SessionStartEvent)
	}{
		{
			name: "valid SessionStart event",
			input: `{
				"session_id": "start-session",
				"transcript_path": "/transcript.jsonl",
				"cwd": "/home/user/project",
				"hook_event_name": "SessionStart"
			}`,
			wantErr: false,
			checkFunc: func(t *testing.T, e *SessionStartEvent) {
				if e.SessionID != "start-session" {
					t.Errorf("Expected session_id 'start-session', got '%s'", e.SessionID)
				}
				if e.CWD != "/home/user/project" {
					t.Errorf("Expected cwd '/home/user/project', got '%s'", e.CWD)
				}
				if e.HookEventName != "SessionStart" {
					t.Errorf("Expected hook_event_name 'SessionStart', got '%s'", e.HookEventName)
				}
			},
		},
		{
			name:    "invalid JSON",
			input:   `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			event, err := ParseSessionStartEvent(reader)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, event)
			}
		})
	}
}

func TestParseSessionEndEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		checkFunc func(*testing.T, *SessionEndEvent)
	}{
		{
			name: "valid SessionEnd event",
			input: `{
				"session_id": "end-session",
				"transcript_path": "/transcript.jsonl",
				"cwd": "/project",
				"hook_event_name": "SessionEnd"
			}`,
			wantErr: false,
			checkFunc: func(t *testing.T, e *SessionEndEvent) {
				if e.SessionID != "end-session" {
					t.Errorf("Expected session_id 'end-session', got '%s'", e.SessionID)
				}
				if e.HookEventName != "SessionEnd" {
					t.Errorf("Expected hook_event_name 'SessionEnd', got '%s'", e.HookEventName)
				}
			},
		},
		{
			name:    "empty input",
			input:   ``,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			event, err := ParseSessionEndEvent(reader)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, event)
			}
		})
	}
}

func TestParseCodexNotifyEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		checkFunc func(*testing.T, *CodexNotifyEvent)
	}{
		{
			name: "valid agent-turn-complete event",
			input: `{
				"type": "agent-turn-complete",
				"thread-id": "12345678-1234-1234-1234-123456789abc",
				"turn-id": 5,
				"cwd": "/home/user/project",
				"input-messages": ["Fix the bug", "Also update tests"],
				"last-assistant-message": "Done!"
			}`,
			wantErr: false,
			checkFunc: func(t *testing.T, e *CodexNotifyEvent) {
				if e.Type != "agent-turn-complete" {
					t.Errorf("Expected type 'agent-turn-complete', got '%s'", e.Type)
				}
				if e.ThreadID != "12345678-1234-1234-1234-123456789abc" {
					t.Errorf("Expected thread_id '12345678-1234-1234-1234-123456789abc', got '%s'", e.ThreadID)
				}
				if e.TurnID != 5 {
					t.Errorf("Expected turn_id 5, got %d", e.TurnID)
				}
				if e.CWD != "/home/user/project" {
					t.Errorf("Expected cwd '/home/user/project', got '%s'", e.CWD)
				}
				if len(e.InputMessages) != 2 {
					t.Errorf("Expected 2 input messages, got %d", len(e.InputMessages))
				}
				if e.LastAssistantMessage != "Done!" {
					t.Errorf("Expected last_assistant_message 'Done!', got '%s'", e.LastAssistantMessage)
				}
			},
		},
		{
			name: "event with empty input messages",
			input: `{
				"type": "agent-turn-complete",
				"thread-id": "abcd-efgh",
				"turn-id": 1,
				"cwd": "/project",
				"input-messages": [],
				"last-assistant-message": ""
			}`,
			wantErr: false,
			checkFunc: func(t *testing.T, e *CodexNotifyEvent) {
				if len(e.InputMessages) != 0 {
					t.Errorf("Expected empty input messages, got %d", len(e.InputMessages))
				}
				if e.TurnID != 1 {
					t.Errorf("Expected turn_id 1, got %d", e.TurnID)
				}
			},
		},
		{
			name:    "invalid JSON",
			input:   `{"type": agent-turn`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			event, err := ParseCodexNotifyEvent(reader)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, event)
			}
		})
	}
}

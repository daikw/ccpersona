// Package hook provides unified handling for both Claude Code and Codex hook events
package hook

import (
	"encoding/json"
	"fmt"
	"io"
)

// UnifiedHookEvent represents a normalized hook event that works for Claude Code, Codex, and Cursor
type UnifiedHookEvent struct {
	Source     string      // "claude-code", "codex", or "cursor"
	SessionID  string      // Session/Thread/Conversation identifier
	CWD        string      // Current working directory
	EventType  string      // Event type (e.g., "UserPromptSubmit", "agent-turn-complete", "sessionStart")
	UserInput  []string    // User's input messages
	AIResponse string      // AI's response message
	RawEvent   interface{} // Original event for type-specific handling
}

// DetectAndParse automatically detects the hook source and parses the event
func DetectAndParse(r io.Reader) (*UnifiedHookEvent, error) {
	// Read all data from reader
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	// Try to parse as a generic JSON first to detect the source
	var generic map[string]interface{}
	if err := json.Unmarshal(data, &generic); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Detect source by checking for distinctive fields
	if _, hasType := generic["type"]; hasType {
		if typeVal, ok := generic["type"].(string); ok && typeVal == "agent-turn-complete" {
			// This is a Codex event
			return parseCodexEvent(data)
		}
	}

	if _, hasHookEventName := generic["hook_event_name"]; hasHookEventName {
		// Distinguish between Claude Code and Cursor by checking for conversation_id
		// Cursor uses conversation_id, Claude Code uses session_id
		if _, hasConversationID := generic["conversation_id"]; hasConversationID {
			// This is a Cursor event
			return parseCursorEvent(data, generic)
		}
		// This is a Claude Code event
		return parseClaudeCodeEvent(data, generic)
	}

	return nil, fmt.Errorf("unknown hook event format")
}

func parseCodexEvent(data []byte) (*UnifiedHookEvent, error) {
	var event CodexNotifyEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to parse Codex event: %w", err)
	}

	return &UnifiedHookEvent{
		Source:     "codex",
		SessionID:  event.ThreadID,
		CWD:        event.CWD,
		EventType:  event.Type,
		UserInput:  event.InputMessages,
		AIResponse: event.LastAssistantMessage,
		RawEvent:   &event,
	}, nil
}

func parseClaudeCodeEvent(data []byte, generic map[string]interface{}) (*UnifiedHookEvent, error) {
	hookEventName, _ := generic["hook_event_name"].(string)

	switch hookEventName {
	case "UserPromptSubmit":
		var event UserPromptSubmitEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse UserPromptSubmit event: %w", err)
		}
		return &UnifiedHookEvent{
			Source:     "claude-code",
			SessionID:  event.SessionID,
			CWD:        event.CWD,
			EventType:  hookEventName,
			UserInput:  []string{event.Prompt},
			AIResponse: "",
			RawEvent:   &event,
		}, nil

	case "Stop", "SubagentStop":
		var event StopEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse Stop event: %w", err)
		}
		return &UnifiedHookEvent{
			Source:     "claude-code",
			SessionID:  event.SessionID,
			CWD:        event.CWD,
			EventType:  hookEventName,
			UserInput:  []string{},
			AIResponse: "", // Will be read from transcript
			RawEvent:   &event,
		}, nil

	case "Notification":
		var event NotificationEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse Notification event: %w", err)
		}
		return &UnifiedHookEvent{
			Source:     "claude-code",
			SessionID:  event.SessionID,
			CWD:        event.CWD,
			EventType:  hookEventName,
			UserInput:  []string{},
			AIResponse: event.Message,
			RawEvent:   &event,
		}, nil

	case "SessionStart":
		var event SessionStartEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse SessionStart event: %w", err)
		}
		return &UnifiedHookEvent{
			Source:     "claude-code",
			SessionID:  event.SessionID,
			CWD:        event.CWD,
			EventType:  hookEventName,
			UserInput:  []string{},
			AIResponse: "",
			RawEvent:   &event,
		}, nil

	case "SessionEnd":
		var event SessionEndEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse SessionEnd event: %w", err)
		}
		return &UnifiedHookEvent{
			Source:     "claude-code",
			SessionID:  event.SessionID,
			CWD:        event.CWD,
			EventType:  hookEventName,
			UserInput:  []string{},
			AIResponse: "",
			RawEvent:   &event,
		}, nil

	default:
		// For other Claude Code events, just parse as generic HookEvent
		var event HookEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse Claude Code event: %w", err)
		}
		return &UnifiedHookEvent{
			Source:     "claude-code",
			SessionID:  event.SessionID,
			CWD:        event.CWD,
			EventType:  hookEventName,
			UserInput:  []string{},
			AIResponse: "",
			RawEvent:   &event,
		}, nil
	}
}

func parseCursorEvent(data []byte, generic map[string]interface{}) (*UnifiedHookEvent, error) {
	hookEventName, _ := generic["hook_event_name"].(string)

	// Get CWD from workspace_roots if available
	cwd := ""
	if roots, ok := generic["workspace_roots"].([]interface{}); ok && len(roots) > 0 {
		if root, ok := roots[0].(string); ok {
			cwd = root
		}
	}

	switch hookEventName {
	case "sessionStart":
		var event CursorSessionStartEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse Cursor sessionStart event: %w", err)
		}
		return &UnifiedHookEvent{
			Source:     "cursor",
			SessionID:  event.ConversationID,
			CWD:        cwd,
			EventType:  hookEventName,
			UserInput:  []string{},
			AIResponse: "",
			RawEvent:   &event,
		}, nil

	case "beforeSubmitPrompt":
		var event CursorBeforeSubmitPromptEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse Cursor beforeSubmitPrompt event: %w", err)
		}
		return &UnifiedHookEvent{
			Source:     "cursor",
			SessionID:  event.ConversationID,
			CWD:        cwd,
			EventType:  hookEventName,
			UserInput:  []string{event.Prompt},
			AIResponse: "",
			RawEvent:   &event,
		}, nil

	case "stop":
		var event CursorStopEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse Cursor stop event: %w", err)
		}
		return &UnifiedHookEvent{
			Source:     "cursor",
			SessionID:  event.ConversationID,
			CWD:        cwd,
			EventType:  hookEventName,
			UserInput:  []string{},
			AIResponse: "",
			RawEvent:   &event,
		}, nil

	case "afterAgentResponse":
		var event CursorAfterAgentResponseEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse Cursor afterAgentResponse event: %w", err)
		}
		return &UnifiedHookEvent{
			Source:     "cursor",
			SessionID:  event.ConversationID,
			CWD:        cwd,
			EventType:  hookEventName,
			UserInput:  []string{},
			AIResponse: event.Text, // AI response is directly in the event
			RawEvent:   &event,
		}, nil

	default:
		// For other Cursor events, parse as generic CursorHookEvent
		var event CursorHookEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to parse Cursor event: %w", err)
		}
		return &UnifiedHookEvent{
			Source:     "cursor",
			SessionID:  event.ConversationID,
			CWD:        cwd,
			EventType:  hookEventName,
			UserInput:  []string{},
			AIResponse: "",
			RawEvent:   &event,
		}, nil
	}
}

// IsCodex returns true if the event is from Codex
func (e *UnifiedHookEvent) IsCodex() bool {
	return e.Source == "codex"
}

// IsClaudeCode returns true if the event is from Claude Code
func (e *UnifiedHookEvent) IsClaudeCode() bool {
	return e.Source == "claude-code"
}

// IsCursor returns true if the event is from Cursor
func (e *UnifiedHookEvent) IsCursor() bool {
	return e.Source == "cursor"
}

// GetCodexEvent returns the underlying Codex event if available
func (e *UnifiedHookEvent) GetCodexEvent() (*CodexNotifyEvent, bool) {
	if event, ok := e.RawEvent.(*CodexNotifyEvent); ok {
		return event, true
	}
	return nil, false
}

// GetClaudeCodeEvent returns the underlying Claude Code event if available
func (e *UnifiedHookEvent) GetClaudeCodeEvent() (interface{}, bool) {
	if e.IsClaudeCode() {
		return e.RawEvent, true
	}
	return nil, false
}

// GetCursorEvent returns the underlying Cursor event if available
func (e *UnifiedHookEvent) GetCursorEvent() (interface{}, bool) {
	if e.IsCursor() {
		return e.RawEvent, true
	}
	return nil, false
}

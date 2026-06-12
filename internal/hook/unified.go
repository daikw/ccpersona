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

	// Source detection is decoupled from event-type interpretation.
	// Codex is identified by its source-level shape (a "type" field corroborated
	// by Codex-specific fields, or the known type value alone). The concrete
	// value of "type" is interpreted downstream in parseCodexEvent, so a future
	// Codex event type beyond "agent-turn-complete" is still recognized as Codex
	// as long as it carries Codex-specific fields.
	if isCodexEvent(generic) {
		return parseCodexEvent(data)
	}

	// Both Claude Code and Cursor carry hook_event_name; disambiguate by scoring
	// multiple signals rather than trusting a single field, so one schema change
	// cannot silently misroute a Cursor event into the Claude Code parser.
	if _, hasHookEventName := generic["hook_event_name"]; hasHookEventName {
		if isCursorEvent(generic) {
			return parseCursorEvent(data, generic)
		}
		return parseClaudeCodeEvent(data, generic)
	}

	return nil, fmt.Errorf("unknown hook event format")
}

// isCodexEvent reports whether the generic payload looks like a Codex notify
// event. A "type" field is necessary but not sufficient: another source could
// add a "type" field someday, and parseCodexEvent does not validate required
// fields, so matching on "type" alone would silently misroute such events with
// an empty ThreadID. Corroboration order: Codex-specific fields admit unknown
// future type values; without them, only the known type value is accepted.
func isCodexEvent(generic map[string]interface{}) bool {
	if _, hasType := generic["type"]; !hasType {
		return false
	}
	if hasAnyKey(generic, "thread-id", "turn-id", "input-messages", "last-assistant-message") {
		return true
	}
	if t, ok := generic["type"].(string); ok && t == "agent-turn-complete" {
		return true
	}
	return false
}

// isCursorEvent decides Cursor vs Claude Code by majority vote over independent
// signals, since both sources carry hook_event_name. Signals (each +1 toward
// Cursor): a conversation_id field, a Cursor-specific field (workspace_roots /
// cursor_version / generation_id), and a camelCase hook_event_name (Cursor uses
// camelCase like "sessionStart"; Claude Code uses PascalCase like "SessionStart").
//
// Requiring >= 2 agreeing signals avoids misrouting on a single schema change.
// On a tie or insufficient signal we fall back to the legacy rule
// (conversation_id present => Cursor) to preserve prior behavior exactly.
func isCursorEvent(generic map[string]interface{}) bool {
	_, hasConversationID := generic["conversation_id"]

	score := 0
	if hasConversationID {
		score++
	}
	if hasAnyKey(generic, "workspace_roots", "cursor_version", "generation_id") {
		score++
	}
	if name, ok := generic["hook_event_name"].(string); ok && isCamelCase(name) {
		score++
	}

	if score >= 2 {
		return true
	}
	// Fallback: legacy single-field rule for compatibility.
	return hasConversationID
}

func hasAnyKey(m map[string]interface{}, keys ...string) bool {
	for _, k := range keys {
		if _, ok := m[k]; ok {
			return true
		}
	}
	return false
}

// isCamelCase reports whether s begins with a lowercase ASCII letter, which
// distinguishes Cursor's camelCase event names from Claude Code's PascalCase.
func isCamelCase(s string) bool {
	if s == "" {
		return false
	}
	c := s[0]
	return c >= 'a' && c <= 'z'
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

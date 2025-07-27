// Package hook provides types and utilities for handling Claude Code hook events.
// For more information about Claude Code hooks, see:
// https://docs.anthropic.com/en/docs/claude-code/hooks
package hook

import (
	"encoding/json"
	"io"
	"os"
)

// HookEvent represents the common hook event data from Claude Code
type HookEvent struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	CWD            string `json:"cwd,omitempty"`
	HookEventName  string `json:"hook_event_name"`
}

// UserPromptSubmitEvent represents the UserPromptSubmit hook event
type UserPromptSubmitEvent struct {
	HookEvent
	Prompt string `json:"prompt"`
}

// StopEvent represents the Stop/SubagentStop hook event
type StopEvent struct {
	HookEvent
	StopHookActive bool `json:"stop_hook_active,omitempty"`
}

// NotificationEvent represents the Notification hook event
type NotificationEvent struct {
	HookEvent
	Message string `json:"message"`
}

// PreToolUseEvent represents the PreToolUse hook event
type PreToolUseEvent struct {
	HookEvent
	ToolName   string                 `json:"tool_name"`
	ToolParams map[string]interface{} `json:"tool_params,omitempty"`
}

// PostToolUseEvent represents the PostToolUse hook event
type PostToolUseEvent struct {
	HookEvent
	ToolName   string                 `json:"tool_name"`
	ToolResult map[string]interface{} `json:"tool_result,omitempty"`
}

// PreCompactEvent represents the PreCompact hook event
type PreCompactEvent struct {
	HookEvent
	CompactMode string `json:"compact_mode"` // "manual" or "auto"
}

// ParseHookEvent reads and parses the hook event from stdin
func ParseHookEvent(r io.Reader) (*HookEvent, error) {
	var event HookEvent
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ParseUserPromptSubmitEvent reads and parses UserPromptSubmit event from stdin
func ParseUserPromptSubmitEvent(r io.Reader) (*UserPromptSubmitEvent, error) {
	var event UserPromptSubmitEvent
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ParseStopEvent reads and parses Stop event from stdin
func ParseStopEvent(r io.Reader) (*StopEvent, error) {
	var event StopEvent
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ParseNotificationEvent reads and parses Notification event from stdin
func ParseNotificationEvent(r io.Reader) (*NotificationEvent, error) {
	var event NotificationEvent
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ReadHookEvent is a convenience function to read hook event from stdin
func ReadHookEvent() (*HookEvent, error) {
	return ParseHookEvent(os.Stdin)
}

// ReadUserPromptSubmitEvent is a convenience function to read UserPromptSubmit event from stdin
func ReadUserPromptSubmitEvent() (*UserPromptSubmitEvent, error) {
	return ParseUserPromptSubmitEvent(os.Stdin)
}

// ReadStopEvent is a convenience function to read Stop event from stdin
func ReadStopEvent() (*StopEvent, error) {
	return ParseStopEvent(os.Stdin)
}

// ReadNotificationEvent is a convenience function to read Notification event from stdin
func ReadNotificationEvent() (*NotificationEvent, error) {
	return ParseNotificationEvent(os.Stdin)
}
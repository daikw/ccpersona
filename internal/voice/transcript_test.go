package voice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadLinesReverseLongLine tests reading files with very long lines
func TestReadLinesReverseLongLine(t *testing.T) {
	// Create a test file with a very long line
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "long_line.jsonl")

	// Create a long content (100KB)
	longContent := strings.Repeat("A", 100000)

	lines := []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{
		{Type: "short", Text: "Short line"},
		{Type: "long", Text: longContent},
		{Type: "end", Text: "End line"},
	}

	// Write test file
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = f.Close()
	})

	encoder := json.NewEncoder(f)
	for _, line := range lines {
		if err := encoder.Encode(line); err != nil {
			t.Fatal(err)
		}
	}

	// Test reading
	reader := NewTranscriptReader(DefaultConfig())

	file, err := os.Open(testFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = file.Close()
	})

	readLines, err := reader.readLinesReverse(file)
	if err != nil {
		t.Fatalf("Failed to read long line: %v", err)
	}

	// Verify
	if len(readLines) != 3 {
		t.Fatalf("Expected 3 lines, got %d", len(readLines))
	}

	// Check reversed order
	var lastLine struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(readLines[0]), &lastLine); err != nil {
		t.Fatal(err)
	}

	if lastLine.Type != "end" {
		t.Errorf("Expected first line to be 'end', got %s", lastLine.Type)
	}

	// Check long line
	var longLine struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(readLines[1]), &longLine); err != nil {
		t.Fatal(err)
	}

	if len(longLine.Text) != 100000 {
		t.Errorf("Expected long line to be 100000 chars, got %d", len(longLine.Text))
	}
}

// TestGetLatestAssistantMessage tests parsing Claude Code transcript format
func TestGetLatestAssistantMessage(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "transcript.jsonl")

	// Create test transcript in Claude Code format
	messages := []TranscriptMessage{
		{
			Type: "start",
		},
		{
			Type: "user",
			UUID: "user-1",
			Message: struct {
				Role    string `json:"role"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
				} `json:"content"`
			}{
				Role: "user",
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
				}{
					{Type: "text", Text: "Hello"},
				},
			},
		},
		{
			Type: "assistant",
			UUID: "assistant-1",
			Message: struct {
				Role    string `json:"role"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
				} `json:"content"`
			}{
				Role: "assistant",
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
				}{
					{Type: "thinking"},
					{Type: "text", Text: "Hello! How can I help you?"},
				},
			},
		},
		{
			Type: "stop",
		},
	}

	// Write test file
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = f.Close()
	})

	encoder := json.NewEncoder(f)
	for _, msg := range messages {
		if err := encoder.Encode(msg); err != nil {
			t.Fatal(err)
		}
	}

	// Test
	reader := NewTranscriptReader(DefaultConfig())

	text, err := reader.GetLatestAssistantMessage(testFile)
	if err != nil {
		t.Fatalf("Failed to get assistant message: %v", err)
	}

	expected := "Hello! How can I help you?"
	if text != expected {
		t.Errorf("Expected '%s', got '%s'", expected, text)
	}
}

// TestGetLatestAssistantMessageNoText tests handling messages without text content
func TestGetLatestAssistantMessageNoText(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "transcript_no_text.jsonl")

	// Create test transcript with assistant message without text
	messages := []TranscriptMessage{
		{
			Type: "assistant",
			UUID: "assistant-1",
			Message: struct {
				Role    string `json:"role"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
				} `json:"content"`
			}{
				Role: "assistant",
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
				}{
					{Type: "thinking"},
					{Type: "tool_use"},
				},
			},
		},
	}

	// Write test file
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = f.Close()
	})

	encoder := json.NewEncoder(f)
	for _, msg := range messages {
		if err := encoder.Encode(msg); err != nil {
			t.Fatal(err)
		}
	}

	// Test
	reader := NewTranscriptReader(DefaultConfig())

	_, err = reader.GetLatestAssistantMessage(testFile)
	if err == nil {
		t.Error("Expected error for no text content, got nil")
	}

	if !strings.Contains(err.Error(), "no assistant message found") {
		t.Errorf("Expected 'no assistant message found' error, got: %v", err)
	}
}

// TestProcessTextModes tests different reading modes
func TestProcessTextModes(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		input    string
		expected string
		config   *Config
	}{
		{
			name:     "FirstLine",
			mode:     ModeFirstLine,
			input:    "First line\nSecond line\nThird line",
			expected: "First line",
			config:   DefaultConfig(),
		},
		{
			name:     "LineLimit",
			mode:     ModeLineLimit,
			input:    "Line 1\nLine 2\nLine 3\nLine 4",
			expected: "Line 1 Line 2 Line 3",
			config: &Config{
				ReadingMode: ModeLineLimit,
				MaxLines:    3,
			},
		},
		{
			name:     "AfterFirst",
			mode:     ModeAfterFirst,
			input:    "Skip this\nKeep this\nAnd this",
			expected: "Keep this And this",
			config: &Config{
				ReadingMode: ModeAfterFirst,
			},
		},
		{
			name:     "FullText",
			mode:     ModeFullText,
			input:    "All\nof\nthis\ntext",
			expected: "All of this text",
			config: &Config{
				ReadingMode: ModeFullText,
			},
		},
		{
			name:     "CharLimit",
			mode:     ModeCharLimit,
			input:    "This is a long text that should be truncated",
			expected: "This is a long t",
			config: &Config{
				ReadingMode: ModeCharLimit,
				MaxChars:    16,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewTranscriptReader(tt.config)
			result := reader.ProcessText(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestGetMessageWithUUID tests UUID mode message extraction
func TestGetMessageWithUUID(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "transcript_uuid.jsonl")

	// Create test transcript with multiple messages with same UUID
	messages := []TranscriptMessage{
		{
			Type: "assistant",
			UUID: "msg-1",
			Message: struct {
				Role    string `json:"role"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
				} `json:"content"`
			}{
				Role: "assistant",
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
				}{
					{Type: "text", Text: "Part 1"},
				},
			},
		},
		{
			Type: "assistant",
			UUID: "msg-1",
			Message: struct {
				Role    string `json:"role"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
				} `json:"content"`
			}{
				Role: "assistant",
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
				}{
					{Type: "text", Text: "Part 2"},
				},
			},
		},
		{
			Type: "assistant",
			UUID: "msg-2", // Different UUID
			Message: struct {
				Role    string `json:"role"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
				} `json:"content"`
			}{
				Role: "assistant",
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
				}{
					{Type: "text", Text: "Different message"},
				},
			},
		},
	}

	// Write test file
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = f.Close()
	})

	encoder := json.NewEncoder(f)
	for _, msg := range messages {
		if err := encoder.Encode(msg); err != nil {
			t.Fatal(err)
		}
	}

	// Test with UUID mode
	config := DefaultConfig()
	config.UUIDMode = true
	reader := NewTranscriptReader(config)

	text, err := reader.GetLatestAssistantMessage(testFile)
	if err != nil {
		t.Fatalf("Failed to get assistant message: %v", err)
	}

	// Should get the latest UUID (msg-2)
	expected := "Different message"
	if text != expected {
		t.Errorf("Expected '%s', got '%s'", expected, text)
	}
}

// TestReadLinesWithTruncation tests reading files with extremely long lines (>1MB)
func TestReadLinesWithTruncation(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "very_long_line.jsonl")

	// Create a very long content (2MB)
	veryLongContent := strings.Repeat("A", 2*1024*1024)

	lines := []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{
		{Type: "short", Text: "Short line"},
		{Type: "very_long", Text: veryLongContent},
		{Type: "end", Text: "End line"},
	}

	// Write test file
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = f.Close()
	})

	encoder := json.NewEncoder(f)
	for _, line := range lines {
		if err := encoder.Encode(line); err != nil {
			t.Fatal(err)
		}
	}

	// Test reading with truncation fallback
	reader := NewTranscriptReader(DefaultConfig())

	file, err := os.Open(testFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = file.Close()
	})

	readLines, err := reader.readLinesReverse(file)
	if err != nil {
		t.Fatalf("Failed to read file with very long lines: %v", err)
	}

	// Verify we got lines (should have triggered truncation)
	if len(readLines) == 0 {
		t.Fatal("Expected to read some lines, got none")
	}

	// Check that we can parse at least one line
	foundValidLine := false
	for _, line := range readLines {
		var parsed struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(line), &parsed); err == nil {
			foundValidLine = true
			break
		}
	}

	if !foundValidLine {
		t.Error("Expected to find at least one valid JSON line after truncation")
	}
}

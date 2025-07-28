package voice

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
)

// TranscriptReader reads Claude Code transcript files
type TranscriptReader struct {
	config *Config
}

// NewTranscriptReader creates a new transcript reader
func NewTranscriptReader(config *Config) *TranscriptReader {
	return &TranscriptReader{
		config: config,
	}
}

// FindLatestTranscript finds the most recent transcript file
func (tr *TranscriptReader) FindLatestTranscript() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	projectsDir := filepath.Join(homeDir, ".claude", "projects")
	
	var transcriptFiles []string
	err = filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() && strings.HasSuffix(path, ".jsonl") {
			transcriptFiles = append(transcriptFiles, path)
		}
		return nil
	})
	
	if err != nil {
		return "", fmt.Errorf("failed to walk projects directory: %w", err)
	}

	if len(transcriptFiles) == 0 {
		return "", fmt.Errorf("no transcript files found")
	}

	// Sort by modification time (newest first)
	sort.Slice(transcriptFiles, func(i, j int) bool {
		infoI, errI := os.Stat(transcriptFiles[i])
		infoJ, errJ := os.Stat(transcriptFiles[j])
		if errI != nil || errJ != nil {
			return false
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})

	log.Debug().Str("file", transcriptFiles[0]).Msg("Found latest transcript")
	return transcriptFiles[0], nil
}

// GetLatestAssistantMessage extracts the latest assistant message from transcript
func (tr *TranscriptReader) GetLatestAssistantMessage(transcriptPath string) (string, error) {
	file, err := os.Open(transcriptPath)
	if err != nil {
		return "", fmt.Errorf("failed to open transcript: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	if tr.config.UUIDMode {
		return tr.getMessageWithUUID(file)
	}
	return tr.getMessageSimple(file)
}

// getMessageSimple extracts the first text from the latest assistant message (fast mode)
func (tr *TranscriptReader) getMessageSimple(file *os.File) (string, error) {
	// Read file in reverse to find the latest assistant message
	lines, err := tr.readLinesReverse(file)
	if err != nil {
		return "", err
	}

	for _, line := range lines {
		var msg TranscriptMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		if msg.Type == "assistant" && msg.Message.Role == "assistant" {
			// Find first text content
			for _, content := range msg.Message.Content {
				if content.Type == "text" && content.Text != "" {
					log.Debug().Str("text_length", fmt.Sprintf("%d", len(content.Text))).Msg("Found assistant message")
					return content.Text, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no assistant message found")
}

// getMessageWithUUID extracts all text from messages with the same UUID (complete mode)
func (tr *TranscriptReader) getMessageWithUUID(file *os.File) (string, error) {
	lines, err := tr.readLinesReverse(file)
	if err != nil {
		return "", err
	}

	// Find the latest assistant UUID
	var latestUUID string
	for _, line := range lines {
		var msg TranscriptMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		if msg.Type == "assistant" && msg.UUID != "" {
			latestUUID = msg.UUID
			break
		}
	}

	if latestUUID == "" {
		return "", fmt.Errorf("no assistant message found")
	}

	// Collect all text from messages with this UUID
	var texts []string
	for _, line := range lines {
		var msg TranscriptMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		if msg.UUID == latestUUID && msg.Message.Role == "assistant" {
			for _, content := range msg.Message.Content {
				if content.Type == "text" && content.Text != "" {
					texts = append(texts, content.Text)
				}
			}
		}
	}

	if len(texts) == 0 {
		return "", fmt.Errorf("no assistant message found")
	}

	// Reverse to get original order
	for i := len(texts)/2 - 1; i >= 0; i-- {
		opp := len(texts) - 1 - i
		texts[i], texts[opp] = texts[opp], texts[i]
	}

	result := strings.Join(texts, " ")
	log.Debug().
		Str("uuid", latestUUID).
		Int("text_count", len(texts)).
		Int("total_length", len(result)).
		Msg("Found assistant message with UUID")
	
	return result, nil
}

// readLinesReverse reads file lines in reverse order
func (tr *TranscriptReader) readLinesReverse(file *os.File) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(file)
	
	// Increase buffer size to handle very long lines (1MB instead of default 64KB)
	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)
	
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	
	if err := scanner.Err(); err != nil {
		// If we hit a "token too long" error, try to read with line truncation
		if strings.Contains(err.Error(), "token too long") {
			log.Warn().Msg("Very long line detected, attempting to read with truncation")
			return tr.readLinesWithTruncation(file)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Reverse the lines
	for i := len(lines)/2 - 1; i >= 0; i-- {
		opp := len(lines) - 1 - i
		lines[i], lines[opp] = lines[opp], lines[i]
	}

	return lines, nil
}

// readLinesWithTruncation reads file lines with truncation for extremely long lines
func (tr *TranscriptReader) readLinesWithTruncation(file *os.File) ([]string, error) {
	// Reset file position to beginning
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to reset file position: %w", err)
	}
	
	var lines []string
	const maxLineLength = 512 * 1024 // 512KB max per line
	
	reader := bufio.NewReader(file)
	lineNumber := 0
	
	for {
		lineNumber++
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read line %d: %w", lineNumber, err)
		}
		
		// Remove trailing newline
		line = strings.TrimRight(line, "\n\r")
		
		// Truncate if too long
		if len(line) > maxLineLength {
			originalLength := len(line)
			line = line[:maxLineLength]
			log.Warn().
				Int("line_number", lineNumber).
				Int("original_length", originalLength).
				Int("truncated_length", maxLineLength).
				Msg("Truncated extremely long line")
		}
		
		if line != "" {
			lines = append(lines, line)
		}
		
		if err == io.EOF {
			break
		}
	}
	
	// Reverse the lines
	for i := len(lines)/2 - 1; i >= 0; i-- {
		opp := len(lines) - 1 - i
		lines[i], lines[opp] = lines[opp], lines[i]
	}
	
	log.Info().Int("total_lines", len(lines)).Msg("Successfully read file with line truncation")
	return lines, nil
}

// ProcessText applies reading mode restrictions to the text
func (tr *TranscriptReader) ProcessText(text string) string {
	switch tr.config.ReadingMode {
	case ModeFirstLine:
		// First line only
		if idx := strings.Index(text, "\n"); idx != -1 {
			text = text[:idx]
		}
		
	case ModeLineLimit:
		// Limited number of lines
		lines := strings.Split(text, "\n")
		if len(lines) > tr.config.MaxLines {
			lines = lines[:tr.config.MaxLines]
		}
		text = strings.Join(lines, " ")
		
	case ModeAfterFirst:
		// Skip first line
		if idx := strings.Index(text, "\n"); idx != -1 && idx < len(text)-1 {
			text = text[idx+1:]
		}
		text = strings.ReplaceAll(text, "\n", " ")
		
	case ModeFullText:
		// Full text with newlines replaced by spaces
		text = strings.ReplaceAll(text, "\n", " ")
		
	case ModeCharLimit:
		// Character limit
		text = strings.ReplaceAll(text, "\n", " ")
		runes := []rune(text)
		if len(runes) > tr.config.MaxChars {
			text = string(runes[:tr.config.MaxChars])
		}
	}

	log.Debug().
		Str("mode", tr.config.ReadingMode).
		Int("length", len(text)).
		Msg("Processed text")

	return text
}
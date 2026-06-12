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
				if content.Type == "text" && strings.TrimSpace(content.Text) != "" {
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

// tailWindowSize is the chunk of the transcript tail read per pass. Transcript
// JSONL files can reach hundreds of MB, so we never load the whole file: we
// read a fixed-size window from the end and grow it only if the target line is
// not yet within reach.
const tailWindowSize = 1024 * 1024 // 1MB

// readLinesReverse returns the complete lines from the tail of the file in
// reverse order (newest first), growing the tail window until the
// mode-specific stop condition is met or the whole file has been read.
//
// The stop condition differs per mode because of what each consumer needs:
//   - simple mode reads only the newest assistant line, so one assistant line
//     in the window is enough.
//   - UUID mode joins every fragment of the newest assistant message, which
//     spans multiple JSONL lines. A window holding only one assistant UUID may
//     have cut off earlier fragments of that same message, so we require two
//     distinct assistant UUIDs: seeing an older message's UUID proves all
//     fragments of the newer one lie after it, i.e. fully inside the window.
func (tr *TranscriptReader) readLinesReverse(file *os.File) ([]string, error) {
	stop := containsAssistantLine
	if tr.config.UUIDMode {
		stop = containsTwoAssistantUUIDs
	}
	return tr.readLinesReverseUntil(file, stop)
}

// readLinesReverseUntil reads tail windows of increasing size until stop
// reports the target is fully within the window or the file start is reached,
// keeping at most one window resident in memory.
func (tr *TranscriptReader) readLinesReverseUntil(file *os.File, stop func([]string) bool) ([]string, error) {
	size, err := fileSize(file)
	if err != nil {
		return nil, err
	}

	window := int64(tailWindowSize)
	for {
		atFileStart := window >= size
		readFrom := size - window
		if readFrom < 0 {
			readFrom = 0
		}

		lines, err := tr.readLinesFromOffset(file, readFrom, atFileStart)
		if err != nil {
			return nil, err
		}

		// atFileStart guarantees termination for files where stop never holds.
		if atFileStart || stop(lines) {
			return lines, nil
		}

		// Clamp to size instead of doubling past it: avoids int64 overflow on
		// pathological sizes (e.g. sparse files) and makes the next pass final.
		if window > size/2 {
			window = size
		} else {
			window *= 2
		}
	}
}

// readLinesFromOffset reads the byte range [offset, EOF) and returns its
// complete lines reversed. When dropPartialFirst is true the first line is
// discarded because the window started mid-line; when the offset is the file
// start (offset == 0 / atFileStart) every line is complete and kept.
func (tr *TranscriptReader) readLinesFromOffset(file *os.File, offset int64, atFileStart bool) ([]string, error) {
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek transcript: %w", err)
	}

	var lines []string
	reader := bufio.NewReader(file)
	first := true

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read transcript: %w", err)
		}

		hadNewline := strings.HasSuffix(line, "\n")
		trimmed := strings.TrimRight(line, "\n\r")

		// A window that does not begin at the file start splits the first line
		// across the window boundary, so that fragment is dropped.
		dropPartial := first && offset > 0 && !atFileStart
		first = false

		// The trailing fragment without a newline at EOF is a complete final
		// line only when EOF terminates it; ReadString returns it with err==EOF.
		if trimmed != "" && !dropPartial {
			if hadNewline || err == io.EOF {
				lines = append(lines, trimmed)
			}
		}

		if err == io.EOF {
			break
		}
	}

	for i := len(lines)/2 - 1; i >= 0; i-- {
		opp := len(lines) - 1 - i
		lines[i], lines[opp] = lines[opp], lines[i]
	}

	return lines, nil
}

// fileSize returns the size of an open file.
func fileSize(file *os.File) (int64, error) {
	info, err := file.Stat()
	if err != nil {
		return 0, fmt.Errorf("failed to stat transcript: %w", err)
	}
	return info.Size(), nil
}

// containsAssistantLine reports whether any line parses as an assistant message.
func containsAssistantLine(lines []string) bool {
	for _, line := range lines {
		var msg TranscriptMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		if msg.Type == "assistant" {
			return true
		}
	}
	return false
}

// containsTwoAssistantUUIDs reports whether the lines hold assistant messages
// with at least two distinct non-empty UUIDs. See readLinesReverse for why
// this guarantees the newest message's fragments are complete in the window.
func containsTwoAssistantUUIDs(lines []string) bool {
	var first string
	for _, line := range lines {
		var msg TranscriptMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		if msg.Type != "assistant" || msg.UUID == "" {
			continue
		}
		if first == "" {
			first = msg.UUID
		} else if msg.UUID != first {
			return true
		}
	}
	return false
}

// ProcessText applies reading mode restrictions to the text
func (tr *TranscriptReader) ProcessText(text string) string {
	// Normalize mode for backward compatibility
	normalizedMode := NormalizeReadingMode(tr.config.ReadingMode)

	switch normalizedMode {
	case ModeShort:
		// First line only (formerly first_line)
		text = strings.TrimSpace(text)
		if idx := strings.Index(text, "\n"); idx != -1 {
			text = strings.TrimSpace(text[:idx])
		}

	case ModeFull:
		// Full text with newlines replaced by spaces (formerly full_text/char_limit)
		text = strings.ReplaceAll(text, "\n", " ")
		// Apply character limit if specified
		if tr.config.MaxChars > 0 {
			runes := []rune(text)
			if len(runes) > tr.config.MaxChars {
				text = string(runes[:tr.config.MaxChars])
			}
		}
	}

	log.Debug().
		Str("mode", normalizedMode).
		Str("original_mode", tr.config.ReadingMode).
		Int("length", len(text)).
		Msg("Processed text")

	return text
}

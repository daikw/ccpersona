package voice

// Config represents voice synthesis configuration
type Config struct {
	// Engine settings
	EnginePriority     string  `json:"engine_priority"`     // "voicevox" or "aivisspeech"
	VoicevoxSpeaker    int     `json:"voicevox_speaker"`    // VOICEVOX speaker ID
	AivisSpeechSpeaker int64   `json:"aivisspeech_speaker"` // AivisSpeech speaker ID
	VolumeScale        float64 `json:"volume_scale"`        // Volume scale (0.0-2.0, default 1.0)
	SpeedScale         float64 `json:"speed_scale"`         // Speed scale (0.5-2.0, default 1.0)

	// Reading settings
	ReadingMode string `json:"reading_mode"` // short (first line) or full (entire text)
	MaxChars    int    `json:"max_chars"`    // Character limit for 'full' mode (0 = unlimited)

	// Processing settings
	UUIDMode bool `json:"uuid_mode"` // Use UUID search mode (slower but complete)
}

// DefaultConfig returns the default voice configuration
func DefaultConfig() *Config {
	return &Config{
		EnginePriority:     "aivisspeech",
		VoicevoxSpeaker:    3,          // ずんだもん
		AivisSpeechSpeaker: 1512153248, // Default AivisSpeech speaker
		VolumeScale:        1.0,        // Default volume
		SpeedScale:         1.0,        // Default speed
		ReadingMode:        "short",    // First line only
		MaxChars:           0,          // No limit by default
		UUIDMode:           false,
	}
}

// TranscriptMessage represents a message in Claude Code transcript
type TranscriptMessage struct {
	Type    string `json:"type"`
	UUID    string `json:"uuid,omitempty"`
	Message struct {
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
		} `json:"content"`
	} `json:"message,omitempty"`
}

// AudioQuery represents the audio query for voice synthesis
type AudioQuery struct {
	Text              string  `json:"text"`
	SpeedScale        float64 `json:"speedScale"`
	PitchScale        float64 `json:"pitchScale"`
	VolumeScale       float64 `json:"volumeScale"`
	PrePhonemeLength  float64 `json:"prePhonemeLength"`
	PostPhonemeLength float64 `json:"postPhonemeLength"`
}

// ReadingMode constants
// Primary modes (recommended):
const (
	ModeShort = "short" // Read first line only
	ModeFull  = "full"  // Read full text (with optional char limit)
)

// Legacy mode aliases for backward compatibility:
const (
	ModeFirstLine  = "first_line"  // Alias for "short"
	ModeLineLimit  = "line_limit"  // Deprecated: use "full" with --chars
	ModeAfterFirst = "after_first" // Deprecated: use "full"
	ModeFullText   = "full_text"   // Alias for "full"
	ModeCharLimit  = "char_limit"  // Alias for "full" with --chars
)

// NormalizeReadingMode converts legacy mode names to canonical names
func NormalizeReadingMode(mode string) string {
	switch mode {
	case ModeFirstLine, ModeShort:
		return ModeShort
	case ModeFullText, ModeFull, ModeLineLimit, ModeAfterFirst, ModeCharLimit:
		return ModeFull
	default:
		return ModeShort // Default
	}
}

// Engine constants
const (
	EngineVoicevox    = "voicevox"
	EngineAivisSpeech = "aivisspeech"
)

// Default engine URLs
const (
	VoicevoxURL    = "http://127.0.0.1:50021"
	AivisSpeechURL = "http://127.0.0.1:10101"
)

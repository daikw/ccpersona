package voice

// Config represents voice synthesis configuration
type Config struct {
	// Engine settings
	EnginePriority     string `json:"engine_priority"`     // "voicevox" or "aivisspeech"
	VoicevoxSpeaker    int    `json:"voicevox_speaker"`    // VOICEVOX speaker ID
	AivisSpeechSpeaker int64  `json:"aivisspeech_speaker"` // AivisSpeech speaker ID

	// Reading settings
	ReadingMode string `json:"reading_mode"` // first_line, line_limit, after_first, full_text, char_limit
	MaxChars    int    `json:"max_chars"`    // Character limit for char_limit mode
	MaxLines    int    `json:"max_lines"`    // Line limit for line_limit mode

	// Processing settings
	UUIDMode bool `json:"uuid_mode"` // Use UUID search mode (slower but complete)
}

// DefaultConfig returns the default voice configuration
func DefaultConfig() *Config {
	return &Config{
		EnginePriority:     "aivisspeech",
		VoicevoxSpeaker:    3,          // ずんだもん
		AivisSpeechSpeaker: 1512153248, // Default AivisSpeech speaker
		ReadingMode:        "first_line",
		MaxChars:           500,
		MaxLines:           3,
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
const (
	ModeFirstLine  = "first_line"
	ModeLineLimit  = "line_limit"
	ModeAfterFirst = "after_first"
	ModeFullText   = "full_text"
	ModeCharLimit  = "char_limit"
)

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

package provider

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/polly"
	"github.com/aws/aws-sdk-go-v2/service/polly/types"
	"github.com/rs/zerolog/log"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// PollyClient interface defines the methods we need from the Polly client
type PollyClient interface {
	DescribeVoices(ctx context.Context, params *polly.DescribeVoicesInput, optFns ...func(*polly.Options)) (*polly.DescribeVoicesOutput, error)
	SynthesizeSpeech(ctx context.Context, params *polly.SynthesizeSpeechInput, optFns ...func(*polly.Options)) (*polly.SynthesizeSpeechOutput, error)
}

// PollyProvider implements the Provider interface for Amazon Polly
type PollyProvider struct {
	client PollyClient
	region string
}

// NewPollyProvider creates a new Amazon Polly TTS provider
func NewPollyProvider(region string) (*PollyProvider, error) {
	if region == "" {
		region = "us-east-1" // Default region
	}

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Polly client
	client := polly.NewFromConfig(cfg)

	return &PollyProvider{
		client: client,
		region: region,
	}, nil
}

// Name returns the provider name
func (p *PollyProvider) Name() string {
	return "polly"
}

// ListVoices returns available Amazon Polly voices
func (p *PollyProvider) ListVoices(ctx context.Context) ([]Voice, error) {
	input := &polly.DescribeVoicesInput{}

	result, err := p.client.DescribeVoices(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list Polly voices: %w", err)
	}

	voices := make([]Voice, 0, len(result.Voices))
	for _, v := range result.Voices {
		voice := Voice{
			ID:       string(v.Id),
			Name:     aws.ToString(v.Name),
			Language: string(v.LanguageCode),
			Description: fmt.Sprintf("%s voice, %s engine supported",
				cases.Title(language.English).String(string(v.Gender)),
				formatSupportedEngines(v.SupportedEngines)),
		}

		// Set gender from Polly gender enum
		switch v.Gender {
		case types.GenderFemale:
			voice.Gender = "female"
		case types.GenderMale:
			voice.Gender = "male"
		}

		voices = append(voices, voice)
	}

	return voices, nil
}

// Synthesize generates audio from text using Amazon Polly
func (p *PollyProvider) Synthesize(ctx context.Context, text string, options SynthesizeOptions) (io.ReadCloser, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Set defaults
	voiceID := options.Voice
	if voiceID == "" {
		voiceID = "Joanna" // Default voice
	}

	outputFormat := options.Format
	if outputFormat == "" {
		outputFormat = "mp3"
	}

	// Convert format to Polly format
	var pollyFormat types.OutputFormat
	switch strings.ToLower(outputFormat) {
	case "mp3":
		pollyFormat = types.OutputFormatMp3
	case "ogg":
		pollyFormat = types.OutputFormatOggVorbis
	case "pcm":
		pollyFormat = types.OutputFormatPcm
	default:
		return nil, fmt.Errorf("unsupported audio format: %s", outputFormat)
	}

	// Parse engine from dedicated engine field or use neural by default
	engine := types.EngineNeural
	if options.Engine != "" {
		switch strings.ToLower(options.Engine) {
		case "standard":
			engine = types.EngineStandard
		case "neural":
			engine = types.EngineNeural
		case "long-form":
			engine = types.EngineLongForm
		case "generative":
			engine = types.EngineGenerative
		default:
			log.Warn().Str("engine", options.Engine).Msg("Unknown engine, using neural")
		}
	}

	// Prepare synthesis input
	input := &polly.SynthesizeSpeechInput{
		Text:         aws.String(text),
		VoiceId:      types.VoiceId(voiceID),
		OutputFormat: pollyFormat,
		Engine:       engine,
	}

	// Set sample rate if specified in dedicated sample rate field
	if options.SampleRate != "" {
		sampleRate := options.SampleRate
		switch sampleRate {
		case "8000", "16000", "22050", "24000":
			input.SampleRate = aws.String(sampleRate)
		default:
			log.Warn().Str("sample_rate", sampleRate).Msg("Invalid sample rate, using default")
		}
	}

	// Check if text contains SSML tags
	if strings.Contains(text, "<speak>") || strings.Contains(text, "<prosody") {
		input.TextType = types.TextTypeSsml
	} else {
		input.TextType = types.TextTypeText
	}

	log.Debug().
		Str("voice_id", voiceID).
		Str("output_format", string(pollyFormat)).
		Str("engine", string(engine)).
		Str("text_type", string(input.TextType)).
		Msg("Making Polly synthesis request")

	// Make synthesis request
	result, err := p.client.SynthesizeSpeech(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to synthesize speech: %w", err)
	}

	log.Debug().
		Str("content_type", aws.ToString(result.ContentType)).
		Msg("Polly synthesis request successful")

	return result.AudioStream, nil
}

// IsAvailable checks if Amazon Polly provider is available
func (p *PollyProvider) IsAvailable(ctx context.Context) bool {
	// Create a context with timeout for availability check
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Try to list voices to check if service is available
	input := &polly.DescribeVoicesInput{}

	_, err := p.client.DescribeVoices(checkCtx, input)
	return err == nil
}

// PollyProviderFromConfig creates a Polly provider from configuration
func PollyProviderFromConfig(config map[string]interface{}) (*PollyProvider, error) {
	region := "us-east-1" // Default region

	// Get region from config
	if r, ok := config["region"].(string); ok && r != "" {
		region = r
	}

	return NewPollyProvider(region)
}

// formatSupportedEngines formats the list of supported engines for display
func formatSupportedEngines(engines []types.Engine) string {
	if len(engines) == 0 {
		return "unknown"
	}

	engineNames := make([]string, len(engines))
	for i, engine := range engines {
		engineNames[i] = string(engine)
	}

	return strings.Join(engineNames, ", ")
}

// GetPollyVoicesByLanguage returns Polly voices filtered by language
func (p *PollyProvider) GetPollyVoicesByLanguage(ctx context.Context, languageCode string) ([]Voice, error) {
	input := &polly.DescribeVoicesInput{
		LanguageCode: types.LanguageCode(languageCode),
	}

	result, err := p.client.DescribeVoices(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list Polly voices for language %s: %w", languageCode, err)
	}

	voices := make([]Voice, 0, len(result.Voices))
	for _, v := range result.Voices {
		voice := Voice{
			ID:       string(v.Id),
			Name:     aws.ToString(v.Name),
			Language: string(v.LanguageCode),
			Description: fmt.Sprintf("%s voice, %s engine supported",
				cases.Title(language.English).String(string(v.Gender)),
				formatSupportedEngines(v.SupportedEngines)),
		}

		// Set gender from Polly gender enum
		switch v.Gender {
		case types.GenderFemale:
			voice.Gender = "female"
		case types.GenderMale:
			voice.Gender = "male"
		}

		voices = append(voices, voice)
	}

	return voices, nil
}
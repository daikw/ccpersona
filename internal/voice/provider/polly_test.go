package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/polly"
	"github.com/aws/aws-sdk-go-v2/service/polly/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPollyClient is a mock implementation of the Polly API client
type MockPollyClient struct {
	mock.Mock
}

func (m *MockPollyClient) DescribeVoices(ctx context.Context, params *polly.DescribeVoicesInput, optFns ...func(*polly.Options)) (*polly.DescribeVoicesOutput, error) {
	args := m.Called(ctx, params)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.(*polly.DescribeVoicesOutput), args.Error(1)
}

func (m *MockPollyClient) SynthesizeSpeech(ctx context.Context, params *polly.SynthesizeSpeechInput, optFns ...func(*polly.Options)) (*polly.SynthesizeSpeechOutput, error) {
	args := m.Called(ctx, params)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.(*polly.SynthesizeSpeechOutput), args.Error(1)
}

// MockReadCloser is a mock implementation of io.ReadCloser
type MockReadCloser struct {
	data   []byte
	pos    int
	closed bool
}

func NewMockReadCloser(data []byte) *MockReadCloser {
	return &MockReadCloser{data: data, pos: 0, closed: false}
}

func (m *MockReadCloser) Read(p []byte) (n int, err error) {
	if m.closed {
		return 0, errors.New("reader closed")
	}
	if m.pos >= len(m.data) {
		return 0, io.EOF
	}
	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func (m *MockReadCloser) Close() error {
	m.closed = true
	return nil
}

func TestPollyProvider_Name(t *testing.T) {
	provider := &PollyProvider{}
	assert.Equal(t, "polly", provider.Name())
}

func TestPollyProvider_ListVoices(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   *polly.DescribeVoicesOutput
		mockError      error
		expectedVoices []Voice
		expectedError  string
	}{
		{
			name: "successful voice listing",
			mockResponse: &polly.DescribeVoicesOutput{
				Voices: []types.Voice{
					{
						Id:               types.VoiceId("Joanna"),
						Name:             aws.String("Joanna"),
						LanguageCode:     types.LanguageCode("en-US"),
						Gender:           types.GenderFemale,
						SupportedEngines: []types.Engine{types.EngineNeural, types.EngineStandard},
					},
					{
						Id:               types.VoiceId("Matthew"),
						Name:             aws.String("Matthew"),
						LanguageCode:     types.LanguageCode("en-US"),  
						Gender:           types.GenderMale,
						SupportedEngines: []types.Engine{types.EngineNeural},
					},
				},
			},
			expectedVoices: []Voice{
				{
					ID:          "Joanna",
					Name:        "Joanna",
					Language:    "en-US",
					Gender:      "female",
					Description: "Female voice, neural, standard engine supported",
				},
				{
					ID:          "Matthew",
					Name:        "Matthew",
					Language:    "en-US",
					Gender:      "male",
					Description: "Male voice, neural engine supported",
				},
			},
		},
		{
			name:          "API error",
			mockError:     errors.New("API error"),
			expectedError: "failed to list Polly voices: API error",
		},
		{
			name: "empty voice list",
			mockResponse: &polly.DescribeVoicesOutput{
				Voices: []types.Voice{},
			},
			expectedVoices: []Voice{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPollyClient{}
			provider := &PollyProvider{client: mockClient, region: "us-east-1"}

			mockClient.On("DescribeVoices", mock.Anything, mock.Anything).Return(tt.mockResponse, tt.mockError)

			voices, err := provider.ListVoices(context.Background())

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedVoices, voices)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestPollyProvider_Synthesize(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		options        SynthesizeOptions
		mockResponse   *polly.SynthesizeSpeechOutput
		mockError      error
		expectedError  string
		validateInput  func(*testing.T, *polly.SynthesizeSpeechInput)
	}{
		{
			name: "successful synthesis with defaults",
			text: "Hello world",
			options: SynthesizeOptions{},
			mockResponse: &polly.SynthesizeSpeechOutput{
				AudioStream: NewMockReadCloser([]byte("mock audio data")),
				ContentType: aws.String("audio/mpeg"),
			},
			validateInput: func(t *testing.T, input *polly.SynthesizeSpeechInput) {
				assert.Equal(t, "Hello world", *input.Text)
				assert.Equal(t, types.VoiceId("Joanna"), input.VoiceId)
				assert.Equal(t, types.OutputFormatMp3, input.OutputFormat)
				assert.Equal(t, types.EngineNeural, input.Engine)
				assert.Equal(t, types.TextTypeText, input.TextType)
			},
		},
		{
			name: "synthesis with custom options",
			text: "Custom text",
			options: SynthesizeOptions{
				Voice:      "Emma",
				Format:     "ogg",
				Engine:     "standard",
				SampleRate: "16000",
			},
			mockResponse: &polly.SynthesizeSpeechOutput{
				AudioStream: NewMockReadCloser([]byte("mock audio data")),
				ContentType: aws.String("audio/ogg"),
			},
			validateInput: func(t *testing.T, input *polly.SynthesizeSpeechInput) {
				assert.Equal(t, "Custom text", *input.Text)
				assert.Equal(t, types.VoiceId("Emma"), input.VoiceId)
				assert.Equal(t, types.OutputFormatOggVorbis, input.OutputFormat)
				assert.Equal(t, types.EngineStandard, input.Engine)
				assert.Equal(t, "16000", *input.SampleRate)
			},
		},
		{
			name: "synthesis with SSML",
			text: "<speak>Hello <prosody rate='slow'>world</prosody></speak>",
			options: SynthesizeOptions{},
			mockResponse: &polly.SynthesizeSpeechOutput{
				AudioStream: NewMockReadCloser([]byte("mock audio data")),
				ContentType: aws.String("audio/mpeg"),
			},
			validateInput: func(t *testing.T, input *polly.SynthesizeSpeechInput) {
				assert.Equal(t, types.TextTypeSsml, input.TextType)
			},
		},
		{
			name: "synthesis with all engines",
			text: "Test",
			options: SynthesizeOptions{Engine: "generative"},
			mockResponse: &polly.SynthesizeSpeechOutput{
				AudioStream: NewMockReadCloser([]byte("mock audio data")),
				ContentType: aws.String("audio/mpeg"),
			},
			validateInput: func(t *testing.T, input *polly.SynthesizeSpeechInput) {
				assert.Equal(t, types.EngineGenerative, input.Engine)
			},
		},
		{
			name:          "empty text error",
			text:          "",
			options:       SynthesizeOptions{},
			expectedError: "text cannot be empty",
		},
		{
			name: "unsupported format error",
			text: "Hello",
			options: SynthesizeOptions{Format: "unsupported"},
			expectedError: "unsupported audio format: unsupported",
		},
		{
			name: "API synthesis error",
			text: "Hello",
			options: SynthesizeOptions{},
			mockError: errors.New("synthesis failed"),
			expectedError: "failed to synthesize speech: synthesis failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPollyClient{}
			provider := &PollyProvider{client: mockClient, region: "us-east-1"}

			if tt.expectedError == "" || tt.mockError != nil {
				mockClient.On("SynthesizeSpeech", mock.Anything, mock.MatchedBy(func(input *polly.SynthesizeSpeechInput) bool {
					if tt.validateInput != nil {
						tt.validateInput(t, input)
					}
					return true
				})).Return(tt.mockResponse, tt.mockError)
			}

			result, err := provider.Synthesize(context.Background(), tt.text, tt.options)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				
				// Test that we can read from the result
				data, readErr := io.ReadAll(result)
				assert.NoError(t, readErr)
				assert.Equal(t, []byte("mock audio data"), data)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestPollyProvider_Synthesize_EngineValidation(t *testing.T) {
	tests := []struct {
		engine   string
		expected types.Engine
	}{
		{"neural", types.EngineNeural},
		{"standard", types.EngineStandard},
		{"long-form", types.EngineLongForm},
		{"generative", types.EngineGenerative},
		{"NEURAL", types.EngineNeural}, // Case insensitive
		{"invalid", types.EngineNeural}, // Default fallback
		{"", types.EngineNeural},        // Empty fallback
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("engine_%s", tt.engine), func(t *testing.T) {
			mockClient := &MockPollyClient{}
			provider := &PollyProvider{client: mockClient, region: "us-east-1"}

			mockClient.On("SynthesizeSpeech", mock.Anything, mock.MatchedBy(func(input *polly.SynthesizeSpeechInput) bool {
				assert.Equal(t, tt.expected, input.Engine)
				return true
			})).Return(&polly.SynthesizeSpeechOutput{
				AudioStream: NewMockReadCloser([]byte("test")),
				ContentType: aws.String("audio/mpeg"),
			}, nil)

			_, err := provider.Synthesize(context.Background(), "test", SynthesizeOptions{Engine: tt.engine})
			assert.NoError(t, err)
			mockClient.AssertExpectations(t)
		})
	}
}

func TestPollyProvider_Synthesize_SampleRateValidation(t *testing.T) {
	tests := []struct {
		sampleRate string
		shouldSet  bool
	}{
		{"8000", true},
		{"16000", true},
		{"22050", true},
		{"24000", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("sample_rate_%s", tt.sampleRate), func(t *testing.T) {
			mockClient := &MockPollyClient{}
			provider := &PollyProvider{client: mockClient, region: "us-east-1"}

			mockClient.On("SynthesizeSpeech", mock.Anything, mock.MatchedBy(func(input *polly.SynthesizeSpeechInput) bool {
				if tt.shouldSet {
					assert.NotNil(t, input.SampleRate)
					assert.Equal(t, tt.sampleRate, *input.SampleRate)
				} else {
					assert.Nil(t, input.SampleRate)
				}
				return true
			})).Return(&polly.SynthesizeSpeechOutput{
				AudioStream: NewMockReadCloser([]byte("test")),
				ContentType: aws.String("audio/mpeg"),
			}, nil)

			_, err := provider.Synthesize(context.Background(), "test", SynthesizeOptions{SampleRate: tt.sampleRate})
			assert.NoError(t, err)
			mockClient.AssertExpectations(t)
		})
	}
}

func TestPollyProvider_Synthesize_FormatValidation(t *testing.T) {
	tests := []struct {
		format   string
		expected types.OutputFormat
	}{
		{"mp3", types.OutputFormatMp3},
		{"ogg", types.OutputFormatOggVorbis},
		{"pcm", types.OutputFormatPcm},
		{"MP3", types.OutputFormatMp3}, // Case insensitive
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("format_%s", tt.format), func(t *testing.T) {
			mockClient := &MockPollyClient{}
			provider := &PollyProvider{client: mockClient, region: "us-east-1"}

			mockClient.On("SynthesizeSpeech", mock.Anything, mock.MatchedBy(func(input *polly.SynthesizeSpeechInput) bool {
				assert.Equal(t, tt.expected, input.OutputFormat)
				return true
			})).Return(&polly.SynthesizeSpeechOutput{
				AudioStream: NewMockReadCloser([]byte("test")),
				ContentType: aws.String("audio/mpeg"),
			}, nil)

			_, err := provider.Synthesize(context.Background(), "test", SynthesizeOptions{Format: tt.format})
			assert.NoError(t, err)
			mockClient.AssertExpectations(t)
		})
	}
}

func TestPollyProvider_Synthesize_SSMLDetection(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected types.TextType
	}{
		{"plain text", "Hello world", types.TextTypeText},
		{"SSML with speak tag", "<speak>Hello world</speak>", types.TextTypeSsml},
		{"SSML with prosody tag", "Hello <prosody rate='slow'>world</prosody>", types.TextTypeSsml},
		{"text with angle brackets but not SSML", "Hello <not-ssml> world", types.TextTypeText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPollyClient{}
			provider := &PollyProvider{client: mockClient, region: "us-east-1"}

			mockClient.On("SynthesizeSpeech", mock.Anything, mock.MatchedBy(func(input *polly.SynthesizeSpeechInput) bool {
				assert.Equal(t, tt.expected, input.TextType)
				return true
			})).Return(&polly.SynthesizeSpeechOutput{
				AudioStream: NewMockReadCloser([]byte("test")),
				ContentType: aws.String("audio/mpeg"),
			}, nil)

			_, err := provider.Synthesize(context.Background(), tt.text, SynthesizeOptions{})
			assert.NoError(t, err)
			mockClient.AssertExpectations(t)
		})
	}
}

func TestPollyProvider_IsAvailable(t *testing.T) {
	tests := []struct {
		name          string
		mockResponse  *polly.DescribeVoicesOutput
		mockError     error
		expectedAvail bool
	}{
		{
			name: "service available",
			mockResponse: &polly.DescribeVoicesOutput{
				Voices: []types.Voice{
					{Id: types.VoiceId("test")},
				},
			},
			expectedAvail: true,
		},
		{
			name:          "service unavailable",
			mockError:     errors.New("network error"),
			expectedAvail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPollyClient{}
			provider := &PollyProvider{client: mockClient, region: "us-east-1"}

			mockClient.On("DescribeVoices", mock.Anything, mock.Anything).Return(tt.mockResponse, tt.mockError)

			available := provider.IsAvailable(context.Background())
			assert.Equal(t, tt.expectedAvail, available)
			mockClient.AssertExpectations(t)
		})
	}
}

func TestPollyProvider_GetPollyVoicesByLanguage(t *testing.T) {
	tests := []struct {
		name           string
		languageCode   string
		mockResponse   *polly.DescribeVoicesOutput
		mockError      error
		expectedVoices []Voice
		expectedError  string
	}{
		{
			name:         "successful language-specific listing",
			languageCode: "en-US",
			mockResponse: &polly.DescribeVoicesOutput{
				Voices: []types.Voice{
					{
						Id:               types.VoiceId("Joanna"),
						Name:             aws.String("Joanna"),
						LanguageCode:     types.LanguageCode("en-US"),
						Gender:           types.GenderFemale,
						SupportedEngines: []types.Engine{types.EngineNeural},
					},
				},
			},
			expectedVoices: []Voice{
				{
					ID:          "Joanna",
					Name:        "Joanna",
					Language:    "en-US",
					Gender:      "female",
					Description: "Female voice, neural engine supported",
				},
			},
		},
		{
			name:          "API error",
			languageCode:  "en-US",
			mockError:     errors.New("API error"),
			expectedError: "failed to list Polly voices for language en-US: API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPollyClient{}
			provider := &PollyProvider{client: mockClient, region: "us-east-1"}

			mockClient.On("DescribeVoices", mock.Anything, mock.MatchedBy(func(input *polly.DescribeVoicesInput) bool {
				assert.Equal(t, types.LanguageCode(tt.languageCode), input.LanguageCode)
				return true
			})).Return(tt.mockResponse, tt.mockError)

			voices, err := provider.GetPollyVoicesByLanguage(context.Background(), tt.languageCode)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedVoices, voices)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestPollyProviderFromConfig(t *testing.T) {
	tests := []struct {
		name           string
		config         map[string]interface{}
		expectedRegion string
	}{
		{
			name:           "default region",
			config:         map[string]interface{}{},
			expectedRegion: "us-east-1",
		},
		{
			name: "custom region",
			config: map[string]interface{}{
				"region": "eu-west-1",
			},
			expectedRegion: "eu-west-1",
		},
		{
			name: "invalid region type",
			config: map[string]interface{}{
				"region": 123, // Invalid type
			},
			expectedRegion: "us-east-1", // Should fall back to default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := PollyProviderFromConfig(tt.config)
			
			// We can't easily test the actual AWS client creation without mocking more,
			// but we can verify the function doesn't crash and returns a provider
			if strings.Contains(tt.name, "invalid") {
				// For invalid configs, we expect it to still work with defaults
				assert.NoError(t, err)
				assert.NotNil(t, provider)
			} else {
				assert.NotNil(t, provider)
				assert.Equal(t, tt.expectedRegion, provider.region)
			}
		})
	}
}

func TestFormatSupportedEngines(t *testing.T) {
	tests := []struct {
		name     string
		engines  []types.Engine
		expected string
	}{
		{
			name:     "empty engines",
			engines:  []types.Engine{},
			expected: "unknown",
		},
		{
			name:     "single engine",
			engines:  []types.Engine{types.EngineNeural},
			expected: "neural",
		},
		{
			name:     "multiple engines",
			engines:  []types.Engine{types.EngineNeural, types.EngineStandard},
			expected: "neural, standard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSupportedEngines(tt.engines)
			assert.Equal(t, tt.expected, result)
		})
	}
}
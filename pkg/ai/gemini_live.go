package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
)

const (
	DefaultGeminiLiveURL   = "wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent"
	DefaultGeminiLiveModel = "gemini-2.0-flash-exp"
	DefaultGeminiVoice     = "Puck"
)

// Gemini Live protocol structures

type PrebuiltVoiceConfig struct {
	VoiceName string `json:"voiceName"` // e.g. "Puck", "Charon", "Aoede"
}

type VoiceConfig struct {
	PrebuiltVoiceConfig PrebuiltVoiceConfig `json:"prebuiltVoiceConfig"`
}

type SpeechConfig struct {
	VoiceConfig VoiceConfig `json:"voiceConfig"`
}

type GenerationConfig struct {
	ResponseModalities []string      `json:"responseModalities"` // ["AUDIO"]
	SpeechConfig       *SpeechConfig `json:"speechConfig,omitempty"`
}

type ContentPart struct {
	Text       string      `json:"text,omitempty"`
	InlineData *InlineData `json:"inlineData,omitempty"`
}

type InlineData struct {
	MimeType string `json:"mimeType"` // e.g. "audio/pcm;rate=24000"
	Data     string `json:"data"`     // Base64 PCM data
}

type Content struct {
	Parts []ContentPart `json:"parts"`
	Role  string        `json:"role,omitempty"`
}

type BidiGenerateContentSetup struct {
	Model             string            `json:"model"`
	GenerationConfig  *GenerationConfig `json:"generationConfig,omitempty"`
	SystemInstruction *Content          `json:"systemInstruction,omitempty"`
}

type SetupMessage struct {
	Setup BidiGenerateContentSetup `json:"setup"`
}

type RealtimeMediaChunk struct {
	MimeType string `json:"mime_type"` // e.g. "audio/pcm;rate=16000"
	Data     string `json:"data"`      // Base64 PCM
}

type RealtimeBlob struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type RealtimeInputPayload struct {
	MediaChunks []RealtimeMediaChunk `json:"media_chunks,omitempty"`
	Audio       *RealtimeBlob        `json:"audio,omitempty"`
	Text        string               `json:"text,omitempty"`
	AudioEnd    bool                 `json:"audioStreamEnd,omitempty"`
}

type RealtimeInputMessage struct {
	RealtimeInput RealtimeInputPayload `json:"realtime_input"`
}

// Support both standard camelCase and snake_case for maximum compatibility
type RealtimeInputCamelMessage struct {
	RealtimeInput RealtimeInputPayload `json:"realtimeInput"`
}

type Transcription struct {
	Text string `json:"text"`
}

type ModelTurn struct {
	Parts []ContentPart `json:"parts"`
}

type ServerContent struct {
	ModelTurn           *ModelTurn     `json:"modelTurn,omitempty"`
	Interrupted         bool           `json:"interrupted,omitempty"`
	TurnComplete        bool           `json:"turnComplete,omitempty"`
	InputTranscription  *Transcription `json:"inputTranscription,omitempty"`
	OutputTranscription *Transcription `json:"outputTranscription,omitempty"`
}

type BidiServerMessage struct {
	ServerContent *ServerContent `json:"serverContent,omitempty"`
}

// LiveAudioChunk represents an audio chunk received from the live AI engine.
type LiveAudioChunk struct {
	PCMData      []byte
	SampleRate   int
	IsTurnEnd    bool
	Interrupted  bool
	TranscriptIn string
	TranscriptOut string
}

// GeminiLiveConfig configures the live WebSocket connection.
type GeminiLiveConfig struct {
	BaseURL           string
	APIKey            string
	Model             string
	VoiceName         string
	SystemInstruction string
	InputSampleRate   int // 16000 or 24000
}

// GeminiLiveClient manages a bidirectional WebSocket session with Google Gemini Live API.
type GeminiLiveClient struct {
	cfg        GeminiLiveConfig
	conn       *websocket.Conn
	mu         sync.Mutex
	audioOutCh chan LiveAudioChunk
	errCh      chan error
	doneCh     chan struct{}
	closed     bool
}

// NewGeminiLiveClient creates a new client instance.
func NewGeminiLiveClient(cfg GeminiLiveConfig) *GeminiLiveClient {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultGeminiLiveURL
	}
	if cfg.Model == "" {
		cfg.Model = DefaultGeminiLiveModel
	}
	if cfg.VoiceName == "" {
		cfg.VoiceName = DefaultGeminiVoice
	}
	if cfg.InputSampleRate <= 0 {
		cfg.InputSampleRate = 16000
	}

	return &GeminiLiveClient{
		cfg:        cfg,
		audioOutCh: make(chan LiveAudioChunk, 256),
		errCh:      make(chan error, 16),
		doneCh:     make(chan struct{}),
	}
}

// SetConnection sets an existing or mock WebSocket connection (ideal for testing).
func (c *GeminiLiveClient) SetConnection(conn *websocket.Conn) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.conn = conn
}

// Connect dials the Gemini Live WebSocket endpoint and sends the setup message.
func (c *GeminiLiveClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil
	}

	url := fmt.Sprintf("%s?key=%s", c.cfg.BaseURL, c.cfg.APIKey)
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Gemini Live API WebSocket: %w", err)
	}
	c.conn = conn

	// Send initial setup frame
	modelName := c.cfg.Model
	if len(modelName) > 0 && modelName[:7] != "models/" {
		modelName = "models/" + modelName
	}

	setupMsg := SetupMessage{
		Setup: BidiGenerateContentSetup{
			Model: modelName,
			GenerationConfig: &GenerationConfig{
				ResponseModalities: []string{"AUDIO"},
				SpeechConfig: &SpeechConfig{
					VoiceConfig: VoiceConfig{
						PrebuiltVoiceConfig: PrebuiltVoiceConfig{
							VoiceName: c.cfg.VoiceName,
						},
					},
				},
			},
		},
	}

	if c.cfg.SystemInstruction != "" {
		setupMsg.Setup.SystemInstruction = &Content{
			Parts: []ContentPart{
				{Text: c.cfg.SystemInstruction},
			},
		}
	}

	setupBytes, err := json.Marshal(setupMsg)
	if err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to marshal setup message: %w", err)
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, setupBytes); err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to send setup message: %w", err)
	}

	return nil
}

// StartReadLoop begins reading messages from Gemini Live in the background.
func (c *GeminiLiveClient) StartReadLoop() {
	go c.readPump()
}

// SendRealtimeAudio sends a PCM audio chunk to Gemini Live API.
func (c *GeminiLiveClient) SendRealtimeAudio(pcm16k []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || c.conn == nil {
		return errors.New("gemini live connection closed or uninitialized")
	}

	b64Audio := base64.StdEncoding.EncodeToString(pcm16k)
	mimeType := fmt.Sprintf("audio/pcm;rate=%d", c.cfg.InputSampleRate)

	msg := RealtimeInputMessage{
		RealtimeInput: RealtimeInputPayload{
			MediaChunks: []RealtimeMediaChunk{
				{
					MimeType: mimeType,
					Data:     b64Audio,
				},
			},
		},
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal realtime audio payload: %w", err)
	}

	return c.conn.WriteMessage(websocket.TextMessage, payload)
}

// SendText sends text input into the live session.
func (c *GeminiLiveClient) SendText(text string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || c.conn == nil {
		return errors.New("gemini live connection closed or uninitialized")
	}

	msg := RealtimeInputMessage{
		RealtimeInput: RealtimeInputPayload{
			Text: text,
		},
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal text payload: %w", err)
	}

	return c.conn.WriteMessage(websocket.TextMessage, payload)
}

// AudioOutChan returns a channel providing incoming audio chunks from Gemini.
func (c *GeminiLiveClient) AudioOutChan() <-chan LiveAudioChunk {
	return c.audioOutCh
}

// Errors returns the error notification channel.
func (c *GeminiLiveClient) Errors() <-chan error {
	return c.errCh
}

// readPump receives and parses Gemini Live WebSocket server messages.
func (c *GeminiLiveClient) readPump() {
	defer func() {
		c.Close()
	}()

	for {
		c.mu.Lock()
		conn := c.conn
		closed := c.closed
		c.mu.Unlock()

		if closed || conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			select {
			case <-c.doneCh:
				return
			default:
				select {
				case c.errCh <- err:
				default:
				}
				return
			}
		}

		var serverMsg BidiServerMessage
		if err := json.Unmarshal(message, &serverMsg); err != nil {
			log.Printf("[GeminiLive] Warning: could not parse server message: %v", err)
			continue
		}

		if serverMsg.ServerContent == nil {
			continue
		}

		sc := serverMsg.ServerContent

		// Check for server-side interruption detection
		if sc.Interrupted {
			chunk := LiveAudioChunk{
				Interrupted: true,
			}
			select {
			case c.audioOutCh <- chunk:
			default:
			}
			continue
		}

		var inTranscript, outTranscript string
		if sc.InputTranscription != nil {
			inTranscript = sc.InputTranscription.Text
		}
		if sc.OutputTranscription != nil {
			outTranscript = sc.OutputTranscription.Text
		}

		// Extract audio parts
		if sc.ModelTurn != nil {
			for _, part := range sc.ModelTurn.Parts {
				if part.InlineData != nil && len(part.InlineData.Data) > 0 {
					rawPCM, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
					if err != nil {
						log.Printf("[GeminiLive] Base64 decode audio chunk error: %v", err)
						continue
					}

					chunk := LiveAudioChunk{
						PCMData:       rawPCM,
						SampleRate:    24000, // Gemini native output is 24kHz
						IsTurnEnd:     sc.TurnComplete,
						TranscriptIn:  inTranscript,
						TranscriptOut: outTranscript,
					}

					select {
					case c.audioOutCh <- chunk:
					default:
					}
				}
			}
		} else if sc.TurnComplete {
			chunk := LiveAudioChunk{
				IsTurnEnd:     true,
				TranscriptIn:  inTranscript,
				TranscriptOut: outTranscript,
			}
			select {
			case c.audioOutCh <- chunk:
			default:
			}
		}
	}
}

// Close closes the WebSocket connection and releases resources.
func (c *GeminiLiveClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true
	close(c.doneCh)

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

package ai

import (
	"context"
	"math"
	"sync"
)

// DefaultFillerPhrases provides Turkish conversational filler phrases for instant injection.
var DefaultFillerPhrases = []string{
	"Ha tamam bir saniye bakıyorum...",
	"Hemen sistemden kontrol ediyorum...",
	"Anladım, bir dakika bakayım...",
	"Hemen bakıyorum efendim...",
}

// STTService defines an interface for real-time speech-to-text (e.g. Deepgram).
type STTService interface {
	ProcessAudio(ctx context.Context, chunk []byte) (transcript string, isFinal bool, err error)
}

// LLMService defines an interface for LLM response generation.
type LLMService interface {
	GenerateResponse(ctx context.Context, prompt string, persona string) (string, error)
}

// TTSService defines an interface for text-to-speech synthesis (e.g. Cartesia / ElevenLabs).
type TTSService interface {
	Synthesize(ctx context.Context, text string, voiceID string) ([]byte, int, error)
}

// MockSimpleTTS generates synthetic 16-bit PCM audio (tone/carrier) for testing or fallback.
func GenerateSyntheticSpeechPCM(text string, sampleRate int) []byte {
	// Generate ~30ms of audio per word or minimum 200ms
	words := len(text) / 4
	if words < 4 {
		words = 4
	}
	durationSec := float64(words) * 0.12
	totalSamples := int(float64(sampleRate) * durationSec)

	pcm := make([]byte, totalSamples*2)
	freq := 220.0 // 220 Hz base vocal carrier

	for i := 0; i < totalSamples; i++ {
		t := float64(i) / float64(sampleRate)
		// Speech-like amplitude modulated tone
		val := math.Sin(2*math.Pi*freq*t) * math.Sin(2*math.Pi*4*t) * 8000.0
		sample := int16(val)
		pcm[i*2] = byte(sample)
		pcm[i*2+1] = byte(sample >> 8)
	}
	return pcm
}

// DefaultMockTTS provides fallback synthetic TTS when external provider is not configured.
type DefaultMockTTS struct {
	SampleRate int
}

func (m *DefaultMockTTS) Synthesize(ctx context.Context, text string, voiceID string) ([]byte, int, error) {
	rate := m.SampleRate
	if rate <= 0 {
		rate = 16000
	}
	return GenerateSyntheticSpeechPCM(text, rate), rate, nil
}

// ModularStreamerConfig configures the fallback modular pipeline.
type ModularStreamerConfig struct {
	PersonaName   string
	PersonaPrompt string
	VoiceID       string
	FillerPhrases []string
	SampleRate    int // default 16000
}

// ModularStreamer orchestrates Deepgram STT -> Instant Filler Injection -> LLM -> Cartesia/ElevenLabs TTS.
type ModularStreamer struct {
	cfg        ModularStreamerConfig
	stt        STTService
	llm        LLMService
	tts        TTSService
	audioOutCh chan LiveAudioChunk
	errCh      chan error

	mu          sync.Mutex
	ctx         context.Context
	cancel      context.CancelFunc
	vad         *VADEngine
	closed      bool
	fillerIdx   int
	isInferring bool
}

// NewModularStreamer initializes a modular streaming fallback pipeline.
func NewModularStreamer(cfg ModularStreamerConfig, stt STTService, llm LLMService, tts TTSService) *ModularStreamer {
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = 16000
	}
	if len(cfg.FillerPhrases) == 0 {
		cfg.FillerPhrases = DefaultFillerPhrases
	}
	if tts == nil {
		tts = &DefaultMockTTS{SampleRate: cfg.SampleRate}
	}

	vadCfg := DefaultVADConfig()
	vadCfg.SampleRate = cfg.SampleRate

	ctx, cancel := context.WithCancel(context.Background())

	return &ModularStreamer{
		cfg:        cfg,
		stt:        stt,
		llm:        llm,
		tts:        tts,
		vad:        NewVADEngine(vadCfg),
		audioOutCh: make(chan LiveAudioChunk, 256),
		errCh:      make(chan error, 16),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// AudioOutChan returns audio chunks ready to stream to caller.
func (m *ModularStreamer) AudioOutChan() <-chan LiveAudioChunk {
	return m.audioOutCh
}

// Errors returns errors encountered in processing.
func (m *ModularStreamer) Errors() <-chan error {
	return m.errCh
}

// SendRealtimeAudio processes inbound caller audio through VAD and STT.
func (m *ModularStreamer) SendRealtimeAudio(pcmChunk []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	// Run VAD on caller chunk
	vadRes := m.vad.ProcessChunk(pcmChunk)

	// If speech ended or threshold completed, trigger filler injection & LLM
	if vadRes.SpeechEnded && !m.isInferring {
		m.isInferring = true
		go m.handleTurn("Kullanıcı konuşması")
	}

	// Feed to STT if service is available
	if m.stt != nil {
		go func(chunk []byte) {
			transcript, isFinal, err := m.stt.ProcessAudio(m.ctx, chunk)
			if err != nil {
				return
			}
			if isFinal && transcript != "" {
				m.mu.Lock()
				if !m.isInferring {
					m.isInferring = true
					go m.handleTurn(transcript)
				}
				m.mu.Unlock()
			}
		}(pcmChunk)
	}

	return nil
}

// TriggerTurn manually initiates a turn with input text (useful for testing or event triggers).
func (m *ModularStreamer) TriggerTurn(userInput string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isInferring = true
	go m.handleTurn(userInput)
}

// handleTurn runs instant filler injection and queries the LLM followed by TTS.
func (m *ModularStreamer) handleTurn(userInput string) {
	defer func() {
		m.mu.Lock()
		m.isInferring = false
		m.mu.Unlock()
	}()

	m.mu.Lock()
	ctx := m.ctx
	fillers := m.cfg.FillerPhrases
	idx := m.fillerIdx % len(fillers)
	m.fillerIdx++
	selectedFiller := fillers[idx]
	m.mu.Unlock()

	// 1. INSTANT FILLER INJECTION:
	// Synthesize and stream filler phrase immediately to keep caller engaged without delay
	fillerPCM, rate, err := m.tts.Synthesize(ctx, selectedFiller, m.cfg.VoiceID)
	if err == nil && len(fillerPCM) > 0 {
		select {
		case <-ctx.Done():
			return
		case m.audioOutCh <- LiveAudioChunk{
			PCMData:       fillerPCM,
			SampleRate:    rate,
			TranscriptOut: selectedFiller,
			IsTurnEnd:     false,
		}:
		}
	}

	// 2. LLM INFERENCE
	var responseText string
	if m.llm != nil {
		resp, err := m.llm.GenerateResponse(ctx, userInput, m.cfg.PersonaPrompt)
		if err != nil {
			select {
			case m.errCh <- err:
			default:
			}
			responseText = "Anlayamadım, lütfen tekrar eder misiniz?"
		} else {
			responseText = resp
		}
	} else {
		responseText = "Size bu konuda yardımcı olmaktan memnuniyet duyarım."
	}

	// 3. CARTESIA / ELEVENLABS TTS SYNTHESIS
	select {
	case <-ctx.Done():
		return
	default:
	}

	mainPCM, rate, err := m.tts.Synthesize(ctx, responseText, m.cfg.VoiceID)
	if err != nil {
		select {
		case m.errCh <- err:
		default:
		}
		return
	}

	// Stream synthesized main audio
	select {
	case <-ctx.Done():
		return
	case m.audioOutCh <- LiveAudioChunk{
		PCMData:       mainPCM,
		SampleRate:    rate,
		TranscriptOut: responseText,
		IsTurnEnd:     true,
	}:
	}
}

// Interrupt immediately cancels ongoing synthesis and clears generation context.
func (m *ModularStreamer) Interrupt() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
	}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.isInferring = false

	// Drain queued audio
	for len(m.audioOutCh) > 0 {
		select {
		case <-m.audioOutCh:
		default:
		}
	}

	// Send interrupted marker
	select {
	case m.audioOutCh <- LiveAudioChunk{Interrupted: true}:
	default:
	}
}

// Close terminates the modular pipeline.
func (m *ModularStreamer) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}
	m.closed = true
	if m.cancel != nil {
		m.cancel()
	}
	return nil
}

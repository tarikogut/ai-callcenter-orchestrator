package ai

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
)

// FreeSwitchStartMessage is the initial handshake JSON sent by mod_audio_fork.
type FreeSwitchStartMessage struct {
	Event      string `json:"event"`       // e.g. "start", "stop"
	CallUUID   string `json:"call_uuid"`   // Channel unique ID
	SampleRate int    `json:"sample_rate"` // e.g. 8000 or 16000
	Channels   int    `json:"channels"`    // 1 (mono)
	Format     string `json:"format"`      // "l16" / "pcm"
	TenantID   string `json:"tenant_id"`
}

// CallControlAction represents a command dispatched to FreeSWITCH ESL / mod_audio_fork.
type CallControlAction struct {
	CallUUID  string                 `json:"call_uuid"`
	Command   string                 `json:"command"` // "uuid_break", "uuid_kill", "uuid_transfer"
	Target    string                 `json:"target"`
	Params    map[string]interface{} `json:"params,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// WSConn defines the WebSocket connection interface used for FreeSWITCH mod_audio_fork.
type WSConn interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	Close() error
}

// LiveAIEngine is the common interface implemented by Gemini Live and Modular Fallback engines.
type LiveAIEngine interface {
	SendRealtimeAudio(pcmChunk []byte) error
	AudioOutChan() <-chan LiveAudioChunk
	Errors() <-chan error
	Close() error
}

// StreamerConfig configures an AudioStreamer instance.
type StreamerConfig struct {
	CallUUID         string
	TenantID         string
	InboundRate      int // FreeSWITCH inbound rate (8000 or 16000)
	OutboundRate     int // FreeSWITCH outbound rate (8000 or 16000)
	AIRate           int // AI Engine native sample rate (16000 or 24000)
	VADConfig        *VADConfig
	OnInterrupt      func(InterruptSignal)
	OnControlAction  func(CallControlAction)
}

// AudioStreamer bridges FreeSWITCH mod_audio_fork PCM WebSocket with the Live AI engine.
type AudioStreamer struct {
	cfg        StreamerConfig
	wsConn     WSConn
	aiEngine   LiveAIEngine
	vad        *VADEngine

	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex

	playbackQueue chan []byte
	isAIPlaying   bool
	closed        bool
	transcriptIn  string
	transcriptOut string
}

// NewAudioStreamer creates and initializes an AudioStreamer session.
func NewAudioStreamer(cfg StreamerConfig, ws WSConn, aiEngine LiveAIEngine) *AudioStreamer {
	if cfg.InboundRate <= 0 {
		cfg.InboundRate = 8000 // FreeSWITCH standard narrowband
	}
	if cfg.OutboundRate <= 0 {
		cfg.OutboundRate = cfg.InboundRate
	}
	if cfg.AIRate <= 0 {
		cfg.AIRate = 16000 // Gemini / STT default
	}

	vadCfg := DefaultVADConfig()
	if cfg.VADConfig != nil {
		vadCfg = *cfg.VADConfig
	}
	vadCfg.CallUUID = cfg.CallUUID
	vadCfg.SampleRate = cfg.InboundRate

	ctx, cancel := context.WithCancel(context.Background())

	streamer := &AudioStreamer{
		cfg:           cfg,
		wsConn:        ws,
		aiEngine:      aiEngine,
		vad:           NewVADEngine(vadCfg),
		playbackQueue: make(chan []byte, 500),
		ctx:           ctx,
		cancel:        cancel,
	}

	// Wire barge-in callback
	streamer.vad.OnInterrupt(func(sig InterruptSignal) {
		streamer.handleBargeIn(sig)
	})

	return streamer
}

// CallUUID returns the call unique identifier.
func (s *AudioStreamer) CallUUID() string {
	return s.cfg.CallUUID
}

// TenantID returns the tenant identifier.
func (s *AudioStreamer) TenantID() string {
	return s.cfg.TenantID
}

// SetOnControlAction registers or updates the control action callback.
func (s *AudioStreamer) SetOnControlAction(fn func(CallControlAction)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.OnControlAction = fn
}

// SetOnInterrupt registers or updates the interrupt callback.
func (s *AudioStreamer) SetOnInterrupt(fn func(InterruptSignal)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.OnInterrupt = fn
}

// VAD returns the underlying VAD engine.
func (s *AudioStreamer) VAD() *VADEngine {
	return s.vad
}

// PlaybackQueue returns the audio playback queue channel.
func (s *AudioStreamer) PlaybackQueue() chan []byte {
	return s.playbackQueue
}

// IsAIPlaying returns whether AI playback is actively streaming to the caller.
func (s *AudioStreamer) IsAIPlaying() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isAIPlaying
}

// setAIPlaying updates the active playing state and informs the VAD engine.
func (s *AudioStreamer) setAIPlaying(playing bool) {
	s.mu.Lock()
	s.isAIPlaying = playing
	s.mu.Unlock()
	s.vad.SetAIPlaying(playing)
}

// handleBargeIn is called when caller speech interrupts active AI playback.
func (s *AudioStreamer) handleBargeIn(sig InterruptSignal) {
	log.Printf("[AudioStreamer] 🛑 BARGE-IN DETECTED for call %s (RMS: %.1f dBFS: %.1f). Breaking playback!",
		s.cfg.CallUUID, sig.RMS, sig.DBFS)

	// 1. Immediately drain the pending playback queue
	s.drainPlaybackQueue()

	// 2. Mark AI as stopped playing
	s.setAIPlaying(false)

	// 3. Emit uuid_break control action to FreeSWITCH
	breakAction := CallControlAction{
		CallUUID:  s.cfg.CallUUID,
		Command:   "uuid_break",
		Target:    "all",
		Timestamp: time.Now(),
	}

	if s.cfg.OnControlAction != nil {
		s.cfg.OnControlAction(breakAction)
	}

	// 4. Notify external interrupt callback
	if s.cfg.OnInterrupt != nil {
		s.cfg.OnInterrupt(sig)
	}

	// 5. If modular engine, invoke its interrupt
	if mod, ok := s.aiEngine.(*ModularStreamer); ok {
		mod.Interrupt()
	}
}

// drainPlaybackQueue flushes any buffered audio chunks pending transmission to FreeSWITCH.
func (s *AudioStreamer) drainPlaybackQueue() {
	for {
		select {
		case <-s.playbackQueue:
		default:
			return
		}
	}
}

// Start begins bidirectional audio streaming loops.
func (s *AudioStreamer) Start() {
	var wg sync.WaitGroup
	wg.Add(3)

	// 1. Read from FreeSWITCH -> Process VAD -> Send to Live AI
	go func() {
		defer wg.Done()
		s.readFromFreeSwitchLoop()
	}()

	// 2. Read from Live AI -> Resample -> Enqueue to PlaybackQueue
	go func() {
		defer wg.Done()
		s.readFromAIEngineLoop()
	}()

	// 3. Read from PlaybackQueue -> Write binary frames to FreeSWITCH
	go func() {
		defer wg.Done()
		s.writeToFreeSwitchLoop()
	}()

	wg.Wait()
	s.Close()
}

// readFromFreeSwitchLoop receives audio chunks from FreeSWITCH mod_audio_fork.
func (s *AudioStreamer) readFromFreeSwitchLoop() {
	defer s.cancel()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		if s.wsConn == nil {
			return
		}

		msgType, data, err := s.wsConn.ReadMessage()
		if err != nil {
			log.Printf("[AudioStreamer] FreeSWITCH read error (call %s): %v", s.cfg.CallUUID, err)
			return
		}

		switch msgType {
		case websocket.BinaryMessage:
			// Process incoming caller PCM
			if len(data) == 0 {
				continue
			}

			// Pass chunk through VAD & Barge-in engine
			s.vad.ProcessChunk(data)

			// Resample to AI engine rate (e.g. 8kHz -> 16kHz)
			var pcmToSend []byte
			if s.cfg.InboundRate != s.cfg.AIRate {
				pcmToSend = ResamplePCM(data, s.cfg.InboundRate, s.cfg.AIRate)
			} else {
				pcmToSend = data
			}

			// Forward to Live AI engine
			if s.aiEngine != nil {
				if err := s.aiEngine.SendRealtimeAudio(pcmToSend); err != nil {
					log.Printf("[AudioStreamer] Error forwarding audio to AI: %v", err)
				}
			}

		case websocket.TextMessage:
			// Control JSON messages from FreeSWITCH mod_audio_fork
			var msg FreeSwitchStartMessage
			if err := json.Unmarshal(data, &msg); err == nil {
				if msg.Event == "stop" {
					log.Printf("[AudioStreamer] FreeSWITCH stop event received for %s", s.cfg.CallUUID)
					return
				}
				if msg.SampleRate > 0 {
					s.cfg.InboundRate = msg.SampleRate
					s.cfg.OutboundRate = msg.SampleRate
				}
			}
		}
	}
}

// readFromAIEngineLoop receives synthesized audio chunks from Gemini Live / Modular engine.
func (s *AudioStreamer) readFromAIEngineLoop() {
	defer s.cancel()

	if s.aiEngine == nil {
		return
	}

	aiOutCh := s.aiEngine.AudioOutChan()

	for {
		select {
		case <-s.ctx.Done():
			return
		case chunk, ok := <-aiOutCh:
			if !ok {
				return
			}

			// If server indicated an interruption, break immediately
			if chunk.Interrupted {
				s.drainPlaybackQueue()
				s.setAIPlaying(false)
				continue
			}

			if chunk.TranscriptIn != "" {
				s.mu.Lock()
				s.transcriptIn = chunk.TranscriptIn
				s.mu.Unlock()
			}
			if chunk.TranscriptOut != "" {
				s.mu.Lock()
				s.transcriptOut = chunk.TranscriptOut
				s.mu.Unlock()
			}

			if len(chunk.PCMData) == 0 {
				if chunk.IsTurnEnd {
					// Mark turn end when playback finishes
					time.AfterFunc(100*time.Millisecond, func() {
						s.setAIPlaying(false)
					})
				}
				continue
			}

			// Resample AI output to FreeSWITCH rate (e.g. 24kHz -> 8kHz or 16kHz)
			sourceRate := chunk.SampleRate
			if sourceRate <= 0 {
				sourceRate = 24000
			}

			targetRate := s.cfg.OutboundRate
			var resampled []byte
			if sourceRate != targetRate {
				resampled = ResamplePCM(chunk.PCMData, sourceRate, targetRate)
			} else {
				resampled = chunk.PCMData
			}

			// Mark that AI is playing audio
			s.setAIPlaying(true)

			// Slice into 20ms frames for smooth FreeSWITCH jitter buffering
			// 20ms @ 8000Hz = 160 samples = 320 bytes
			// 20ms @ 16000Hz = 320 samples = 640 bytes
			frameBytes := (targetRate * 20 / 1000) * 2
			framer := NewAudioFramer(frameBytes)
			frames := framer.Push(resampled)

			for _, f := range frames {
				select {
				case <-s.ctx.Done():
					return
				case s.playbackQueue <- f:
				}
			}

			if remainder := framer.Flush(); len(remainder) > 0 {
				select {
				case <-s.ctx.Done():
					return
				case s.playbackQueue <- remainder:
				}
			}

			if chunk.IsTurnEnd {
				// Turn completed; will unset isAIPlaying once playbackQueue drains
				go func() {
					for len(s.playbackQueue) > 0 {
						time.Sleep(10 * time.Millisecond)
					}
					s.setAIPlaying(false)
				}()
			}
		}
	}
}

// writeToFreeSwitchLoop streams buffered PCM frames back to FreeSWITCH.
func (s *AudioStreamer) writeToFreeSwitchLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case pcmFrame, ok := <-s.playbackQueue:
			if !ok {
				return
			}
			if len(pcmFrame) == 0 {
				continue
			}

			if s.wsConn != nil {
				if err := s.wsConn.WriteMessage(websocket.BinaryMessage, pcmFrame); err != nil {
					log.Printf("[AudioStreamer] Error writing audio to FreeSWITCH (call %s): %v", s.cfg.CallUUID, err)
					return
				}
			}
		}
	}
}

// GetTranscripts returns the accumulated caller and AI transcriptions.
func (s *AudioStreamer) GetTranscripts() (string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.transcriptIn, s.transcriptOut
}

// Close gracefully terminates the streaming session.
func (s *AudioStreamer) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()

	s.cancel()

	if s.aiEngine != nil {
		_ = s.aiEngine.Close()
	}

	if s.wsConn != nil {
		_ = s.wsConn.Close()
	}

	s.drainPlaybackQueue()
	close(s.playbackQueue)
	return nil
}

// AudioStreamManager maintains active audio streamers by CallUUID.
type AudioStreamManager struct {
	mu        sync.RWMutex
	streamers map[string]*AudioStreamer
}

// NewAudioStreamManager creates a new stream registry.
func NewAudioStreamManager() *AudioStreamManager {
	return &AudioStreamManager{
		streamers: make(map[string]*AudioStreamer),
	}
}

// Register adds an active streamer.
func (m *AudioStreamManager) Register(callUUID string, streamer *AudioStreamer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streamers[callUUID] = streamer
}

// Get retrieves a streamer by callUUID.
func (m *AudioStreamManager) Get(callUUID string) (*AudioStreamer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.streamers[callUUID]
	return s, ok
}

// Unregister removes and closes a streamer.
func (m *AudioStreamManager) Unregister(callUUID string) {
	m.mu.Lock()
	s, ok := m.streamers[callUUID]
	if ok {
		delete(m.streamers, callUUID)
	}
	m.mu.Unlock()

	if ok && s != nil {
		_ = s.Close()
	}
}

// HandleHangup terminates the audio session for a hung up call.
func (m *AudioStreamManager) HandleHangup(callUUID string) {
	m.Unregister(callUUID)
}

package ai

import (
	"sync"
	"time"
)

// VADState indicates whether speech or silence is currently detected.
type VADState int

const (
	VADStateSilence VADState = iota
	VADStateSpeaking
)

func (s VADState) String() string {
	switch s {
	case VADStateSpeaking:
		return "SPEAKING"
	default:
		return "SILENCE"
	}
}

// InterruptSignal is emitted when caller barge-in or user speech interruption occurs.
type InterruptSignal struct {
	CallUUID    string    `json:"call_uuid"`
	Timestamp   time.Time `json:"timestamp"`
	Energy      float64   `json:"energy"`
	RMS         float64   `json:"rms"`
	DBFS        float64   `json:"dbfs"`
	Reason      string    `json:"reason"` // "barge_in", "user_speech"
}

// VADConfig holds tuning parameters for voice activity and barge-in detection.
type VADConfig struct {
	SampleRate       int           // e.g. 8000 or 16000 Hz
	EnergyThreshold  float64       // RMS threshold, default ~500.0
	MinSpeechFrames  int           // Consecutive frames needed to confirm speech start, default 2
	MinSilenceFrames int           // Consecutive frames needed to confirm speech end, default 10 (~200ms)
	BargeInCooldown  time.Duration // Minimum duration between barge-in interrupts, default 300ms
	CallUUID         string        // Associated call UUID
}

// DefaultVADConfig returns production defaults for FreeSWITCH audio streams.
func DefaultVADConfig() VADConfig {
	return VADConfig{
		SampleRate:       16000,
		EnergyThreshold:  500.0,
		MinSpeechFrames:  2,
		MinSilenceFrames: 10,
		BargeInCooldown:  300 * time.Millisecond,
	}
}

// VADResult holds the classification output of an audio chunk.
type VADResult struct {
	IsSpeech      bool
	SpeechStarted bool
	SpeechEnded   bool
	RMS           float64
	DBFS          float64
	BargeInFired  bool
}

// VADEngine evaluates incoming audio frames for speech energy and coordinates barge-in interrupts.
type VADEngine struct {
	mu     sync.RWMutex
	config VADConfig

	state             VADState
	consecutiveSpeech int
	consecutiveSilent int

	aiPlaying      bool
	lastInterrupt  time.Time
	interruptChan  chan InterruptSignal
	interruptFuncs []func(InterruptSignal)
}

// NewVADEngine creates and initializes a new VAD and Barge-In engine.
func NewVADEngine(cfg VADConfig) *VADEngine {
	if cfg.EnergyThreshold <= 0 {
		cfg.EnergyThreshold = 500.0
	}
	if cfg.MinSpeechFrames <= 0 {
		cfg.MinSpeechFrames = 2
	}
	if cfg.MinSilenceFrames <= 0 {
		cfg.MinSilenceFrames = 10
	}
	if cfg.BargeInCooldown <= 0 {
		cfg.BargeInCooldown = 300 * time.Millisecond
	}

	return &VADEngine{
		config:        cfg,
		state:         VADStateSilence,
		interruptChan: make(chan InterruptSignal, 64),
	}
}

// SetAIPlaying updates the status of whether the AI is currently speaking or playing audio to caller.
func (v *VADEngine) SetAIPlaying(playing bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.aiPlaying = playing
}

// IsAIPlaying checks if AI audio is currently actively playing.
func (v *VADEngine) IsAIPlaying() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.aiPlaying
}

// OnInterrupt registers a callback to be notified when barge-in is triggered.
func (v *VADEngine) OnInterrupt(fn func(InterruptSignal)) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.interruptFuncs = append(v.interruptFuncs, fn)
}

// InterruptChan provides a channel that receives interrupt signals.
func (v *VADEngine) InterruptChan() <-chan InterruptSignal {
	return v.interruptChan
}

// ProcessChunk analyzes a raw 16-bit PCM chunk, updates speech state, and fires barge-in if warranted.
func (v *VADEngine) ProcessChunk(pcmData []byte) VADResult {
	rms := CalculateRMS(pcmData)
	dbfs := CalculateDBFS(rms)

	v.mu.Lock()
	defer v.mu.Unlock()

	result := VADResult{
		RMS:  rms,
		DBFS: dbfs,
	}

	isAboveThreshold := rms >= v.config.EnergyThreshold

	if isAboveThreshold {
		v.consecutiveSpeech++
		v.consecutiveSilent = 0

		if v.state == VADStateSilence && v.consecutiveSpeech >= v.config.MinSpeechFrames {
			v.state = VADStateSpeaking
			result.SpeechStarted = true
		}
	} else {
		v.consecutiveSilent++
		if v.consecutiveSilent >= v.config.MinSilenceFrames {
			if v.state == VADStateSpeaking {
				result.SpeechEnded = true
			}
			v.state = VADStateSilence
			v.consecutiveSpeech = 0
		}
	}

	result.IsSpeech = (v.state == VADStateSpeaking)

	// Check Barge-In Condition:
	// If caller is speaking (or just started speech) WHILE the AI is actively outputting audio,
	// fire an immediate barge-in interruption signal.
	if v.aiPlaying && (result.SpeechStarted || (result.IsSpeech && isAboveThreshold)) {
		now := time.Now()
		if now.Sub(v.lastInterrupt) >= v.config.BargeInCooldown {
			v.lastInterrupt = now
			result.BargeInFired = true

			sig := InterruptSignal{
				CallUUID:  v.config.CallUUID,
				Timestamp: now,
				Energy:    rms,
				RMS:       rms,
				DBFS:      dbfs,
				Reason:    "barge_in",
			}

			// Non-blocking send to channel
			select {
			case v.interruptChan <- sig:
			default:
			}

			// Notify registered callbacks
			for _, fn := range v.interruptFuncs {
				go fn(sig)
			}
		}
	}

	return result
}

// EmitManualInterrupt allows external components to force a barge-in interrupt.
func (v *VADEngine) EmitManualInterrupt(reason string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	now := time.Now()
	v.lastInterrupt = now

	sig := InterruptSignal{
		CallUUID:  v.config.CallUUID,
		Timestamp: now,
		Energy:    0,
		RMS:       0,
		DBFS:      0,
		Reason:    reason,
	}

	select {
	case v.interruptChan <- sig:
	default:
	}

	for _, fn := range v.interruptFuncs {
		go fn(sig)
	}
}

// Reset clears speech frame counters and state.
func (v *VADEngine) Reset() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.state = VADStateSilence
	v.consecutiveSpeech = 0
	v.consecutiveSilent = 0
	v.aiPlaying = false
}

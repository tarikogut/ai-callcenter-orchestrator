package ai

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/fasthttp/websocket"
)

// Helper: generate 16-bit PCM mono sine wave audio bytes
func generateSinePCM(freq float64, sampleRate int, durationSec float64, amplitude int16) []byte {
	numSamples := int(float64(sampleRate) * durationSec)
	data := make([]byte, numSamples*2)
	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		sample := int16(float64(amplitude) * math.Sin(2*math.Pi*freq*t))
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], uint16(sample))
	}
	return data
}

// Helper: generate silence PCM
func generateSilencePCM(sampleRate int, durationSec float64) []byte {
	numSamples := int(float64(sampleRate) * durationSec)
	return make([]byte, numSamples*2)
}

// ---------------------------------------------------------------------
// 1. VAD & Energy Detection Tests
// ---------------------------------------------------------------------

func TestCalculateRMSAndDBFS(t *testing.T) {
	// Silence test
	silence := generateSilencePCM(16000, 0.02)
	rmsSilence := CalculateRMS(silence)
	if rmsSilence != 0 {
		t.Errorf("Expected RMS 0 for silence, got %f", rmsSilence)
	}
	dbfsSilence := CalculateDBFS(rmsSilence)
	if dbfsSilence != -100 {
		t.Errorf("Expected dBFS -100 for silence, got %f", dbfsSilence)
	}

	// Sine wave with amplitude 10,000 (theoretical RMS ~ 7071)
	sine := generateSinePCM(440, 16000, 0.02, 10000)
	rmsSine := CalculateRMS(sine)
	if rmsSine < 6900 || rmsSine > 7200 {
		t.Errorf("Expected RMS ~7071, got %f", rmsSine)
	}

	dbfsSine := CalculateDBFS(rmsSine)
	if dbfsSine < -15 || dbfsSine > -12 {
		t.Errorf("Expected dBFS ~ -13.3 dBFS, got %f", dbfsSine)
	}
}

func TestVADSpeechDetectionAndTransitions(t *testing.T) {
	cfg := VADConfig{
		SampleRate:       16000,
		EnergyThreshold:  500.0,
		MinSpeechFrames:  2,
		MinSilenceFrames: 3,
		BargeInCooldown:  100 * time.Millisecond,
		CallUUID:         "test-call-1",
	}
	vad := NewVADEngine(cfg)

	speechChunk := generateSinePCM(300, 16000, 0.02, 4000) // RMS ~ 2828 > 500
	silentChunk := generateSilencePCM(16000, 0.02)

	// Frame 1: speech above threshold, but MinSpeechFrames is 2 -> not speaking yet
	res1 := vad.ProcessChunk(speechChunk)
	if res1.IsSpeech || res1.SpeechStarted {
		t.Errorf("Frame 1: should not trigger speech yet with MinSpeechFrames=2")
	}

	// Frame 2: second speech frame -> speech confirmed!
	res2 := vad.ProcessChunk(speechChunk)
	if !res2.IsSpeech || !res2.SpeechStarted {
		t.Errorf("Frame 2: should trigger SpeechStarted=true and IsSpeech=true")
	}

	// Frame 3: continuing speech
	res3 := vad.ProcessChunk(speechChunk)
	if !res3.IsSpeech || res3.SpeechStarted {
		t.Errorf("Frame 3: should be IsSpeech=true and SpeechStarted=false (already started)")
	}

	// Feed 2 silent chunks: MinSilenceFrames is 3, so still marked as speaking (hangover)
	vad.ProcessChunk(silentChunk)
	vad.ProcessChunk(silentChunk)
	if !vad.ProcessChunk(speechChunk).IsSpeech {
		t.Errorf("Intermittent silence should be bridged by hangover")
	}

	// Feed 3 consecutive silent chunks -> should trigger SpeechEnded
	vad.ProcessChunk(silentChunk)
	vad.ProcessChunk(silentChunk)
	resEnd := vad.ProcessChunk(silentChunk)
	if !resEnd.SpeechEnded || resEnd.IsSpeech {
		t.Errorf("Frame 3 of silence: should trigger SpeechEnded=true and IsSpeech=false")
	}
}

// ---------------------------------------------------------------------
// 2. Barge-In Interruption Tests
// ---------------------------------------------------------------------

func TestVADBargeInTriggerWhenAIPlaying(t *testing.T) {
	cfg := VADConfig{
		SampleRate:       16000,
		EnergyThreshold:  500.0,
		MinSpeechFrames:  2,
		MinSilenceFrames: 5,
		BargeInCooldown:  200 * time.Millisecond,
		CallUUID:         "call-bargein-test",
	}
	vad := NewVADEngine(cfg)

	speechChunk := generateSinePCM(400, 16000, 0.02, 5000)

	var interruptReceived *InterruptSignal
	var interruptMu sync.Mutex
	vad.OnInterrupt(func(sig InterruptSignal) {
		interruptMu.Lock()
		interruptReceived = &sig
		interruptMu.Unlock()
	})

	// Case 1: AI is NOT playing. Caller speaks -> No barge-in should be fired.
	vad.SetAIPlaying(false)
	vad.ProcessChunk(speechChunk)
	res1 := vad.ProcessChunk(speechChunk)
	if res1.BargeInFired {
		t.Errorf("Barge-in should NOT fire when AI is not playing")
	}
	if interruptReceived != nil {
		t.Errorf("InterruptSignal should not be dispatched when AI is not playing")
	}

	// Reset VAD state
	vad.Reset()

	// Case 2: AI IS playing. Caller speaks -> Barge-in MUST fire!
	vad.SetAIPlaying(true)
	vad.ProcessChunk(speechChunk)
	res2 := vad.ProcessChunk(speechChunk)
	if !res2.BargeInFired {
		t.Errorf("Barge-in MUST fire when caller speaks while AI is playing")
	}

	// Wait briefly for goroutine callback
	time.Sleep(30 * time.Millisecond)
	interruptMu.Lock()
	if interruptReceived == nil {
		t.Fatalf("Expected InterruptSignal callback to be called")
	}
	if interruptReceived.CallUUID != "call-bargein-test" {
		t.Errorf("Expected CallUUID 'call-bargein-test', got '%s'", interruptReceived.CallUUID)
	}
	if interruptReceived.Reason != "barge_in" {
		t.Errorf("Expected reason 'barge_in', got '%s'", interruptReceived.Reason)
	}
	interruptMu.Unlock()

	// Check channel delivery as well
	select {
	case sig := <-vad.InterruptChan():
		if sig.CallUUID != "call-bargein-test" {
			t.Errorf("Signal from channel mismatch: %+v", sig)
		}
	default:
		t.Errorf("Signal not found in InterruptChan")
	}

	// Case 3: Cooldown enforcement. Immediate subsequent frame should NOT fire barge-in again.
	res3 := vad.ProcessChunk(speechChunk)
	if res3.BargeInFired {
		t.Errorf("Subsequent frame within cooldown should NOT re-fire barge-in")
	}
}

// ---------------------------------------------------------------------
// 3. Audio Framing & Resampling Tests
// ---------------------------------------------------------------------

func TestAudioFramer(t *testing.T) {
	// Frame size: 320 bytes (20ms @ 8kHz)
	framer := NewAudioFramer(320)

	// Push 100 bytes: no frame ready
	frames := framer.Push(make([]byte, 100))
	if len(frames) != 0 {
		t.Fatalf("Expected 0 frames for 100 bytes, got %d", len(frames))
	}

	// Push 600 bytes: total buffer now 700 bytes -> 2 full frames (640 bytes), 60 remainder
	frames = framer.Push(make([]byte, 600))
	if len(frames) != 2 {
		t.Fatalf("Expected 2 frames, got %d", len(frames))
	}
	if len(frames[0]) != 320 || len(frames[1]) != 320 {
		t.Errorf("Frames should have size 320")
	}

	// Flush remainder (60 bytes padded to 320)
	flushed := framer.Flush()
	if len(flushed) != 320 {
		t.Fatalf("Expected flushed frame size 320, got %d", len(flushed))
	}

	// After flush, buffer should be empty
	if framer.Flush() != nil {
		t.Errorf("Subsequent flush should return nil")
	}
}

func TestPCMResamplingRatios(t *testing.T) {
	// 8kHz -> 16kHz (factor 2)
	pcm8k := generateSinePCM(400, 8000, 0.05, 5000) // 400 samples = 800 bytes
	pcm16k := ResamplePCM(pcm8k, 8000, 16000)
	expected16kLen := len(pcm8k) * 2
	if len(pcm16k) != expected16kLen {
		t.Errorf("8k->16k: expected %d bytes, got %d", expected16kLen, len(pcm16k))
	}

	// 16kHz -> 8kHz (factor 1/2)
	pcmBack8k := ResamplePCM(pcm16k, 16000, 8000)
	expected8kLen := len(pcm16k) / 2
	if len(pcmBack8k) != expected8kLen {
		t.Errorf("16k->8k: expected %d bytes, got %d", expected8kLen, len(pcmBack8k))
	}

	// 24kHz -> 8kHz (factor 1/3)
	pcm24k := generateSinePCM(400, 24000, 0.06, 5000) // 1440 samples = 2880 bytes
	pcm24to8 := ResamplePCM(pcm24k, 24000, 8000)
	expected24to8Len := len(pcm24k) / 3
	if len(pcm24to8) != expected24to8Len {
		t.Errorf("24k->8k: expected %d bytes, got %d", expected24to8Len, len(pcm24to8))
	}

	// 24kHz -> 16kHz (ratio 2/3)
	pcm24to16 := ResamplePCM(pcm24k, 24000, 16000)
	expected24to16Len := (len(pcm24k) / 6) * 4 // (samples / 3 * 2) * 2 bytes
	if len(pcm24to16) != expected24to16Len {
		t.Errorf("24k->16k: expected %d bytes, got %d", expected24to16Len, len(pcm24to16))
	}

	// 16kHz -> 24kHz (ratio 3/2)
	pcm16to24 := ResamplePCM(pcm16k, 16000, 24000)
	expected16to24Len := (len(pcm16k) / 4) * 6
	if len(pcm16to24) != expected16to24Len {
		t.Errorf("16k->24k: expected %d bytes, got %d", expected16to24Len, len(pcm16to24))
	}
}

// ---------------------------------------------------------------------
// 4. Gemini Live WebSocket Protocol Serialization Tests
// ---------------------------------------------------------------------

func TestGeminiLiveProtocolPayloads(t *testing.T) {
	// 1. Setup message verification
	setup := SetupMessage{
		Setup: BidiGenerateContentSetup{
			Model: "models/gemini-2.0-flash-exp",
			GenerationConfig: &GenerationConfig{
				ResponseModalities: []string{"AUDIO"},
				SpeechConfig: &SpeechConfig{
					VoiceConfig: VoiceConfig{
						PrebuiltVoiceConfig: PrebuiltVoiceConfig{VoiceName: "Puck"},
					},
				},
			},
			SystemInstruction: &Content{
				Parts: []ContentPart{{Text: "You are an AI call center agent."}},
			},
		},
	}

	setupJSON, err := json.Marshal(setup)
	if err != nil {
		t.Fatalf("Failed to marshal setup message: %v", err)
	}

	var parsedSetup map[string]interface{}
	if err := json.Unmarshal(setupJSON, &parsedSetup); err != nil {
		t.Fatalf("Failed to unmarshal setup JSON: %v", err)
	}
	setupObj := parsedSetup["setup"].(map[string]interface{})
	if setupObj["model"] != "models/gemini-2.0-flash-exp" {
		t.Errorf("Unexpected model in setup: %v", setupObj["model"])
	}

	// 2. Realtime Input audio chunk verification
	dummyAudio := []byte{0x00, 0x10, 0x20, 0x30}
	b64Audio := base64.StdEncoding.EncodeToString(dummyAudio)

	rtMsg := RealtimeInputMessage{
		RealtimeInput: RealtimeInputPayload{
			MediaChunks: []RealtimeMediaChunk{
				{
					MimeType: "audio/pcm;rate=16000",
					Data:     b64Audio,
				},
			},
		},
	}

	rtJSON, err := json.Marshal(rtMsg)
	if err != nil {
		t.Fatalf("Failed to marshal realtime input: %v", err)
	}

	var parsedRT map[string]interface{}
	if err := json.Unmarshal(rtJSON, &parsedRT); err != nil {
		t.Fatalf("Failed to unmarshal realtime input: %v", err)
	}
	rtPayload := parsedRT["realtime_input"].(map[string]interface{})
	chunks := rtPayload["media_chunks"].([]interface{})
	if len(chunks) != 1 {
		t.Fatalf("Expected 1 media chunk, got %d", len(chunks))
	}
	chunk0 := chunks[0].(map[string]interface{})
	if chunk0["mime_type"] != "audio/pcm;rate=16000" {
		t.Errorf("Unexpected mime_type: %v", chunk0["mime_type"])
	}
	if chunk0["data"] != b64Audio {
		t.Errorf("Unexpected audio base64 data: %v", chunk0["data"])
	}

	// 3. Server message deserialization test
	serverJSON := `{
		"serverContent": {
			"modelTurn": {
				"parts": [
					{
						"inlineData": {
							"mimeType": "audio/pcm;rate=24000",
							"data": "` + b64Audio + `"
						}
					}
				]
			},
			"turnComplete": true,
			"interrupted": false
		}
	}`

	var srvMsg BidiServerMessage
	if err := json.Unmarshal([]byte(serverJSON), &srvMsg); err != nil {
		t.Fatalf("Failed to unmarshal server message: %v", err)
	}
	if srvMsg.ServerContent == nil {
		t.Fatalf("ServerContent should not be nil")
	}
	if !srvMsg.ServerContent.TurnComplete {
		t.Errorf("Expected TurnComplete=true")
	}
	if srvMsg.ServerContent.Interrupted {
		t.Errorf("Expected Interrupted=false")
	}
	if len(srvMsg.ServerContent.ModelTurn.Parts) != 1 {
		t.Fatalf("Expected 1 part, got %d", len(srvMsg.ServerContent.ModelTurn.Parts))
	}
	decoded, _ := base64.StdEncoding.DecodeString(srvMsg.ServerContent.ModelTurn.Parts[0].InlineData.Data)
	if len(decoded) != len(dummyAudio) {
		t.Errorf("Decoded audio length mismatch: %d vs %d", len(decoded), len(dummyAudio))
	}
}

// ---------------------------------------------------------------------
// 5. Modular Streaming Fallback & Instant Filler Injection Tests
// ---------------------------------------------------------------------

type mockLLM struct {
	response string
}

func (m *mockLLM) GenerateResponse(ctx context.Context, prompt string, persona string) (string, error) {
	return m.response, nil
}

func TestModularStreamerInstantFillerInjection(t *testing.T) {
	customFillers := []string{
		"Ha tamam bir saniye bakıyorum...",
	}

	cfg := ModularStreamerConfig{
		PersonaName:   "Eczacı Asistanı",
		PersonaPrompt: "Test persona",
		VoiceID:       "cartesia-tr-female",
		FillerPhrases: customFillers,
		SampleRate:    16000,
	}

	llm := &mockLLM{response: "Aspirin stoklarımızda mevcuttur."}
	streamer := NewModularStreamer(cfg, nil, llm, nil)
	defer streamer.Close()

	// Trigger a turn
	streamer.TriggerTurn("Aspirin var mı?")

	// 1. First chunk MUST be the instant filler phrase
	select {
	case chunk := <-streamer.AudioOutChan():
		if chunk.TranscriptOut != "Ha tamam bir saniye bakıyorum..." {
			t.Errorf("Expected first chunk to be filler 'Ha tamam bir saniye bakıyorum...', got '%s'", chunk.TranscriptOut)
		}
		if chunk.IsTurnEnd {
			t.Errorf("Filler should have IsTurnEnd=false")
		}
		if len(chunk.PCMData) == 0 {
			t.Errorf("Filler audio data should not be empty")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("Timeout waiting for instant filler audio")
	}

	// 2. Second chunk is the LLM synthesized main answer
	select {
	case chunk := <-streamer.AudioOutChan():
		if chunk.TranscriptOut != "Aspirin stoklarımızda mevcuttur." {
			t.Errorf("Expected LLM answer 'Aspirin stoklarımızda mevcuttur.', got '%s'", chunk.TranscriptOut)
		}
		if !chunk.IsTurnEnd {
			t.Errorf("Main response should have IsTurnEnd=true")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("Timeout waiting for LLM response audio")
	}
}

func TestModularStreamerInterruptClearsQueue(t *testing.T) {
	cfg := ModularStreamerConfig{
		PersonaName:   "Test Agent",
		FillerPhrases: []string{"Bir saniye..."},
		SampleRate:    16000,
	}
	streamer := NewModularStreamer(cfg, nil, nil, nil)
	defer streamer.Close()

	// Trigger turn and immediately interrupt
	streamer.TriggerTurn("Merhaba")
	streamer.Interrupt()

	// Interrupted marker should be received
	select {
	case chunk := <-streamer.AudioOutChan():
		if !chunk.Interrupted {
			t.Errorf("Expected chunk with Interrupted=true after call to Interrupt()")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("Timeout waiting for interrupted chunk")
	}
}

// ---------------------------------------------------------------------
// 6. AudioStreamer FreeSWITCH Integration & Barge-In Playback Cancellation
// ---------------------------------------------------------------------

// MockWSConn simulates FreeSWITCH mod_audio_fork WebSocket connection
type MockWSConn struct {
	readCh  chan []byte
	writeCh chan []byte
	closed  bool
	mu      sync.Mutex
}

func NewMockWSConn() *MockWSConn {
	return &MockWSConn{
		readCh:  make(chan []byte, 100),
		writeCh: make(chan []byte, 100),
	}
}

func (m *MockWSConn) ReadMessage() (messageType int, p []byte, err error) {
	data, ok := <-m.readCh
	if !ok {
		return 0, nil, websocket.ErrCloseSent
	}
	return websocket.BinaryMessage, data, nil
}

func (m *MockWSConn) WriteMessage(messageType int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return websocket.ErrCloseSent
	}
	m.writeCh <- data
	return nil
}

func (m *MockWSConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		close(m.readCh)
	}
	return nil
}

// MockAIEngine simulates a LiveAIEngine
type MockAIEngine struct {
	audioOutCh chan LiveAudioChunk
	errCh      chan error
	inboundAudio [][]byte
	mu         sync.Mutex
}

func NewMockAIEngine() *MockAIEngine {
	return &MockAIEngine{
		audioOutCh: make(chan LiveAudioChunk, 100),
		errCh:      make(chan error, 10),
	}
}

func (m *MockAIEngine) SendRealtimeAudio(pcmChunk []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inboundAudio = append(m.inboundAudio, pcmChunk)
	return nil
}

func (m *MockAIEngine) AudioOutChan() <-chan LiveAudioChunk {
	return m.audioOutCh
}

func (m *MockAIEngine) Errors() <-chan error {
	return m.errCh
}

func (m *MockAIEngine) Close() error {
	return nil
}

func TestAudioStreamerBargeInCancelsPlayback(t *testing.T) {
	mockWS := NewMockWSConn()
	mockAI := NewMockAIEngine()

	var controlActions []CallControlAction
	var actionMu sync.Mutex

	cfg := StreamerConfig{
		CallUUID:     "call-fs-test-999",
		TenantID:     "tenant-test",
		InboundRate:  8000,
		OutboundRate: 8000,
		AIRate:       16000,
		VADConfig: &VADConfig{
			SampleRate:       8000,
			EnergyThreshold:  500.0,
			MinSpeechFrames:  1, // Trigger immediately for test
			BargeInCooldown:  100 * time.Millisecond,
		},
		OnControlAction: func(act CallControlAction) {
			actionMu.Lock()
			controlActions = append(controlActions, act)
			actionMu.Unlock()
		},
	}

	streamer := NewAudioStreamer(cfg, mockWS, mockAI)

	// Run streamer loops in background
	go streamer.Start()
	defer streamer.Close()

	// 1. Simulate AI sending audio -> streamer queues it up and sets isAIPlaying = true
	aiAudio := generateSinePCM(300, 24000, 0.2, 4000)
	mockAI.audioOutCh <- LiveAudioChunk{
		PCMData:    aiAudio,
		SampleRate: 24000,
		IsTurnEnd:  false,
	}

	// Wait for streamer to process AI audio into playbackQueue
	time.Sleep(50 * time.Millisecond)
	if !streamer.IsAIPlaying() {
		t.Errorf("Streamer should be in AI playing state after receiving AI audio")
	}

	// 2. Caller speaks while AI is playing -> send caller speech frame via mockWS
	callerSpeech := generateSinePCM(400, 8000, 0.02, 6000) // RMS > 500
	mockWS.readCh <- callerSpeech

	// Wait for VAD and barge-in handling
	time.Sleep(50 * time.Millisecond)

	// Verify barge-in resulted in uuid_break command emitted
	actionMu.Lock()
	foundBreak := false
	for _, act := range controlActions {
		if act.Command == "uuid_break" && act.CallUUID == "call-fs-test-999" {
			foundBreak = true
			break
		}
	}
	actionMu.Unlock()

	if !foundBreak {
		t.Errorf("Expected 'uuid_break' control action to be emitted on caller barge-in")
	}

	// Verify that playback queue was drained
	if len(streamer.PlaybackQueue()) > 0 {
		t.Errorf("PlaybackQueue should have been drained on barge-in, but has %d items", len(streamer.PlaybackQueue()))
	}

	// Verify that AI playing state is now false
	if streamer.IsAIPlaying() {
		t.Errorf("Streamer should no longer be marked as playing AI audio after barge-in")
	}
}

func TestAudioStreamManagerRegistration(t *testing.T) {
	mgr := NewAudioStreamManager()
	callID := "call-uuid-123"

	mockWS := NewMockWSConn()
	mockAI := NewMockAIEngine()
	cfg := StreamerConfig{CallUUID: callID}
	streamer := NewAudioStreamer(cfg, mockWS, mockAI)

	mgr.Register(callID, streamer)

	retrieved, ok := mgr.Get(callID)
	if !ok || retrieved != streamer {
		t.Fatalf("Failed to retrieve registered streamer")
	}

	mgr.HandleHangup(callID)
	_, ok = mgr.Get(callID)
	if ok {
		t.Errorf("Streamer should be removed after HandleHangup")
	}
}

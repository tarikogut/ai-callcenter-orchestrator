package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/tarikogut/ai-callcenter-orchestrator/pkg/ai"
	"github.com/tarikogut/ai-callcenter-orchestrator/pkg/diameter"
)

// FreeSwitchEvent represents audiofork / ESL events received from FreeSWITCH
type FreeSwitchEvent struct {
	EventName       string `json:"event_name"`        // e.g. "CUSTOM", "CHANNEL_ANSWER", "CHANNEL_HANGUP"
	EventSubclass   string `json:"event_subclass"`    // e.g. "mod_audio_fork::connected"
	CallUUID        string `json:"call_uuid"`         // Unique Channel UUID
	Caller          string `json:"caller_id_number"`  // Caller CLI
	DID             string `json:"destination_number"`// Inbound DID
	TenantID        string `json:"variable_tenant_id"`// Tenant identifier
	WorkflowID      string `json:"variable_workflow_id"`
	Action          string `json:"action"`            // "start", "stop", "user_speech"
	TranscriptText  string `json:"transcript_text"`   // Real-time transcribed text from user
	AudioData       string `json:"audio_data"`        // Base64 PCM data if forwarded
}

// CallControlAction represents an instruction emitted to FreeSWITCH
type CallControlAction struct {
	CallUUID    string                 `json:"call_uuid"`
	Command     string                 `json:"command"`      // "uuid_transfer", "uuid_kill", "playback", "speak"
	Target      string                 `json:"target"`       // e.g. "101 XML default"
	Params      map[string]interface{} `json:"params,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// CallWorkflowResolver function retrieves workflow definition for a given DID or tenant
type CallWorkflowResolver func(tenantID, did string) (*WorkflowDefinition, error)

// CallHandler coordinates voice call lifecycle and executes workflows
type CallHandler struct {
	engine           *Engine
	diamClient       diameter.RoClient
	workflowResolver CallWorkflowResolver
	streamManager    *ai.AudioStreamManager

	mu               sync.RWMutex
	activeCalls      map[string]*ExecutionContext // CallUUID -> ExecutionContext
	actionQueue      chan CallControlAction
}

// NewCallHandler creates a new FreeSWITCH call handler
func NewCallHandler(engine *Engine, diamClient diameter.RoClient, resolver CallWorkflowResolver) *CallHandler {
	return &CallHandler{
		engine:           engine,
		diamClient:       diamClient,
		workflowResolver: resolver,
		streamManager:    ai.NewAudioStreamManager(),
		activeCalls:      make(map[string]*ExecutionContext),
		actionQueue:      make(chan CallControlAction, 1000),
	}
}

// StreamManager returns the active audio stream manager.
func (h *CallHandler) StreamManager() *ai.AudioStreamManager {
	return h.streamManager
}

// SetStreamManager sets an external audio stream manager.
func (h *CallHandler) SetStreamManager(sm *ai.AudioStreamManager) {
	h.streamManager = sm
}

// AttachAudioStreamer registers an active audio stream and binds its barge-in control signals.
func (h *CallHandler) AttachAudioStreamer(streamer *ai.AudioStreamer) {
	if streamer == nil {
		return
	}
	if h.streamManager != nil {
		h.streamManager.Register(streamer.CallUUID(), streamer)
	}

	// Wire FreeSWITCH control actions (such as uuid_break on barge-in) to CallHandler actionQueue
	streamer.SetOnControlAction(func(act ai.CallControlAction) {
		h.actionQueue <- CallControlAction{
			CallUUID:  act.CallUUID,
			Command:   act.Command,
			Target:    act.Target,
			Params:    act.Params,
			Timestamp: act.Timestamp,
		}
	})
}

// EmitCallControl dispatches a control action to the outbound action queue.
func (h *CallHandler) EmitCallControl(action CallControlAction) {
	select {
	case h.actionQueue <- action:
	default:
	}
}

// GetActiveCalls returns currently active calls
func (h *CallHandler) GetActiveCall(callUUID string) (*ExecutionContext, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ctx, ok := h.activeCalls[callUUID]
	return ctx, ok
}

// ActionQueue provides emitted FreeSWITCH actions for outbound dispatch
func (h *CallHandler) ActionQueue() <-chan CallControlAction {
	return h.actionQueue
}

// HandleInboundCall starts a call workflow when a FreeSWITCH channel answers or audiofork connects
func (h *CallHandler) HandleInboundCall(ctx context.Context, ev FreeSwitchEvent) (*ExecutionContext, error) {
	if ev.CallUUID == "" {
		return nil, errors.New("call_uuid cannot be empty")
	}

	tenantID := ev.TenantID
	if tenantID == "" {
		tenantID = "default_tenant"
	}

	execCtx := NewExecutionContext(
		tenantID,
		ev.WorkflowID,
		"call-"+ev.CallUUID,
		TriggerTypeCall,
		ev.Caller,
		ev.DID,
	)
	execCtx.CallUUID = ev.CallUUID
	execCtx.CallState = CallStateAnswered

	if ev.TranscriptText != "" {
		execCtx.SetVariable("$last_user_speech", ev.TranscriptText)
	}

	h.mu.Lock()
	h.activeCalls[ev.CallUUID] = execCtx
	h.mu.Unlock()

	// Resolve workflow definition
	wf, err := h.workflowResolver(tenantID, ev.DID)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve workflow for DID %s: %w", ev.DID, err)
	}

	// Execute workflow
	err = h.engine.Execute(ctx, wf, execCtx)
	if err != nil {
		return execCtx, fmt.Errorf("workflow execution failed: %w", err)
	}

	// Emit appropriate call control actions based on final context state
	if execCtx.CallState == CallStateTransferred {
		target := execCtx.TargetExtension
		if target == "" && execCtx.TargetQueue != "" {
			target = execCtx.TargetQueue
		}
		if target != "" {
			h.actionQueue <- CallControlAction{
				CallUUID:  ev.CallUUID,
				Command:   "uuid_transfer",
				Target:    fmt.Sprintf("%s XML default", target),
				Timestamp: time.Now(),
			}
		}
	} else if execCtx.CallState == CallStateHangup {
		h.actionQueue <- CallControlAction{
			CallUUID:  ev.CallUUID,
			Command:   "uuid_kill",
			Target:    "NORMAL_CLEARING",
			Timestamp: time.Now(),
		}
	}

	return execCtx, nil
}

// HandleCallHangup ends active call and terminates Diameter session
func (h *CallHandler) HandleCallHangup(ctx context.Context, callUUID string, durationSec uint32) error {
	h.mu.Lock()
	execCtx, exists := h.activeCalls[callUUID]
	if exists {
		delete(h.activeCalls, callUUID)
	}
	h.mu.Unlock()

	if !exists {
		return nil
	}

	execCtx.CallState = CallStateHangup
	execCtx.IsTerminated = true

	// Terminate active audio stream session if any
	if h.streamManager != nil {
		h.streamManager.HandleHangup(callUUID)
	}

	// Terminate Diameter Ro session if client exists
	if h.diamClient != nil {
		req := diameter.CCRRequest{
			SessionID:      execCtx.SessionID,
			SubscriptionID: execCtx.TenantID,
			RatingType:     diameter.RatingTypeVoiceTime,
			UsedUnits:      durationSec,
			RatePerUnit:    0.01,
		}
		_, _ = h.diamClient.SendCCRTerminate(ctx, req)
	}

	return nil
}

// HTTPWebhookHandler returns an http.HandlerFunc to accept FreeSWITCH events over HTTP
func (h *CallHandler) HTTPWebhookHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var ev FreeSwitchEvent
		if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
			http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
			return
		}

		ctx, err := h.HandleInboundCall(r.Context(), ev)
		if err != nil {
			http.Error(w, "workflow error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "ok",
			"call_uuid":  ev.CallUUID,
			"call_state": ctx.CallState,
			"variables":  ctx.Variables,
		})
	}
}

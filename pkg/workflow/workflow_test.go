package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/tarikogut/ai-callcenter-orchestrator/pkg/diameter"
	"github.com/tarikogut/ai-callcenter-orchestrator/pkg/mcp"
	"github.com/tarikogut/ai-callcenter-orchestrator/pkg/rag"
)

type MockSmsSender struct {
	mu   sync.Mutex
	sent []KamailioOutboundSMS
}

func (m *MockSmsSender) SendSMS(ctx context.Context, outbound KamailioOutboundSMS) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, outbound)
	return nil
}

func (m *MockSmsSender) GetSent() []KamailioOutboundSMS {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]KamailioOutboundSMS(nil), m.sent...)
}

func TestVoiceCallWorkflowExecution(t *testing.T) {
	// 1. Setup Diameter client & balance
	diamClient := diameter.NewClient(diameter.ClientConfig{
		OriginHost: "orchestrator.test",
	})
	tenantID := "tenant_eczane_hayat"
	diamClient.GetBalanceStore().SetBalance(tenantID, 10.0) // 10.0 TL

	// 2. Setup Workflow DAG Engine
	engine := NewEngine(diamClient)

	// 3. Define Voice Call Workflow (Inbound -> Rating -> AI -> Condition -> Transfer)
	wf := &WorkflowDefinition{
		WorkflowID: "wf_call_eczane_01",
		TenantID:   tenantID,
		Name:       "Eczane Çağrı Akışı",
		Nodes: []NodeDefinition{
			{
				ID:   "trigger_1",
				Type: "InboundCallTrigger",
				Config: map[string]interface{}{
					"did": "08501110001",
				},
			},
			{
				ID:   "rating_1",
				Type: "DiameterRatingNode",
				Config: map[string]interface{}{
					"rate_per_min": 0.60,
					"account_id":   tenantID,
				},
			},
			{
				ID:   "agent_1",
				Type: "AiVoiceAgentNode",
				Config: map[string]interface{}{
					"agent_name": "Ayşe (Eczacı Asistanı)",
					"persona":    "Eczane asistanısın. Müşteriyi dinle ve yanıtla.",
					"filler_phrases": []interface{}{
						"Ha tamam bir saniye bakıyorum...",
					},
				},
			},
			{
				ID:   "tool_1",
				Type: "McpToolNode",
				Config: map[string]interface{}{
					"tool_name": "check_stock",
				},
			},
			{
				ID:   "cond_1",
				Type: "ConditionNode",
				Config: map[string]interface{}{
					"field":    "$tool_success",
					"operator": "eq",
					"value":    true,
				},
			},
			{
				ID:   "transfer_1",
				Type: "TransferNode",
				Config: map[string]interface{}{
					"target_extension": "101",
				},
			},
		},
		Connections: []ConnectionDefinition{
			{From: "trigger_1", To: "rating_1"},
			{From: "rating_1", To: "agent_1"},
			{From: "agent_1", To: "tool_1"},
			{From: "tool_1", To: "cond_1"},
			{From: "cond_1", To: "transfer_1", SourcePort: "true"},
		},
	}

	callHandler := NewCallHandler(engine, diamClient, func(tID, did string) (*WorkflowDefinition, error) {
		return wf, nil
	})

	ctx := context.Background()

	// 4. Fire Inbound Call event from FreeSWITCH
	fsEvent := FreeSwitchEvent{
		EventName:      "CHANNEL_ANSWER",
		CallUUID:       "uuid-test-call-1234",
		Caller:         "05321112233",
		DID:            "08501110001",
		TenantID:       tenantID,
		WorkflowID:     "wf_call_eczane_01",
		TranscriptText: "Aspirin var mı acaba?",
	}

	execCtx, err := callHandler.HandleInboundCall(ctx, fsEvent)
	if err != nil {
		t.Fatalf("HandleInboundCall returned error: %v", err)
	}

	// Verify visited nodes
	expectedNodes := []string{"trigger_1", "rating_1", "agent_1", "tool_1", "cond_1", "transfer_1"}
	if len(execCtx.VisitedNodes) != len(expectedNodes) {
		t.Fatalf("expected visited %d nodes, got %d (%v)", len(expectedNodes), len(execCtx.VisitedNodes), execCtx.VisitedNodes)
	}

	// Verify Call state & extension transfer
	if execCtx.CallState != CallStateTransferred {
		t.Errorf("expected call state TRANSFERRED, got %s", execCtx.CallState)
	}
	if execCtx.TargetExtension != "101" {
		t.Errorf("expected target extension '101', got '%s'", execCtx.TargetExtension)
	}

	// Verify action queue emitted
	select {
	case action := <-callHandler.ActionQueue():
		if action.Command != "uuid_transfer" {
			t.Errorf("expected uuid_transfer command, got %s", action.Command)
		}
		if action.Target != "101 XML default" {
			t.Errorf("expected target '101 XML default', got %s", action.Target)
		}
	default:
		t.Errorf("expected action queue to have an action emitted")
	}

	// Verify Diameter balance was debited (0.60 TL for 60s)
	bal := diamClient.GetBalanceStore().GetBalance(tenantID)
	diff := 9.40 - bal
	if diff < 0 {
		diff = -diff
	}
	if diff > 0.001 {
		t.Errorf("expected remaining balance 9.40 TL, got %.2f", bal)
	}

	// 5. Terminate call (duration 45s => 15s refund = 0.15 TL)
	err = callHandler.HandleCallHangup(ctx, fsEvent.CallUUID, 45)
	if err != nil {
		t.Fatalf("HandleCallHangup failed: %v", err)
	}

	balAfterHangup := diamClient.GetBalanceStore().GetBalance(tenantID)
	diffRefund := 9.55 - balAfterHangup
	if diffRefund < 0 {
		diffRefund = -diffRefund
	}
	if diffRefund > 0.001 {
		t.Errorf("expected remaining balance 9.55 TL after refund, got %.2f", balAfterHangup)
	}
}

func TestSmsWorkflowExecution(t *testing.T) {
	// 1. Setup Diameter client & balance
	diamClient := diameter.NewClient(diameter.ClientConfig{
		OriginHost: "orchestrator.test",
	})
	tenantID := "tenant_kargo_express"
	diamClient.GetBalanceStore().SetBalance(tenantID, 5.0) // 5.0 TL

	// 2. Setup Engine
	engine := NewEngine(diamClient)

	// 3. Define SMS Workflow (Inbound -> Rating -> AI -> SmsReply)
	wf := &WorkflowDefinition{
		WorkflowID: "wf_sms_kargo_01",
		TenantID:   tenantID,
		Name:       "Kargo Takip SMS Akışı",
		Nodes: []NodeDefinition{
			{
				ID:   "sms_trigger",
				Type: "InboundSmsTrigger",
				Config: map[string]interface{}{
					"did": "8500",
				},
			},
			{
				ID:   "sms_rating",
				Type: "DiameterRatingNode",
				Config: map[string]interface{}{
					"rate_per_sms": 0.20,
					"account_id":   tenantID,
				},
			},
			{
				ID:   "sms_agent",
				Type: "AiAgentNode",
				Config: map[string]interface{}{
					"agent_name": "Kargo Takip Asistanı",
					"persona":    "Kargo durumunu sorgula.",
				},
			},
			{
				ID:   "sms_reply",
				Type: "SmsReplyNode",
				Config: map[string]interface{}{
					"template": "Sn. {{$caller}}, {{$ai_response}}",
				},
			},
		},
		Connections: []ConnectionDefinition{
			{From: "sms_trigger", To: "sms_rating"},
			{From: "sms_rating", To: "sms_agent"},
			{From: "sms_agent", To: "sms_reply"},
		},
	}

	mockSender := &MockSmsSender{}
	smsHandler := NewSmsHandler(engine, diamClient, func(tID, did string) (*WorkflowDefinition, error) {
		return wf, nil
	}, mockSender)

	// 4. Test Webhook POST /api/v1/webhook/sms
	inboundPayload := KamailioInboundSMS{
		SMSCID:   "sim1",
		From:     "905551234567",
		To:       "8500",
		Text:     "Kargom nerede?",
		TenantID: tenantID,
		MsgID:    "MSG-9988",
	}

	payloadBytes, _ := json.Marshal(inboundPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhook/sms", bytes.NewReader(payloadBytes))
	rec := httptest.NewRecorder()

	handlerFunc := smsHandler.WebhookHandler()
	handlerFunc(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify reply was dispatched via sender
	sentList := mockSender.GetSent()
	if len(sentList) != 1 {
		t.Fatalf("expected 1 outbound SMS sent, got %d", len(sentList))
	}

	outbound := sentList[0]
	if outbound.To != "905551234567" {
		t.Errorf("expected recipient 905551234567, got %s", outbound.To)
	}
	if outbound.From != "8500" {
		t.Errorf("expected sender DID 8500, got %s", outbound.From)
	}

	// Verify Diameter SMS rate was deducted (5.0 - 0.20 = 4.80 TL)
	bal := diamClient.GetBalanceStore().GetBalance(tenantID)
	diff := 4.80 - bal
	if diff < 0 {
		diff = -diff
	}
	if diff > 0.001 {
		t.Errorf("expected remaining balance 4.80 TL, got %.2f", bal)
	}
}

type mockWorkflowMcpClient struct{}

func (m *mockWorkflowMcpClient) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	return []mcp.Tool{
		{Name: "check_stock", Description: "Check stock"},
	}, nil
}

func (m *mockWorkflowMcpClient) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcp.ToolCallResult, error) {
	return &mcp.ToolCallResult{
		Content: []mcp.ToolContent{
			{Type: "text", Text: "Stock confirmed: 42 in stock"},
		},
	}, nil
}

func (m *mockWorkflowMcpClient) Close() error {
	return nil
}

func TestWorkflowAiAgentWithRagAndMcp(t *testing.T) {
	ragEng := rag.NewInMemoryEngine()
	tenantID := "tenant_eczane_rag"
	_ = ragEng.AddDocuments(context.Background(), rag.Document{
		ID:       "faq_calisma",
		TenantID: tenantID,
		Category: "working_hours",
		Question: "Eczane kaçta açılıyor ve kapanıyor?",
		Answer:   "Eczanemiz 08:30 ile 19:00 saatleri arasında açıktır.",
	})

	mcpClient := &mockWorkflowMcpClient{}

	execCtx := NewExecutionContext(tenantID, "wf_rag_01", "sess_rag_01", TriggerTypeCall, "905551234567", "08501112233")
	execCtx.RagEngine = ragEng
	execCtx.McpClient = mcpClient
	execCtx.SetVariable("$last_user_speech", "Eczane kaçta açılıyor?")

	// Test AiAgentNode with RAG injection
	agentNode := NewAiAgentNode("agent_node", map[string]interface{}{
		"agent_name": "Ayşe Eczane Asistanı",
		"enable_rag": true,
	})

	res, err := agentNode.Execute(context.Background(), execCtx)
	if err != nil {
		t.Fatalf("AiAgentNode Execute failed: %v", err)
	}
	ragCtx, _ := res.Outputs["$rag_context"].(string)
	if ragCtx == "" || !strings.Contains(ragCtx, "08:30 ile 19:00") {
		t.Fatalf("expected RAG context injected into outputs, got: %s", ragCtx)
	}

	// Test McpToolNode with McpClient
	mcpNode := NewMcpToolNode("mcp_node", map[string]interface{}{
		"tool_name": "check_stock",
		"arguments": map[string]interface{}{"drug": "Aspirin"},
	})

	mcpRes, err := mcpNode.Execute(context.Background(), execCtx)
	if err != nil {
		t.Fatalf("McpToolNode Execute failed: %v", err)
	}
	toolSuccess, _ := mcpRes.Outputs["$tool_success"].(bool)
	if !toolSuccess {
		t.Fatalf("expected tool_success true, got false")
	}
	toolResult, _ := mcpRes.Outputs["$tool_result"].(map[string]interface{})
	if toolResult["result"] != "Stock confirmed: 42 in stock" {
		t.Fatalf("unexpected tool result: %+v", toolResult)
	}
}


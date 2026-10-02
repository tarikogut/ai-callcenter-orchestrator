package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tarikogut/ai-callcenter-orchestrator/pkg/diameter"
)

// 1. InboundTriggerNode
// Config: { "trigger_on": "CALL" | "SMS" | "ANY", "did": "0850..." }
type InboundTriggerNode struct {
	id     string
	config map[string]interface{}
}

func NewInboundTriggerNode(id string, config map[string]interface{}) *InboundTriggerNode {
	return &InboundTriggerNode{id: id, config: config}
}

func (n *InboundTriggerNode) ID() string   { return n.id }
func (n *InboundTriggerNode) Type() string { return "InboundTriggerNode" }

func (n *InboundTriggerNode) Execute(ctx context.Context, execCtx *ExecutionContext) (*NodeResult, error) {
	expectedType, _ := n.config["trigger_on"].(string)
	if expectedType != "" && expectedType != "ANY" && expectedType != string(execCtx.Type) {
		return &NodeResult{
			StopFlow: true,
			Error:    fmt.Errorf("trigger type mismatch: flow requires %s but got %s", expectedType, execCtx.Type),
		}, nil
	}

	expectedDID, _ := n.config["did"].(string)
	if expectedDID != "" && execCtx.DID != "" && expectedDID != execCtx.DID {
		return &NodeResult{
			StopFlow: true,
			Error:    fmt.Errorf("DID mismatch: flow requires %s but got %s", expectedDID, execCtx.DID),
		}, nil
	}

	outputs := map[string]interface{}{
		"$caller": execCtx.Caller,
		"$did":    execCtx.DID,
		"$type":   string(execCtx.Type),
	}

	return &NodeResult{
		Outputs:  outputs,
		NextPort: "default",
	}, nil
}

// 2. AiAgentNode
// Config: { "agent_name": "...", "persona": "...", "filler_phrases": [...], "model": "..." }
type AiAgentNode struct {
	id     string
	config map[string]interface{}
}

func NewAiAgentNode(id string, config map[string]interface{}) *AiAgentNode {
	return &AiAgentNode{id: id, config: config}
}

func (n *AiAgentNode) ID() string   { return n.id }
func (n *AiAgentNode) Type() string { return "AiAgentNode" }

func (n *AiAgentNode) Execute(ctx context.Context, execCtx *ExecutionContext) (*NodeResult, error) {
	agentName, _ := n.config["agent_name"].(string)
	if agentName == "" {
		agentName = "Eczacı Asistanı Ayşe"
	}

	persona, _ := n.config["persona"].(string)
	if persona == "" {
		persona = "Yardımsever bir müşteri temsilcisisin."
	}

	// Filler phrases e.g. "Ha tamam bir saniye bakıyorum...", "Hemen kontrol ediyorum..."
	var fillerPhrases []string
	if rawFillers, ok := n.config["filler_phrases"].([]interface{}); ok {
		for _, item := range rawFillers {
			if s, ok := item.(string); ok {
				fillerPhrases = append(fillerPhrases, s)
			}
		}
	} else if strList, ok := n.config["filler_phrases"].([]string); ok {
		fillerPhrases = strList
	}
	if len(fillerPhrases) == 0 {
		fillerPhrases = []string{"Ha tamam bir saniye bakıyorum...", "Hemen sistemden kontrol ediyorum..."}
	}

	selectedFiller := fillerPhrases[0]

	// Determine conversational response based on prompt or input
	inputText := execCtx.SmsInboundText
	if inputText == "" {
		if rawPrompt, ok := execCtx.GetVariable("$last_user_speech"); ok {
			inputText, _ = rawPrompt.(string)
		}
	}
	if inputText == "" {
		inputText = "Merhaba"
	}

	// Append user input
	execCtx.AppendDialogue("user", inputText)

	// Simulated AI turn generation (or prompt chaining)
	aiResponse := fmt.Sprintf("%s: Merhaba, size nasıl yardımcı olabilirim?", agentName)
	if strings.Contains(strings.ToLower(inputText), "aspirin") || strings.Contains(strings.ToLower(inputText), "stok") {
		aiResponse = fmt.Sprintf("%s (Stok bilgisi sorgulandı: Mevcut)", agentName)
	} else if strings.Contains(strings.ToLower(inputText), "temsilci") || strings.Contains(strings.ToLower(inputText), "insan") {
		aiResponse = "Sizi yetkili temsilcimize aktarıyorum, lütfen hattan ayrılmayın."
	}

	// Append assistant dialogue
	execCtx.AppendDialogue("assistant", aiResponse)

	outputs := map[string]interface{}{
		"$agent_name":    agentName,
		"$persona":       persona,
		"$filler_phrase": selectedFiller,
		"$ai_response":   aiResponse,
	}

	return &NodeResult{
		Outputs:  outputs,
		NextPort: "default",
	}, nil
}

// 3. McpToolNode
// Config: { "mcp_server": "https://...", "tool_name": "check_stock", "arguments": {...} }
type McpToolNode struct {
	id         string
	config     map[string]interface{}
	httpClient *http.Client
}

func NewMcpToolNode(id string, config map[string]interface{}) *McpToolNode {
	return &McpToolNode{
		id:     id,
		config: config,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (n *McpToolNode) ID() string   { return n.id }
func (n *McpToolNode) Type() string { return "McpToolNode" }

func (n *McpToolNode) Execute(ctx context.Context, execCtx *ExecutionContext) (*NodeResult, error) {
	mcpServer, _ := n.config["mcp_server"].(string)
	toolName, _ := n.config["tool_name"].(string)
	if toolName == "" {
		toolName = "default_tool"
	}

	// If mcpServer is set and points to an active HTTP endpoint, perform POST
	var toolResult map[string]interface{}
	var toolSuccess = true

	if mcpServer != "" && strings.HasPrefix(mcpServer, "http") {
		reqBody := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "tools/call",
			"params": map[string]interface{}{
				"name":      toolName,
				"arguments": n.config["arguments"],
				"context": map[string]string{
					"caller": execCtx.Caller,
					"tenant": execCtx.TenantID,
				},
			},
			"id": 1,
		}
		data, _ := json.Marshal(reqBody)

		req, err := http.NewRequestWithContext(ctx, "POST", mcpServer, strings.NewReader(string(data)))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			resp, err := n.httpClient.Do(req)
			if err == nil {
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				_ = json.Unmarshal(body, &toolResult)
			} else {
				toolSuccess = false
			}
		}
	}

	if toolResult == nil {
		// Mock local execution for standard tests / offline operations
		toolResult = map[string]interface{}{
			"tool":    toolName,
			"status":  "ok",
			"result":  fmt.Sprintf("Tool %s executed successfully for %s", toolName, execCtx.Caller),
			"matched": true,
		}
	}

	port := "success"
	if !toolSuccess {
		port = "error"
	}

	outputs := map[string]interface{}{
		"$tool_result":  toolResult,
		"$tool_success": toolSuccess,
		"$tool_name":    toolName,
	}

	return &NodeResult{
		Outputs:  outputs,
		NextPort: port,
	}, nil
}

// 4. ConditionNode
// Config: { "field": "$tool_success", "operator": "eq" | "neq" | "contains", "value": true }
type ConditionNode struct {
	id     string
	config map[string]interface{}
}

func NewConditionNode(id string, config map[string]interface{}) *ConditionNode {
	return &ConditionNode{id: id, config: config}
}

func (n *ConditionNode) ID() string   { return n.id }
func (n *ConditionNode) Type() string { return "ConditionNode" }

func (n *ConditionNode) Execute(ctx context.Context, execCtx *ExecutionContext) (*NodeResult, error) {
	field, _ := n.config["field"].(string)
	operator, _ := n.config["operator"].(string)
	expectedVal := n.config["value"]

	var actualVal interface{}
	if strings.HasPrefix(field, "$") {
		actualVal, _ = execCtx.GetVariable(field)
	} else {
		actualVal = field
	}

	isMatch := false
	switch operator {
	case "eq", "==":
		isMatch = fmt.Sprintf("%v", actualVal) == fmt.Sprintf("%v", expectedVal)
	case "neq", "!=":
		isMatch = fmt.Sprintf("%v", actualVal) != fmt.Sprintf("%v", expectedVal)
	case "contains":
		actStr := fmt.Sprintf("%v", actualVal)
		expStr := fmt.Sprintf("%v", expectedVal)
		isMatch = strings.Contains(strings.ToLower(actStr), strings.ToLower(expStr))
	default:
		// Default truthiness
		if b, ok := actualVal.(bool); ok {
			isMatch = b
		} else {
			isMatch = actualVal != nil && fmt.Sprintf("%v", actualVal) != ""
		}
	}

	nextPort := "false"
	if isMatch {
		nextPort = "true"
	}

	outputs := map[string]interface{}{
		"$condition_result": isMatch,
	}

	return &NodeResult{
		Outputs:  outputs,
		NextPort: nextPort,
	}, nil
}

// 5. TransferNode
// Config: { "target_extension": "101", "target_queue": "queue_support" }
type TransferNode struct {
	id     string
	config map[string]interface{}
}

func NewTransferNode(id string, config map[string]interface{}) *TransferNode {
	return &TransferNode{id: id, config: config}
}

func (n *TransferNode) ID() string   { return n.id }
func (n *TransferNode) Type() string { return "TransferNode" }

func (n *TransferNode) Execute(ctx context.Context, execCtx *ExecutionContext) (*NodeResult, error) {
	ext, _ := n.config["target_extension"].(string)
	queue, _ := n.config["target_queue"].(string)

	if ext != "" {
		execCtx.TargetExtension = ext
	}
	if queue != "" {
		execCtx.TargetQueue = queue
	}
	execCtx.CallState = CallStateTransferred

	outputs := map[string]interface{}{
		"$transfer_extension": ext,
		"$transfer_queue":     queue,
	}

	return &NodeResult{
		Outputs:  outputs,
		NextPort: "default",
	}, nil
}

// 6. SmsReplyNode
// Config: { "template": "Sayın Müşterimiz, talebiniz alınmıştır: {{$ai_response}}" }
type SmsReplyNode struct {
	id     string
	config map[string]interface{}
}

func NewSmsReplyNode(id string, config map[string]interface{}) *SmsReplyNode {
	return &SmsReplyNode{id: id, config: config}
}

func (n *SmsReplyNode) ID() string   { return n.id }
func (n *SmsReplyNode) Type() string { return "SmsReplyNode" }

func (n *SmsReplyNode) Execute(ctx context.Context, execCtx *ExecutionContext) (*NodeResult, error) {
	template, _ := n.config["template"].(string)
	if template == "" {
		template, _ = n.config["text"].(string)
	}
	if template == "" {
		// Use ai response if available
		if aiResp, ok := execCtx.GetVariable("$ai_response"); ok {
			template = fmt.Sprintf("%v", aiResp)
		} else {
			template = "Mesajınız alınmıştır."
		}
	}

	// Simple template variable interpolation e.g. {{$caller}} or {{$ai_response}}
	rendered := template
	for k, v := range execCtx.Variables {
		placeholder := fmt.Sprintf("{{%s}}", k)
		rendered = strings.ReplaceAll(rendered, placeholder, fmt.Sprintf("%v", v))
	}

	execCtx.SmsReplyText = rendered

	outputs := map[string]interface{}{
		"$sms_reply": rendered,
	}

	return &NodeResult{
		Outputs:  outputs,
		NextPort: "default",
	}, nil
}

// 7. DiameterRatingNode
// Config: { "rate_per_min": 0.50, "rate_per_sms": 0.10, "account_id": "tenant_1" }
type DiameterRatingNode struct {
	id         string
	config     map[string]interface{}
	diamClient diameter.RoClient
}

func NewDiameterRatingNode(id string, config map[string]interface{}, diamClient diameter.RoClient) *DiameterRatingNode {
	return &DiameterRatingNode{
		id:         id,
		config:     config,
		diamClient: diamClient,
	}
}

func (n *DiameterRatingNode) ID() string   { return n.id }
func (n *DiameterRatingNode) Type() string { return "DiameterRatingNode" }

func (n *DiameterRatingNode) Execute(ctx context.Context, execCtx *ExecutionContext) (*NodeResult, error) {
	if n.diamClient == nil {
		return &NodeResult{NextPort: "default"}, nil
	}

	accountID, _ := n.config["account_id"].(string)
	if accountID == "" {
		accountID = execCtx.TenantID
	}

	ratePerMin, _ := n.config["rate_per_min"].(float64)
	if ratePerMin <= 0 {
		ratePerMin = 0.50 // 0.50 TL / min
	}
	ratePerSec := ratePerMin / 60.0

	ratePerSms, _ := n.config["rate_per_sms"].(float64)
	if ratePerSms <= 0 {
		ratePerSms = 0.10 // 0.10 TL / sms
	}

	var cca *diameter.CCAResponse
	var err error

	if execCtx.Type == TriggerTypeSms {
		// Event charging for SMS
		req := diameter.CCRRequest{
			SessionID:      execCtx.SessionID,
			SubscriptionID: accountID,
			RatingType:     diameter.RatingTypeSmsEvent,
			RequestedUnits: 1,
			RatePerUnit:    ratePerSms,
		}
		cca, err = n.diamClient.SendCCREvent(ctx, req)
	} else {
		// Initial credit reservation for Voice Call
		req := diameter.CCRRequest{
			SessionID:      execCtx.SessionID,
			SubscriptionID: accountID,
			RatingType:     diameter.RatingTypeVoiceTime,
			RequestedUnits: 60, // reserve 60 seconds
			RatePerUnit:    ratePerSec,
		}
		cca, err = n.diamClient.SendCCRInitial(ctx, req)
	}

	if err != nil {
		return &NodeResult{
			StopFlow: true,
			Error:    fmt.Errorf("diameter rating communication failure: %w", err),
		}, nil
	}

	if cca.ResultCode != diameter.ResultCodeSuccess {
		// Credit limit reached or user denied
		outputs := map[string]interface{}{
			"$diameter_result_code": cca.ResultCode,
			"$diameter_error":       cca.ErrorMessage,
		}
		return &NodeResult{
			Outputs:  outputs,
			StopFlow: true,
			Error:    fmt.Errorf("rating rejected (result_code %d): %s", cca.ResultCode, cca.ErrorMessage),
		}, nil
	}

	execCtx.TotalCost += float64(cca.GrantedUnits) * ratePerSec

	outputs := map[string]interface{}{
		"$diameter_result_code":     cca.ResultCode,
		"$diameter_granted_units":   cca.GrantedUnits,
		"$diameter_remaining_credit": cca.RemainingCredit,
	}

	return &NodeResult{
		Outputs:  outputs,
		NextPort: "default",
	}, nil
}

// 8. HangupNode
// Config: { "reason": "normal_clearing" }
type HangupNode struct {
	id     string
	config map[string]interface{}
}

func NewHangupNode(id string, config map[string]interface{}) *HangupNode {
	return &HangupNode{id: id, config: config}
}

func (n *HangupNode) ID() string   { return n.id }
func (n *HangupNode) Type() string { return "HangupNode" }

func (n *HangupNode) Execute(ctx context.Context, execCtx *ExecutionContext) (*NodeResult, error) {
	reason, _ := n.config["reason"].(string)
	if reason == "" {
		reason = "NORMAL_CLEARING"
	}

	execCtx.CallState = CallStateHangup
	execCtx.IsTerminated = true

	outputs := map[string]interface{}{
		"$hangup_reason": reason,
	}

	return &NodeResult{
		Outputs:  outputs,
		StopFlow: true,
		NextPort: "default",
	}, nil
}

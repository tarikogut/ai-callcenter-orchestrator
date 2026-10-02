package workflow

import (
	"context"
	"sync"
	"time"
)

// TriggerType indicates the channel origin
type TriggerType string

const (
	TriggerTypeCall TriggerType = "CALL"
	TriggerTypeSms  TriggerType = "SMS"
)

// CallState tracks real-time voice call progression
type CallState string

const (
	CallStateRinging     CallState = "RINGING"
	CallStateAnswered    CallState = "ANSWERED"
	CallStateTransferred CallState = "TRANSFERRED"
	CallStateHangup      CallState = "HANGUP"
)

// ExecutionContext holds variables and channel states for a workflow execution instance
type ExecutionContext struct {
	mu sync.RWMutex

	// Identity & Channel
	WorkflowID string
	TenantID   string
	SessionID  string
	Type       TriggerType // "CALL" or "SMS"
	Caller     string      // Inbound phone number
	DID        string      // Inbound dialed number (DID / Shortcode)

	// Context Variables ($caller, $did, $type, custom outputs)
	Variables map[string]interface{}

	// Call Specifics
	CallState       CallState
	CallUUID        string
	TargetExtension string // e.g. "101"
	TargetQueue     string // e.g. "queue_support"

	// SMS Specifics
	SmsInboundText string
	SmsReplyText   string

	// Dialogue / Audio
	DialogueHistory []MessageTurn
	LastAudioBase64 string
	LastTextPrompt  string

	// Node Execution Trace
	VisitedNodes []string
	NodeOutputs  map[string]map[string]interface{}

	// Diameter Rating State
	TotalCost float64
	Currency  string

	// Knowledge Base & MCP
	RagEngine interface{}
	McpClient interface{}

	// Control flags
	IsTerminated bool
}

// MessageTurn represents a conversational message
type MessageTurn struct {
	Role      string    `json:"role"` // "user", "assistant", "system"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// NewExecutionContext initializes an execution context
func NewExecutionContext(tenantID, workflowID, sessionID string, trigType TriggerType, caller, did string) *ExecutionContext {
	ctx := &ExecutionContext{
		TenantID:        tenantID,
		WorkflowID:      workflowID,
		SessionID:       sessionID,
		Type:            trigType,
		Caller:          caller,
		DID:             did,
		Variables:       make(map[string]interface{}),
		VisitedNodes:    make([]string, 0),
		NodeOutputs:     make(map[string]map[string]interface{}),
		DialogueHistory: make([]MessageTurn, 0),
		Currency:        "TRY",
	}

	// Pre-populate core workflow parameters
	ctx.Variables["$caller"] = caller
	ctx.Variables["$did"] = did
	ctx.Variables["$type"] = string(trigType)
	ctx.Variables["$tenant_id"] = tenantID
	ctx.Variables["$session_id"] = sessionID

	return ctx
}

func (c *ExecutionContext) SetVariable(key string, val interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Variables[key] = val
}

func (c *ExecutionContext) GetVariable(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.Variables[key]
	return val, ok
}

func (c *ExecutionContext) SetNodeOutput(nodeID string, outputs map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.NodeOutputs[nodeID] = outputs
	for k, v := range outputs {
		c.Variables[k] = v
	}
}

func (c *ExecutionContext) AddVisited(nodeID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.VisitedNodes = append(c.VisitedNodes, nodeID)
}

func (c *ExecutionContext) AppendDialogue(role, content string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.DialogueHistory = append(c.DialogueHistory, MessageTurn{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
}

// NodeResult is returned by Node.Execute
type NodeResult struct {
	Outputs     map[string]interface{} // Variables produced by this node
	NextPort    string                 // Output branch or port (e.g. "default", "true", "false", "success", "error")
	StopFlow    bool                   // If true, stop workflow execution immediately
	Error       error                  // Execution error if any
}

// Node is the interface implemented by all workflow block nodes
type Node interface {
	ID() string
	Type() string
	Execute(ctx context.Context, execCtx *ExecutionContext) (*NodeResult, error)
}

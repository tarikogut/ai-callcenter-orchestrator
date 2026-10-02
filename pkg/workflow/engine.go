package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/tarikogut/ai-callcenter-orchestrator/pkg/diameter"
)

// WorkflowDefinition models the JSON structure designed for visual flow builders (ReactFlow / VueFlow / n8n)
type WorkflowDefinition struct {
	WorkflowID     string                 `json:"workflow_id"`
	TenantID       string                 `json:"tenant_id"`
	Name           string                 `json:"name"`
	TenantSettings map[string]interface{} `json:"tenant_settings,omitempty"`
	Nodes          []NodeDefinition       `json:"nodes"`
	Connections    []ConnectionDefinition `json:"connections"`
}

// NodeDefinition models single node in JSON workflow
type NodeDefinition struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"` // e.g. "InboundTriggerNode", "AiVoiceAgentNode", etc.
	Config map[string]interface{} `json:"config"`
}

// ConnectionDefinition models edges between nodes with optional source/target ports
type ConnectionDefinition struct {
	From       string `json:"from"`
	To         string `json:"to"`
	SourcePort string `json:"source_port,omitempty"` // "default", "true", "false", "success", "error"
	TargetPort string `json:"target_port,omitempty"`
}

// Engine parses workflow JSON and executes DAG flows
type Engine struct {
	mu          sync.RWMutex
	diamClient  diameter.RoClient
	nodeFactory map[string]func(id string, config map[string]interface{}) Node
}

// NewEngine creates a new Workflow DAG Engine
func NewEngine(diamClient diameter.RoClient) *Engine {
	e := &Engine{
		diamClient:  diamClient,
		nodeFactory: make(map[string]func(id string, config map[string]interface{}) Node),
	}
	e.registerStandardNodes()
	return e
}

func (e *Engine) registerStandardNodes() {
	// Register both standard names and friendly alias names
	e.nodeFactory["InboundTriggerNode"] = func(id string, cfg map[string]interface{}) Node {
		return NewInboundTriggerNode(id, cfg)
	}
	e.nodeFactory["InboundCallTrigger"] = func(id string, cfg map[string]interface{}) Node {
		return NewInboundTriggerNode(id, cfg)
	}
	e.nodeFactory["InboundSmsTrigger"] = func(id string, cfg map[string]interface{}) Node {
		if cfg == nil {
			cfg = make(map[string]interface{})
		}
		cfg["trigger_on"] = "SMS"
		return NewInboundTriggerNode(id, cfg)
	}

	e.nodeFactory["AiAgentNode"] = func(id string, cfg map[string]interface{}) Node {
		return NewAiAgentNode(id, cfg)
	}
	e.nodeFactory["AiVoiceAgentNode"] = func(id string, cfg map[string]interface{}) Node {
		return NewAiAgentNode(id, cfg)
	}

	e.nodeFactory["McpToolNode"] = func(id string, cfg map[string]interface{}) Node {
		return NewMcpToolNode(id, cfg)
	}

	e.nodeFactory["ConditionNode"] = func(id string, cfg map[string]interface{}) Node {
		return NewConditionNode(id, cfg)
	}

	e.nodeFactory["TransferNode"] = func(id string, cfg map[string]interface{}) Node {
		return NewTransferNode(id, cfg)
	}

	e.nodeFactory["SmsReplyNode"] = func(id string, cfg map[string]interface{}) Node {
		return NewSmsReplyNode(id, cfg)
	}

	e.nodeFactory["DiameterRatingNode"] = func(id string, cfg map[string]interface{}) Node {
		return NewDiameterRatingNode(id, cfg, e.diamClient)
	}

	e.nodeFactory["HangupNode"] = func(id string, cfg map[string]interface{}) Node {
		return NewHangupNode(id, cfg)
	}
}

// RegisterCustomNode allows external plugins or custom actions to extend node catalog
func (e *Engine) RegisterCustomNode(nodeType string, factory func(id string, config map[string]interface{}) Node) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.nodeFactory[nodeType] = factory
}

// InstantiateNode creates a concrete Node instance from NodeDefinition
func (e *Engine) InstantiateNode(def NodeDefinition) (Node, error) {
	e.mu.RLock()
	factory, ok := e.nodeFactory[def.Type]
	e.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown node type: %s", def.Type)
	}
	return factory(def.ID, def.Config), nil
}

// ParseWorkflow parses a raw JSON byte slice into WorkflowDefinition
func ParseWorkflow(data []byte) (*WorkflowDefinition, error) {
	var wf WorkflowDefinition
	if err := json.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("invalid workflow JSON: %w", err)
	}
	if len(wf.Nodes) == 0 {
		return nil, errors.New("workflow contains no nodes")
	}
	return &wf, nil
}

// Execute runs the workflow definition on the provided ExecutionContext
func (e *Engine) Execute(ctx context.Context, wf *WorkflowDefinition, execCtx *ExecutionContext) error {
	if wf == nil || len(wf.Nodes) == 0 {
		return errors.New("empty workflow definition")
	}

	// Index nodes by ID
	nodeMap := make(map[string]Node)
	for _, nDef := range wf.Nodes {
		node, err := e.InstantiateNode(nDef)
		if err != nil {
			return fmt.Errorf("failed to instantiate node %s (%s): %w", nDef.ID, nDef.Type, err)
		}
		nodeMap[nDef.ID] = node
	}

	// Build adjacency lookup: fromNodeID -> list of connections
	outgoing := make(map[string][]ConnectionDefinition)
	incomingCount := make(map[string]int)
	for _, conn := range wf.Connections {
		outgoing[conn.From] = append(outgoing[conn.From], conn)
		incomingCount[conn.To]++
	}

	// Determine starting node:
	// Preferred: first InboundTriggerNode or root node with 0 incoming edges
	var startNodeID string
	for _, nDef := range wf.Nodes {
		if strings.Contains(strings.ToLower(nDef.Type), "trigger") {
			startNodeID = nDef.ID
			break
		}
	}
	if startNodeID == "" {
		for _, nDef := range wf.Nodes {
			if incomingCount[nDef.ID] == 0 {
				startNodeID = nDef.ID
				break
			}
		}
	}
	if startNodeID == "" {
		startNodeID = wf.Nodes[0].ID
	}

	// Sequential / Branching DAG Execution Loop
	currentNodeID := startNodeID
	maxSteps := 100 // Prevent infinite loops in cyclic flows
	step := 0

	for currentNodeID != "" && step < maxSteps {
		step++
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		node, ok := nodeMap[currentNodeID]
		if !ok {
			return fmt.Errorf("node not found in workflow: %s", currentNodeID)
		}

		execCtx.AddVisited(currentNodeID)

		result, err := node.Execute(ctx, execCtx)
		if err != nil {
			return fmt.Errorf("error executing node %s (%s): %w", currentNodeID, node.Type(), err)
		}

		if result != nil {
			if len(result.Outputs) > 0 {
				execCtx.SetNodeOutput(currentNodeID, result.Outputs)
			}
			if result.StopFlow {
				if result.Error != nil {
					return result.Error
				}
				break
			}
		}

		// Determine next node
		nextPort := "default"
		if result != nil && result.NextPort != "" {
			nextPort = result.NextPort
		}

		conns := outgoing[currentNodeID]
		var nextNodeID string

		// First try matching connection by source port
		for _, c := range conns {
			if c.SourcePort == nextPort {
				nextNodeID = c.To
				break
			}
		}

		// Fallback to connection without specific source port or default
		if nextNodeID == "" && len(conns) > 0 {
			for _, c := range conns {
				if c.SourcePort == "" || c.SourcePort == "default" {
					nextNodeID = c.To
					break
				}
			}
		}

		// If still empty but only 1 connection exists, use it
		if nextNodeID == "" && len(conns) == 1 {
			nextNodeID = conns[0].To
		}

		currentNodeID = nextNodeID
	}

	return nil
}

package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// JSON-RPC 2.0 protocol structures
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *JSONRPCError) Error() string {
	return fmt.Sprintf("MCP JSON-RPC error %d: %s", e.Code, e.Message)
}

// MCP Protocol Definitions
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema ToolInputSchema        `json:"inputSchema"`
}

type ToolInputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type ToolCallResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ToolContent struct {
	Type string `json:"type"` // "text", "image", "resource"
	Text string `json:"text,omitempty"`
	Data string `json:"data,omitempty"`
	Mime string `json:"mimeType,omitempty"`
}

// Client defines the MCP client interface
type Client interface {
	ListTools(ctx context.Context) ([]Tool, error)
	CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*ToolCallResult, error)
	Close() error
}

// ClientTransport defines the communication mode: HTTP, SSE, STDIO
type TransportType string

const (
	TransportHTTP  TransportType = "HTTP"
	TransportSSE   TransportType = "SSE"
	TransportSTDIO TransportType = "STDIO"
)

// ClientConfig holds configuration for initializing an MCP Client
type ClientConfig struct {
	Transport   TransportType     `json:"transport"`
	Endpoint    string            `json:"endpoint,omitempty"` // For HTTP / SSE
	Headers     map[string]string `json:"headers,omitempty"`  // Auth tokens, Tenant ID, etc.
	Command     string            `json:"command,omitempty"`  // For STDIO executable
	Args        []string          `json:"args,omitempty"`     // For STDIO executable args
	Timeout     time.Duration     `json:"timeout,omitempty"`
}

// Base client implementation
type mcpClient struct {
	config     ClientConfig
	httpClient *http.Client
	requestID  uint64

	// For STDIO
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	mu     sync.Mutex
}

// NewClient creates an MCP client supporting HTTP, SSE, or STDIO
func NewClient(cfg ClientConfig) (Client, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}

	c := &mcpClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}

	if cfg.Transport == TransportSTDIO {
		if cfg.Command == "" {
			return nil, fmt.Errorf("stdio transport requires a command")
		}
		cmd := exec.Command(cfg.Command, cfg.Args...)
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to open stdin pipe: %w", err)
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			stdin.Close()
			return nil, fmt.Errorf("failed to open stdout pipe: %w", err)
		}
		if err := cmd.Start(); err != nil {
			stdin.Close()
			return nil, fmt.Errorf("failed to start stdio command: %w", err)
		}
		c.cmd = cmd
		c.stdin = stdin
		c.stdout = bufio.NewReader(stdout)
	}

	return c, nil
}

func (c *mcpClient) nextID() uint64 {
	return atomic.AddUint64(&c.requestID, 1)
}

func (c *mcpClient) ListTools(ctx context.Context) ([]Tool, error) {
	resp, err := c.sendRPC(ctx, "tools/list", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	var listResult ToolsListResult
	if err := json.Unmarshal(resp.Result, &listResult); err != nil {
		return nil, fmt.Errorf("failed to parse tools/list result: %w", err)
	}
	return listResult.Tools, nil
}

func (c *mcpClient) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*ToolCallResult, error) {
	if arguments == nil {
		arguments = make(map[string]interface{})
	}
	params := ToolCallParams{
		Name:      name,
		Arguments: arguments,
	}

	resp, err := c.sendRPC(ctx, "tools/call", params)
	if err != nil {
		return nil, err
	}

	var callResult ToolCallResult
	if err := json.Unmarshal(resp.Result, &callResult); err != nil {
		return nil, fmt.Errorf("failed to parse tools/call result: %w", err)
	}
	return &callResult, nil
}

func (c *mcpClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cmd != nil {
		if c.stdin != nil {
			_ = c.stdin.Close()
		}
		if c.cmd.Process != nil {
			_ = c.cmd.Process.Kill()
		}
		_ = c.cmd.Wait()
	}
	return nil
}

func (c *mcpClient) sendRPC(ctx context.Context, method string, params interface{}) (*JSONRPCResponse, error) {
	reqID := c.nextID()
	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      reqID,
		Method:  method,
		Params:  params,
	}

	switch c.config.Transport {
	case TransportHTTP, TransportSSE:
		return c.sendHTTP(ctx, reqBody)
	case TransportSTDIO:
		return c.sendSTDIO(ctx, reqBody)
	default:
		return c.sendHTTP(ctx, reqBody)
	}
}

func (c *mcpClient) sendHTTP(ctx context.Context, rpcReq JSONRPCRequest) (*JSONRPCResponse, error) {
	payload, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.Transport == TransportSSE {
		req.Header.Set("Accept", "text/event-stream, application/json")
	} else {
		req.Header.Set("Accept", "application/json")
	}

	for k, v := range c.config.Headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("http error %d: %s", resp.StatusCode, string(body))
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		return c.readSSEResponse(resp.Body, rpcReq.ID)
	}

	var rpcResp JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("decode json-rpc response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, rpcResp.Error
	}

	return &rpcResp, nil
}

func (c *mcpClient) readSSEResponse(r io.Reader, expectedID interface{}) (*JSONRPCResponse, error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "" || data == "[DONE]" {
				continue
			}
			var rpcResp JSONRPCResponse
			if err := json.Unmarshal([]byte(data), &rpcResp); err == nil {
				if fmt.Sprintf("%v", rpcResp.ID) == fmt.Sprintf("%v", expectedID) || rpcResp.ID == nil {
					if rpcResp.Error != nil {
						return nil, rpcResp.Error
					}
					return &rpcResp, nil
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("sse stream read error: %w", err)
	}
	return nil, fmt.Errorf("sse stream ended without response for request id %v", expectedID)
}

func (c *mcpClient) sendSTDIO(ctx context.Context, rpcReq JSONRPCRequest) (*JSONRPCResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	payload, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("marshal stdio request: %w", err)
	}

	// Write newline-delimited JSON
	if _, err := c.stdin.Write(append(payload, '\n')); err != nil {
		return nil, fmt.Errorf("write stdio: %w", err)
	}

	type result struct {
		resp *JSONRPCResponse
		err  error
	}

	resChan := make(chan result, 1)

	go func() {
		line, err := c.stdout.ReadString('\n')
		if err != nil {
			resChan <- result{nil, fmt.Errorf("read stdio: %w", err)}
			return
		}
		var rpcResp JSONRPCResponse
		if err := json.Unmarshal([]byte(line), &rpcResp); err != nil {
			resChan <- result{nil, fmt.Errorf("parse stdio response: %w", err)}
			return
		}
		if rpcResp.Error != nil {
			resChan <- result{nil, rpcResp.Error}
			return
		}
		resChan <- result{&rpcResp, nil}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-resChan:
		return res.resp, res.err
	}
}

// LLM Schema Conversions

// GeminiFunctionDeclaration represents Google Gemini tool declaration
type GeminiFunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// GeminiTool represents Gemini tool wrapping function declarations
type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"functionDeclarations"`
}

// OpenAIFunction represents OpenAI tool declaration
type OpenAIFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type OpenAITool struct {
	Type     string         `json:"type"` // "function"
	Function OpenAIFunction `json:"function"`
}

// ConvertToGeminiTool converts MCP Tools to Gemini tool declaration format
func ConvertToGeminiTool(tools []Tool) GeminiTool {
	declarations := make([]GeminiFunctionDeclaration, 0, len(tools))
	for _, t := range tools {
		params := map[string]interface{}{
			"type": t.InputSchema.Type,
		}
		if t.InputSchema.Type == "" {
			params["type"] = "OBJECT"
		} else {
			params["type"] = strings.ToUpper(t.InputSchema.Type)
		}
		if t.InputSchema.Properties != nil {
			params["properties"] = t.InputSchema.Properties
		}
		if len(t.InputSchema.Required) > 0 {
			params["required"] = t.InputSchema.Required
		}

		declarations = append(declarations, GeminiFunctionDeclaration{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  params,
		})
	}
	return GeminiTool{FunctionDeclarations: declarations}
}

// ConvertToOpenAITools converts MCP Tools to OpenAI tool declaration format
func ConvertToOpenAITools(tools []Tool) []OpenAITool {
	result := make([]OpenAITool, 0, len(tools))
	for _, t := range tools {
		params := map[string]interface{}{
			"type": "object",
		}
		if t.InputSchema.Type != "" {
			params["type"] = strings.ToLower(t.InputSchema.Type)
		}
		if t.InputSchema.Properties != nil {
			params["properties"] = t.InputSchema.Properties
		}
		if len(t.InputSchema.Required) > 0 {
			params["required"] = t.InputSchema.Required
		}

		result = append(result, OpenAITool{
			Type: "function",
			Function: OpenAIFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}
	return result
}

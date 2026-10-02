package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMCPClientHTTP(t *testing.T) {
	// Setup test HTTP MCP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "tools/list":
			res := ToolsListResult{
				Tools: []Tool{
					{
						Name:        "check_stock",
						Description: "Check drug stock status",
						InputSchema: ToolInputSchema{
							Type: "object",
							Properties: map[string]interface{}{
								"drug_name": map[string]interface{}{"type": "string"},
							},
							Required: []string{"drug_name"},
						},
					},
				},
			}
			rawRes, _ := json.Marshal(res)
			json.NewEncoder(w).Encode(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  rawRes,
			})
		case "tools/call":
			paramsBytes, _ := json.Marshal(req.Params)
			var callParams ToolCallParams
			_ = json.Unmarshal(paramsBytes, &callParams)

			var res ToolCallResult
			if callParams.Name == "check_stock" {
				res = ToolCallResult{
					Content: []ToolContent{
						{
							Type: "text",
							Text: fmt.Sprintf("Stock for %v: 15 boxes available", callParams.Arguments["drug_name"]),
						},
					},
				}
			} else {
				res = ToolCallResult{
					IsError: true,
					Content: []ToolContent{
						{Type: "text", Text: "Unknown tool"},
					},
				}
			}
			rawRes, _ := json.Marshal(res)
			json.NewEncoder(w).Encode(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  rawRes,
			})
		default:
			json.NewEncoder(w).Encode(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &JSONRPCError{Code: -32601, Message: "Method not found"},
			})
		}
	}))
	defer ts.Close()

	client, err := NewClient(ClientConfig{
		Transport: TransportHTTP,
		Endpoint:  ts.URL,
		Timeout:   3 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 1. Test ListTools
	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "check_stock" {
		t.Fatalf("unexpected tools returned: %+v", tools)
	}

	// 2. Test CallTool
	callRes, err := client.CallTool(ctx, "check_stock", map[string]interface{}{
		"drug_name": "Parol 500mg",
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if len(callRes.Content) == 0 || callRes.Content[0].Text != "Stock for Parol 500mg: 15 boxes available" {
		t.Fatalf("unexpected call result: %+v", callRes)
	}

	// 3. Test LLM Schema conversions
	geminiTool := ConvertToGeminiTool(tools)
	if len(geminiTool.FunctionDeclarations) != 1 || geminiTool.FunctionDeclarations[0].Name != "check_stock" {
		t.Fatalf("unexpected Gemini tool conversion: %+v", geminiTool)
	}

	openaiTools := ConvertToOpenAITools(tools)
	if len(openaiTools) != 1 || openaiTools[0].Function.Name != "check_stock" {
		t.Fatalf("unexpected OpenAI tool conversion: %+v", openaiTools)
	}
}

func TestMCPClientSSE(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")

		res := ToolsListResult{
			Tools: []Tool{
				{Name: "get_duty_pharmacy", Description: "Returns on-duty pharmacies"},
			},
		}
		rawRes, _ := json.Marshal(res)
		rpcResp := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  rawRes,
		}
		data, _ := json.Marshal(rpcResp)
		fmt.Fprintf(w, "event: message\ndata: %s\n\n", string(data))
	}))
	defer ts.Close()

	client, err := NewClient(ClientConfig{
		Transport: TransportSSE,
		Endpoint:  ts.URL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools failed over SSE: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "get_duty_pharmacy" {
		t.Fatalf("unexpected tools returned over SSE: %+v", tools)
	}
}

func TestMCPClientSTDIO(t *testing.T) {
	// Create a simple mock script or binary using go run or a temporary script
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "mock_stdio.sh")
	scriptContent := `#!/bin/sh
while IFS= read -r line; do
  echo "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"tools\":[{\"name\":\"track_shipment\"}]}}"
done
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	client, err := NewClient(ClientConfig{
		Transport: TransportSTDIO,
		Command:   scriptPath,
	})
	if err != nil {
		t.Fatalf("failed to create stdio client: %v", err)
	}
	defer client.Close()

	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("STDIO ListTools failed: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "track_shipment" {
		t.Fatalf("unexpected STDIO tools: %+v", tools)
	}
}

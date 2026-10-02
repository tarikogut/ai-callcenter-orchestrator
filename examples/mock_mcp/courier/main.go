package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/tarikogut/ai-callcenter-orchestrator/pkg/mcp"
)

type Shipment struct {
	TrackingNumber string `json:"tracking_number"`
	Recipient      string `json:"recipient"`
	Status         string `json:"status"` // "Dağıtımda", "Teslim Edildi", "Şubede Bekliyor"
	CurrentCity    string `json:"current_city"`
	DeliveryDate   string `json:"delivery_date"`
}

type CourierStore struct {
	mu        sync.RWMutex
	shipments map[string]*Shipment
}

var courierDB = &CourierStore{
	shipments: map[string]*Shipment{
		"KP012345678": {
			TrackingNumber: "KP012345678",
			Recipient:      "Ahmet Yılmaz",
			Status:         "Dağıtımda (Kurye Dağıtım Aracında)",
			CurrentCity:    "Kadıköy / İstanbul",
			DeliveryDate:   time.Now().Format("02.01.2006"),
		},
		"KP987654321": {
			TrackingNumber: "KP987654321",
			Recipient:      "Ayşe Demir",
			Status:         "Şubede Bekliyor (Teslim Alınabilir)",
			CurrentCity:    "Şişli / İstanbul",
			DeliveryDate:   time.Now().AddDate(0, 0, 1).Format("02.01.2006"),
		},
	},
}

func handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req mcp.JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch req.Method {
	case "tools/list":
		res := mcp.ToolsListResult{
			Tools: []mcp.Tool{
				{
					Name:        "track_shipment",
					Description: "Track package status, delivery location, and expected arrival date by tracking number",
					InputSchema: mcp.ToolInputSchema{
						Type: "object",
						Properties: map[string]interface{}{
							"tracking_number": map[string]interface{}{
								"type":        "string",
								"description": "Tracking code (e.g. KP012345678)",
							},
						},
						Required: []string{"tracking_number"},
					},
				},
				{
					Name:        "reschedule_delivery",
					Description: "Reschedule shipment delivery date to a preferred date",
					InputSchema: mcp.ToolInputSchema{
						Type: "object",
						Properties: map[string]interface{}{
							"tracking_number": map[string]interface{}{
								"type":        "string",
								"description": "Tracking code of shipment",
							},
							"new_date": map[string]interface{}{
								"type":        "string",
								"description": "Requested delivery date (e.g. DD.MM.YYYY)",
							},
						},
						Required: []string{"tracking_number", "new_date"},
					},
				},
			},
		}
		rawRes, _ := json.Marshal(res)
		_ = json.NewEncoder(w).Encode(mcp.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  rawRes,
		})

	case "tools/call":
		paramsBytes, _ := json.Marshal(req.Params)
		var callParams mcp.ToolCallParams
		_ = json.Unmarshal(paramsBytes, &callParams)

		var toolResult mcp.ToolCallResult

		switch callParams.Name {
		case "track_shipment":
			trackNo, _ := callParams.Arguments["tracking_number"].(string)
			trackKey := strings.ToUpper(strings.TrimSpace(trackNo))

			courierDB.mu.RLock()
			shipment, found := courierDB.shipments[trackKey]
			courierDB.mu.RUnlock()

			if !found {
				toolResult = mcp.ToolCallResult{
					Content: []mcp.ToolContent{
						{
							Type: "text",
							Text: fmt.Sprintf("'%s' takip numaralı kargo kaydı bulunamadı. Lütfen kontrol edip tekrar deneyiniz.", trackNo),
						},
					},
				}
			} else {
				toolResult = mcp.ToolCallResult{
					Content: []mcp.ToolContent{
						{
							Type: "text",
							Text: fmt.Sprintf("Kargo Takip: %s | Alıcı: %s | Durum: %s | Konum: %s | Tahmini Teslim: %s",
								shipment.TrackingNumber, shipment.Recipient, shipment.Status, shipment.CurrentCity, shipment.DeliveryDate),
						},
					},
				}
			}

		case "reschedule_delivery":
			trackNo, _ := callParams.Arguments["tracking_number"].(string)
			newDate, _ := callParams.Arguments["new_date"].(string)
			trackKey := strings.ToUpper(strings.TrimSpace(trackNo))

			courierDB.mu.Lock()
			shipment, found := courierDB.shipments[trackKey]
			if found {
				shipment.DeliveryDate = newDate
			}
			courierDB.mu.Unlock()

			if !found {
				toolResult = mcp.ToolCallResult{
					Content: []mcp.ToolContent{
						{
							Type: "text",
							Text: fmt.Sprintf("'%s' takip numaralı kargo bulunamadı. Tarih güncellenemedi.", trackNo),
						},
					},
				}
			} else {
				toolResult = mcp.ToolCallResult{
					Content: []mcp.ToolContent{
						{
							Type: "text",
							Text: fmt.Sprintf("Kargo %s için teslimat tarihi başarıyla '%s' olarak güncellendi.", trackNo, newDate),
						},
					},
				}
			}

		default:
			toolResult = mcp.ToolCallResult{
				IsError: true,
				Content: []mcp.ToolContent{
					{
						Type: "text",
						Text: fmt.Sprintf("Bilinmeyen araç: %s", callParams.Name),
					},
				},
			}
		}

		rawRes, _ := json.Marshal(toolResult)
		_ = json.NewEncoder(w).Encode(mcp.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  rawRes,
		})

	default:
		_ = json.NewEncoder(w).Encode(mcp.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &mcp.JSONRPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Method '%s' not found", req.Method),
			},
		})
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9092"
	}

	http.HandleFunc("/mcp", handleRPC)
	http.HandleFunc("/", handleRPC)

	log.Printf("Courier Mock MCP Server listening on :%s (endpoints: / or /mcp)", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}

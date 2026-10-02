package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/tarikogut/ai-callcenter-orchestrator/pkg/mcp"
)

type StockItem struct {
	Stock    int
	Price    float64
	RequiresRx bool
}

var stockDB = map[string]StockItem{
	"parol":       {Stock: 45, Price: 38.50, RequiresRx: false},
	"aspirin":     {Stock: 12, Price: 45.00, RequiresRx: false},
	"amoklavin":   {Stock: 6, Price: 125.00, RequiresRx: true},
	"arveles":     {Stock: 28, Price: 62.00, RequiresRx: false},
	"augmentin":   {Stock: 0, Price: 140.00, RequiresRx: true},
}

type DutyPharmacy struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
	Dist    string `json:"district"`
}

var dutyList = []DutyPharmacy{
	{
		Name:    "Hayat Eczanesi",
		Address: "Bağdat Caddesi No: 142/A Kadıköy / İstanbul",
		Phone:   "0216 350 11 22",
		Dist:    "Kadıköy",
	},
	{
		Name:    "Şifa Nöbetçi Eczanesi",
		Address: "Moda Caddesi No: 88 Kadıköy / İstanbul",
		Phone:   "0216 336 44 55",
		Dist:    "Kadıköy",
	},
	{
		Name:    "Merkez Nöbetçi Eczanesi",
		Address: "Halaskargazi Caddesi No: 45 Şişli / İstanbul",
		Phone:   "0212 240 77 88",
		Dist:    "Şişli",
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
					Name:        "check_stock",
					Description: "Check drug stock availability, price, and prescription requirement in pharmacy",
					InputSchema: mcp.ToolInputSchema{
						Type: "object",
						Properties: map[string]interface{}{
							"drug_name": map[string]interface{}{
								"type":        "string",
								"description": "Name of the medication / drug (e.g. Parol, Aspirin, Arveles)",
							},
						},
						Required: []string{"drug_name"},
					},
				},
				{
					Name:        "get_duty_pharmacy",
					Description: "Find on-duty open pharmacies for night and weekend hours by district or city",
					InputSchema: mcp.ToolInputSchema{
						Type: "object",
						Properties: map[string]interface{}{
							"district": map[string]interface{}{
								"type":        "string",
								"description": "District or neighborhood (e.g. Kadıköy, Şişli)",
							},
						},
						Required: []string{"district"},
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
		case "check_stock":
			drugName, _ := callParams.Arguments["drug_name"].(string)
			drugKey := strings.ToLower(strings.TrimSpace(drugName))
			item, found := stockDB[drugKey]
			if !found {
				toolResult = mcp.ToolCallResult{
					Content: []mcp.ToolContent{
						{
							Type: "text",
							Text: fmt.Sprintf("İlaç '%s' stoklarımızda bulunmamaktadır veya adı eşleşmedi.", drugName),
						},
					},
				}
			} else {
				rxStr := "Reçetesiz temin edilebilir"
				if item.RequiresRx {
					rxStr = "Reçeteye tabidir (SGK / Doktor Reçetesi gerekli)"
				}
				stockStr := fmt.Sprintf("%d kutu mevcut", item.Stock)
				if item.Stock == 0 {
					stockStr = "Tükendi (Stokta kalmadı)"
				}
				toolResult = mcp.ToolCallResult{
					Content: []mcp.ToolContent{
						{
							Type: "text",
							Text: fmt.Sprintf("İlaç: %s | Durum: %s | Fiyat: %.2f TL | %s", drugName, stockStr, item.Price, rxStr),
						},
					},
				}
			}

		case "get_duty_pharmacy":
			dist, _ := callParams.Arguments["district"].(string)
			queryDist := strings.ToLower(strings.TrimSpace(dist))
			var matched []DutyPharmacy
			for _, dp := range dutyList {
				if strings.Contains(strings.ToLower(dp.Dist), queryDist) || queryDist == "" {
					matched = append(matched, dp)
				}
			}
			if len(matched) == 0 {
				toolResult = mcp.ToolCallResult{
					Content: []mcp.ToolContent{
						{
							Type: "text",
							Text: fmt.Sprintf("'%s' bölgesi için nöbetçi eczane kaydı bulunamadı.", dist),
						},
					},
				}
			} else {
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("%s Bölgesi Nöbetçi Eczaneleri:\n", dist))
				for _, dp := range matched {
					sb.WriteString(fmt.Sprintf("- %s: %s | Tel: %s\n", dp.Name, dp.Address, dp.Phone))
				}
				toolResult = mcp.ToolCallResult{
					Content: []mcp.ToolContent{
						{
							Type: "text",
							Text: sb.String(),
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
		port = "9091"
	}

	http.HandleFunc("/mcp", handleRPC)
	http.HandleFunc("/", handleRPC)

	log.Printf("Pharmacy Mock MCP Server listening on :%s (endpoints: / or /mcp)", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}

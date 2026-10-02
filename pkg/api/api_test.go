package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tarikogut/ai-callcenter-orchestrator/pkg/api"
	"github.com/tarikogut/ai-callcenter-orchestrator/pkg/db"
)

func setupTestApp(t *testing.T) *api.APIHandler {
	database, err := db.InitTestDB()
	if err != nil {
		t.Fatalf("Failed to initialize test DB: %v", err)
	}
	return api.NewAPIHandler(database)
}

func TestHealthCheck(t *testing.T) {
	database, err := db.InitTestDB()
	if err != nil {
		t.Fatalf("InitTestDB error: %v", err)
	}
	app := api.SetupApp(database)

	req := httptest.NewRequest("GET", "/healthz", nil)
	resp, err := app.Test(req, 2000)
	if err != nil {
		t.Fatalf("Health check request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestTenantEndpoints(t *testing.T) {
	database, err := db.InitTestDB()
	if err != nil {
		t.Fatalf("InitTestDB error: %v", err)
	}
	app := api.SetupApp(database)

	// 1. List Tenants (should contain seeded eczane_hayat and jet_kargo)
	req := httptest.NewRequest("GET", "/api/v1/admin/tenants", nil)
	resp, err := app.Test(req, 2000)
	if err != nil {
		t.Fatalf("List tenants failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var tenants []db.Tenant
	json.Unmarshal(body, &tenants)
	if len(tenants) < 2 {
		t.Fatalf("Expected at least 2 seeded tenants, got %d", len(tenants))
	}

	// 2. Create Tenant
	createPayload := map[string]string{
		"name":      "Medipol Klinik",
		"subdomain": "medipol_klinik",
	}
	payloadBytes, _ := json.Marshal(createPayload)
	req = httptest.NewRequest("POST", "/api/v1/admin/tenants", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, 2000)
	if err != nil {
		t.Fatalf("Create tenant failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201 Created, got %d", resp.StatusCode)
	}

	var createdTenant db.Tenant
	body, _ = io.ReadAll(resp.Body)
	json.Unmarshal(body, &createdTenant)
	if createdTenant.ID == "" || createdTenant.Subdomain != "medipol_klinik" {
		t.Fatalf("Invalid created tenant: %+v", createdTenant)
	}

	// 3. Get Tenant by ID or Subdomain
	req = httptest.NewRequest("GET", "/api/v1/admin/tenants/"+createdTenant.Subdomain, nil)
	resp, err = app.Test(req, 2000)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Get tenant by subdomain failed: status=%d", resp.StatusCode)
	}

	// 4. Update Tenant
	updatePayload := map[string]string{
		"name": "Medipol Sağlık Grubu",
	}
	payloadBytes, _ = json.Marshal(updatePayload)
	req = httptest.NewRequest("PUT", "/api/v1/admin/tenants/"+createdTenant.ID, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, 2000)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Update tenant failed: status=%d", resp.StatusCode)
	}

	var updatedTenant db.Tenant
	body, _ = io.ReadAll(resp.Body)
	json.Unmarshal(body, &updatedTenant)
	if updatedTenant.Name != "Medipol Sağlık Grubu" {
		t.Errorf("Expected updated name, got %s", updatedTenant.Name)
	}

	// 5. Delete Tenant
	req = httptest.NewRequest("DELETE", "/api/v1/admin/tenants/"+createdTenant.ID, nil)
	resp, err = app.Test(req, 2000)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Delete tenant failed: status=%d", resp.StatusCode)
	}
}

func TestExtensionEndpoints(t *testing.T) {
	database, err := db.InitTestDB()
	if err != nil {
		t.Fatalf("InitTestDB error: %v", err)
	}
	app := api.SetupApp(database)

	tenantID := "tenant-eczane-hayat"

	// 1. Create Extension
	extPayload := map[string]any{
		"tenant_id":      tenantID,
		"ext_number":     "102",
		"name":           "Depo Yetkilisi",
		"password":       "depo102pass",
		"webrtc_enabled": true,
	}
	payloadBytes, _ := json.Marshal(extPayload)
	req := httptest.NewRequest("POST", "/api/v1/customer/extensions", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 2000)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("Create extension failed: status=%d", resp.StatusCode)
	}

	var createdExt db.Extension
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &createdExt)

	// 2. List Extensions with X-Tenant-ID header
	req = httptest.NewRequest("GET", "/api/v1/customer/extensions", nil)
	req.Header.Set("X-Tenant-ID", tenantID)
	resp, err = app.Test(req, 2000)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("List extensions failed: status=%d", resp.StatusCode)
	}
	var extensions []db.Extension
	body, _ = io.ReadAll(resp.Body)
	json.Unmarshal(body, &extensions)
	if len(extensions) < 2 { // 101 seeded + 102 created
		t.Fatalf("Expected at least 2 extensions for tenant, got %d", len(extensions))
	}

	// 3. Update Extension
	newName := "Depo & Sevkiyat"
	updatePayload := map[string]any{
		"name": &newName,
	}
	payloadBytes, _ = json.Marshal(updatePayload)
	req = httptest.NewRequest("PUT", "/api/v1/customer/extensions/"+createdExt.ID, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, 2000)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Update extension failed: status=%d", resp.StatusCode)
	}

	// 4. Delete Extension
	req = httptest.NewRequest("DELETE", "/api/v1/customer/extensions/"+createdExt.ID, nil)
	resp, err = app.Test(req, 2000)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Delete extension failed: status=%d", resp.StatusCode)
	}
}

func TestDIDEndpoints(t *testing.T) {
	database, err := db.InitTestDB()
	if err != nil {
		t.Fatalf("InitTestDB error: %v", err)
	}
	app := api.SetupApp(database)

	tenantID := "tenant-jet-kargo"

	// 1. Create DID
	didPayload := map[string]any{
		"tenant_id": tenantID,
		"number":    "08502229999",
	}
	payloadBytes, _ := json.Marshal(didPayload)
	req := httptest.NewRequest("POST", "/api/v1/customer/dids", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 2000)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("Create DID failed: status=%d", resp.StatusCode)
	}

	var createdDID db.DID
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &createdDID)

	// 2. Get DID
	req = httptest.NewRequest("GET", "/api/v1/customer/dids/"+createdDID.ID, nil)
	resp, err = app.Test(req, 2000)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Get DID failed: status=%d", resp.StatusCode)
	}

	// 3. Update DID
	newNumber := "08502228888"
	updatePayload := map[string]any{"number": &newNumber}
	payloadBytes, _ = json.Marshal(updatePayload)
	req = httptest.NewRequest("PUT", "/api/v1/customer/dids/"+createdDID.ID, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, 2000)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Update DID failed: status=%d", resp.StatusCode)
	}

	// 4. Delete DID
	req = httptest.NewRequest("DELETE", "/api/v1/customer/dids/"+createdDID.ID, nil)
	resp, err = app.Test(req, 2000)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Delete DID failed: status=%d", resp.StatusCode)
	}
}

func TestWorkflowEndpoints(t *testing.T) {
	database, err := db.InitTestDB()
	if err != nil {
		t.Fatalf("InitTestDB error: %v", err)
	}
	app := api.SetupApp(database)

	tenantID := "tenant-eczane-hayat"

	// 1. Create Workflow
	wfPayload := map[string]any{
		"tenant_id":    tenantID,
		"name":         "Gece Nöbetçi Akışı",
		"trigger_type": "CALL",
		"definition":   `{"nodes":[{"id":"node_1","type":"PlayAudioNode"}]}`,
		"is_active":    true,
	}
	payloadBytes, _ := json.Marshal(wfPayload)
	req := httptest.NewRequest("POST", "/api/v1/customer/workflows", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 2000)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("Create workflow failed: status=%d", resp.StatusCode)
	}

	var createdWf db.Workflow
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &createdWf)

	// 2. List Workflows
	req = httptest.NewRequest("GET", "/api/v1/customer/workflows?tenant_id="+tenantID, nil)
	resp, err = app.Test(req, 2000)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("List workflows failed: status=%d", resp.StatusCode)
	}

	// 3. Update Workflow
	isActive := false
	updatePayload := map[string]any{"is_active": &isActive}
	payloadBytes, _ = json.Marshal(updatePayload)
	req = httptest.NewRequest("PUT", "/api/v1/customer/workflows/"+createdWf.ID, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, 2000)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Update workflow failed: status=%d", resp.StatusCode)
	}

	// 4. Delete Workflow
	req = httptest.NewRequest("DELETE", "/api/v1/customer/workflows/"+createdWf.ID, nil)
	resp, err = app.Test(req, 2000)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Delete workflow failed: status=%d", resp.StatusCode)
	}
}

func TestTenantSettingsEndpoints(t *testing.T) {
	database, err := db.InitTestDB()
	if err != nil {
		t.Fatalf("InitTestDB error: %v", err)
	}
	app := api.SetupApp(database)

	tenantID := "tenant-eczane-hayat"

	// 1. Get Settings for seeded tenant
	req := httptest.NewRequest("GET", "/api/v1/customer/settings", nil)
	req.Header.Set("X-Tenant-ID", tenantID)
	resp, err := app.Test(req, 2000)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Get settings failed: status=%d", resp.StatusCode)
	}

	var settings db.TenantSettings
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &settings)
	if settings.PersonaName != "Ayşe (Eczacı Asistanı)" {
		t.Errorf("Expected persona name 'Ayşe (Eczacı Asistanı)', got %s", settings.PersonaName)
	}

	// 2. Update Settings (BYOK, Persona, Filler Phrases)
	updatePayload := map[string]any{
		"api_key":        "sk-new-gemini-byok-key",
		"provider":       "Gemini",
		"persona_name":   "Ayşe Hanım",
		"active_voice_id": "Puck-Enhanced",
		"filler_phrases": []string{"Hemen bakıyorum efendim...", "Reçetenizi inceliyorum..."},
		"mcp_server_url": "http://localhost:8001/mcp/v2",
	}
	payloadBytes, _ := json.Marshal(updatePayload)
	req = httptest.NewRequest("POST", "/api/v1/customer/settings", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)
	resp, err = app.Test(req, 2000)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Upsert settings failed: status=%d", resp.StatusCode)
	}

	var updatedSettings db.TenantSettings
	body, _ = io.ReadAll(resp.Body)
	json.Unmarshal(body, &updatedSettings)
	if updatedSettings.ApiKey != "sk-new-gemini-byok-key" {
		t.Errorf("Expected updated API key, got %s", updatedSettings.ApiKey)
	}
	if updatedSettings.PersonaName != "Ayşe Hanım" {
		t.Errorf("Expected updated persona name, got %s", updatedSettings.PersonaName)
	}
}

func TestCDREndpoints(t *testing.T) {
	database, err := db.InitTestDB()
	if err != nil {
		t.Fatalf("InitTestDB error: %v", err)
	}
	app := api.SetupApp(database)

	tenantID := "tenant-jet-kargo"

	// 1. Create CDR
	cdrPayload := map[string]any{
		"tenant_id":       tenantID,
		"call_id":         "call-uuid-998877",
		"caller":          "+905321112233",
		"callee":          "08502220002",
		"type":            "CALL",
		"duration_sec":    95,
		"status":          "ANSWERED",
		"cost":            1.45,
		"transcript_text": "Kargo durum sorgulaması yapıldı, teslimat şubede.",
		"recording_url":   "https://storage.example.com/recordings/call-998877.mp3",
	}
	payloadBytes, _ := json.Marshal(cdrPayload)
	req := httptest.NewRequest("POST", "/api/v1/customer/cdr", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 2000)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("Create CDR failed: status=%d", resp.StatusCode)
	}

	var createdCDR db.CDR
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &createdCDR)

	// 2. List CDRs with query params
	req = httptest.NewRequest("GET", "/api/v1/customer/cdr?tenant_id="+tenantID+"&status=ANSWERED", nil)
	resp, err = app.Test(req, 2000)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("List CDRs failed: status=%d", resp.StatusCode)
	}

	var listResult struct {
		Total  int      `json:"total"`
		CDRs   []db.CDR `json:"cdrs"`
	}
	body, _ = io.ReadAll(resp.Body)
	json.Unmarshal(body, &listResult)
	if listResult.Total == 0 || len(listResult.CDRs) == 0 {
		t.Fatalf("Expected at least 1 CDR in list, got %+v", listResult)
	}

	// 3. Get CDR by Call ID
	req = httptest.NewRequest("GET", "/api/v1/customer/cdr/call-uuid-998877", nil)
	resp, err = app.Test(req, 2000)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Get CDR by call_id failed: status=%d", resp.StatusCode)
	}
}

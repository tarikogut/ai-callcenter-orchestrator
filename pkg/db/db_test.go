package db_test

import (
	"testing"
	"time"

	"github.com/tarikogut/ai-callcenter-orchestrator/pkg/db"
)

func TestInitTestDBAndSeeding(t *testing.T) {
	database, err := db.InitTestDB()
	if err != nil {
		t.Fatalf("Failed to initialize test DB: %v", err)
	}

	// Verify Eczane Hayat
	var eczane db.Tenant
	if err := database.Preload("Settings").Preload("Extensions").Preload("DIDs").Preload("Workflows").
		Where("subdomain = ?", "eczane_hayat").First(&eczane).Error; err != nil {
		t.Fatalf("Failed to find seeded eczane_hayat: %v", err)
	}

	if eczane.Name != "Hayat Eczanesi" {
		t.Errorf("Expected name 'Hayat Eczanesi', got '%s'", eczane.Name)
	}
	if eczane.Settings == nil || eczane.Settings.PersonaName != "Ayşe (Eczacı Asistanı)" {
		t.Errorf("Expected persona 'Ayşe (Eczacı Asistanı)', got %+v", eczane.Settings)
	}
	if len(eczane.Extensions) == 0 || eczane.Extensions[0].ExtNumber != "101" {
		t.Errorf("Expected extension 101, got %+v", eczane.Extensions)
	}
	if len(eczane.DIDs) == 0 || eczane.DIDs[0].Number != "08501110001" {
		t.Errorf("Expected DID 08501110001, got %+v", eczane.DIDs)
	}

	// Verify Jet Kargo
	var kargo db.Tenant
	if err := database.Preload("Settings").Preload("Extensions").Preload("DIDs").Preload("Workflows").
		Where("subdomain = ?", "jet_kargo").First(&kargo).Error; err != nil {
		t.Fatalf("Failed to find seeded jet_kargo: %v", err)
	}

	if kargo.Name != "Jet Kargo" {
		t.Errorf("Expected name 'Jet Kargo', got '%s'", kargo.Name)
	}
	if kargo.Settings == nil || kargo.Settings.PersonaName != "Can (Kargo Temsilcisi)" {
		t.Errorf("Expected persona 'Can (Kargo Temsilcisi)', got %+v", kargo.Settings)
	}
	if len(kargo.Extensions) == 0 || kargo.Extensions[0].ExtNumber != "201" {
		t.Errorf("Expected extension 201, got %+v", kargo.Extensions)
	}
	if len(kargo.DIDs) == 0 || kargo.DIDs[0].Number != "08502220002" {
		t.Errorf("Expected DID 08502220002, got %+v", kargo.DIDs)
	}
}

func TestModelCRUDOperations(t *testing.T) {
	database, err := db.InitTestDB()
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}

	// 1. Create Tenant
	newTenant := db.Tenant{
		Name:      "Test Klinik",
		Subdomain: "test_klinik",
	}
	if err := database.Create(&newTenant).Error; err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	if newTenant.ID == "" {
		t.Fatalf("Expected non-empty tenant UUID, got empty")
	}

	// 2. Create TenantSettings
	settings := db.TenantSettings{
		TenantID:      newTenant.ID,
		ApiKey:        "sk-test-klinik-key",
		Provider:      "OpenAI",
		PersonaName:   "Dr. Asistan",
		PersonaPrompt: "Klinik randevu asistanı",
		ActiveVoiceID: "Alloy",
		McpServerURL:  "http://localhost:8003/mcp",
		FillerPhrases: []string{"Randevuları kontrol ediyorum...", "Lütfen hatta kalın..."},
	}
	if err := database.Create(&settings).Error; err != nil {
		t.Fatalf("Failed to create settings: %v", err)
	}

	// 3. Create Extension
	ext := db.Extension{
		TenantID:      newTenant.ID,
		ExtNumber:     "301",
		Name:          "Klinik Danışma",
		Password:      "klinik301",
		WebRTCEnabled: true,
	}
	if err := database.Create(&ext).Error; err != nil {
		t.Fatalf("Failed to create extension: %v", err)
	}

	// 4. Create Workflow
	wf := db.Workflow{
		TenantID:    newTenant.ID,
		Name:        "Randevu Alma Akışı",
		TriggerType: "CALL",
		Definition:  `{"nodes":[]}`,
		IsActive:    true,
	}
	if err := database.Create(&wf).Error; err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// 5. Create DID
	did := db.DID{
		TenantID:           newTenant.ID,
		Number:             "08503330003",
		AssignedWorkflowID: wf.ID,
	}
	if err := database.Create(&did).Error; err != nil {
		t.Fatalf("Failed to create DID: %v", err)
	}

	// 6. Create CDR
	cdr := db.CDR{
		TenantID:       newTenant.ID,
		CallID:         "call-uuid-12345",
		Caller:         "+905551112233",
		Callee:         "08503330003",
		Type:           "CALL",
		DurationSec:    125,
		Status:         "ANSWERED",
		Cost:           0.75,
		TranscriptText: "Müşteri randevu oluşturdu.",
		RecordingURL:   "https://storage.example.com/recordings/call-12345.wav",
		CreatedAt:      time.Now(),
	}
	if err := database.Create(&cdr).Error; err != nil {
		t.Fatalf("Failed to create CDR: %v", err)
	}

	// Read & Verify
	var retrievedTenant db.Tenant
	if err := database.Preload("Settings").Preload("Extensions").Preload("DIDs").Preload("Workflows").Preload("CDRs").
		First(&retrievedTenant, "id = ?", newTenant.ID).Error; err != nil {
		t.Fatalf("Failed to read tenant: %v", err)
	}

	if retrievedTenant.Name != "Test Klinik" {
		t.Errorf("Expected name 'Test Klinik', got %s", retrievedTenant.Name)
	}
	if len(retrievedTenant.Extensions) != 1 || retrievedTenant.Extensions[0].ExtNumber != "301" {
		t.Errorf("Expected extension 301, got %+v", retrievedTenant.Extensions)
	}
	if len(retrievedTenant.DIDs) != 1 || retrievedTenant.DIDs[0].Number != "08503330003" {
		t.Errorf("Expected DID 08503330003, got %+v", retrievedTenant.DIDs)
	}
	if len(retrievedTenant.Workflows) != 1 || retrievedTenant.Workflows[0].Name != "Randevu Alma Akışı" {
		t.Errorf("Expected workflow 'Randevu Alma Akışı', got %+v", retrievedTenant.Workflows)
	}
	if len(retrievedTenant.CDRs) != 1 || retrievedTenant.CDRs[0].CallID != "call-uuid-12345" {
		t.Errorf("Expected CDR with call_id 'call-uuid-12345', got %+v", retrievedTenant.CDRs)
	}

	// Update Workflow
	database.Model(&wf).Update("Name", "Güncellenmiş Randevu Akışı")
	var updatedWf db.Workflow
	database.First(&updatedWf, "id = ?", wf.ID)
	if updatedWf.Name != "Güncellenmiş Randevu Akışı" {
		t.Errorf("Expected updated workflow name, got %s", updatedWf.Name)
	}

	// Delete DID
	if err := database.Delete(&did).Error; err != nil {
		t.Fatalf("Failed to delete DID: %v", err)
	}
	var count int64
	database.Model(&db.DID{}).Where("id = ?", did.ID).Count(&count)
	if count != 0 {
		t.Errorf("Expected DID count 0 after deletion, got %d", count)
	}
}

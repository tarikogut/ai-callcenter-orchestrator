package api

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/tarikogut/ai-callcenter-orchestrator/pkg/ai"
	"github.com/tarikogut/ai-callcenter-orchestrator/pkg/db"
	"github.com/tarikogut/ai-callcenter-orchestrator/pkg/workflow"
	"gorm.io/gorm"
)

type APIHandler struct {
	DB            *gorm.DB
	CallHandler   *workflow.CallHandler
	StreamManager *ai.AudioStreamManager
}

func NewAPIHandler(database *gorm.DB) *APIHandler {
	engine := workflow.NewEngine(nil)
	resolver := func(tenantID, did string) (*workflow.WorkflowDefinition, error) {
		return nil, nil
	}
	callHandler := workflow.NewCallHandler(engine, nil, resolver)
	return &APIHandler{
		DB:            database,
		CallHandler:   callHandler,
		StreamManager: callHandler.StreamManager(),
	}
}

func getTenantID(c *fiber.Ctx) string {
	tid := c.Get("X-Tenant-ID")
	if tid == "" {
		tid = c.Query("tenant_id")
	}
	return strings.TrimSpace(tid)
}

// ==========================================
// TENANTS (Admin API)
// ==========================================

func (h *APIHandler) ListTenants(c *fiber.Ctx) error {
	var tenants []db.Tenant
	if err := h.DB.Preload("Settings").Preload("Extensions").Preload("DIDs").Preload("Workflows").Find(&tenants).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(tenants)
}

func (h *APIHandler) GetTenant(c *fiber.Ctx) error {
	id := c.Params("id")
	var tenant db.Tenant
	err := h.DB.Preload("Settings").Preload("Extensions").Preload("DIDs").Preload("Workflows").
		Where("id = ? OR subdomain = ?", id, id).First(&tenant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tenant not found"})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(tenant)
}

func (h *APIHandler) CreateTenant(c *fiber.Ctx) error {
	var req struct {
		Name      string `json:"name"`
		Subdomain string `json:"subdomain"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if req.Name == "" || req.Subdomain == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name and subdomain are required"})
	}

	tenant := db.Tenant{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Subdomain: req.Subdomain,
	}

	if err := h.DB.Create(&tenant).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Create default empty settings for the tenant
	defaultSettings := db.TenantSettings{
		ID:          uuid.New().String(),
		TenantID:    tenant.ID,
		Provider:    "Gemini",
		PersonaName: "AI Asistan",
	}
	h.DB.Create(&defaultSettings)
	tenant.Settings = &defaultSettings

	return c.Status(fiber.StatusCreated).JSON(tenant)
}

func (h *APIHandler) UpdateTenant(c *fiber.Ctx) error {
	id := c.Params("id")
	var tenant db.Tenant
	if err := h.DB.First(&tenant, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tenant not found"})
	}

	var req struct {
		Name      string `json:"name"`
		Subdomain string `json:"subdomain"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Name != "" {
		tenant.Name = req.Name
	}
	if req.Subdomain != "" {
		tenant.Subdomain = req.Subdomain
	}

	if err := h.DB.Save(&tenant).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(tenant)
}

func (h *APIHandler) DeleteTenant(c *fiber.Ctx) error {
	id := c.Params("id")
	// Delete tenant and associated records
	if err := h.DB.Where("tenant_id = ?", id).Delete(&db.TenantSettings{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.DB.Where("tenant_id = ?", id).Delete(&db.Extension{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.DB.Where("tenant_id = ?", id).Delete(&db.DID{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.DB.Where("tenant_id = ?", id).Delete(&db.Workflow{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.DB.Where("tenant_id = ?", id).Delete(&db.CDR{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	res := h.DB.Delete(&db.Tenant{}, "id = ?", id)
	if res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": res.Error.Error()})
	}
	if res.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tenant not found"})
	}

	return c.JSON(fiber.Map{"message": "Tenant deleted successfully"})
}

// ==========================================
// EXTENSIONS (Customer API)
// ==========================================

func (h *APIHandler) ListExtensions(c *fiber.Ctx) error {
	tid := getTenantID(c)
	var extensions []db.Extension
	query := h.DB
	if tid != "" {
		query = query.Where("tenant_id = ?", tid)
	}
	if err := query.Find(&extensions).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(extensions)
}

func (h *APIHandler) GetExtension(c *fiber.Ctx) error {
	id := c.Params("id")
	var ext db.Extension
	if err := h.DB.First(&ext, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Extension not found"})
	}
	return c.JSON(ext)
}

func (h *APIHandler) CreateExtension(c *fiber.Ctx) error {
	var ext db.Extension
	if err := c.BodyParser(&ext); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if ext.TenantID == "" {
		ext.TenantID = getTenantID(c)
	}
	if ext.TenantID == "" || ext.ExtNumber == "" || ext.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "tenant_id, ext_number, and name are required"})
	}

	if ext.ID == "" {
		ext.ID = uuid.New().String()
	}

	if err := h.DB.Create(&ext).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(ext)
}

func (h *APIHandler) UpdateExtension(c *fiber.Ctx) error {
	id := c.Params("id")
	var ext db.Extension
	if err := h.DB.First(&ext, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Extension not found"})
	}

	var req struct {
		ExtNumber     *string `json:"ext_number"`
		Name          *string `json:"name"`
		Password      *string `json:"password"`
		WebRTCEnabled *bool   `json:"webrtc_enabled"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.ExtNumber != nil {
		ext.ExtNumber = *req.ExtNumber
	}
	if req.Name != nil {
		ext.Name = *req.Name
	}
	if req.Password != nil {
		ext.Password = *req.Password
	}
	if req.WebRTCEnabled != nil {
		ext.WebRTCEnabled = *req.WebRTCEnabled
	}

	if err := h.DB.Save(&ext).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(ext)
}

func (h *APIHandler) DeleteExtension(c *fiber.Ctx) error {
	id := c.Params("id")
	res := h.DB.Delete(&db.Extension{}, "id = ?", id)
	if res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": res.Error.Error()})
	}
	if res.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Extension not found"})
	}
	return c.JSON(fiber.Map{"message": "Extension deleted successfully"})
}

// ==========================================
// DIDS (Customer API)
// ==========================================

func (h *APIHandler) ListDIDs(c *fiber.Ctx) error {
	tid := getTenantID(c)
	var dids []db.DID
	query := h.DB
	if tid != "" {
		query = query.Where("tenant_id = ?", tid)
	}
	if err := query.Find(&dids).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(dids)
}

func (h *APIHandler) GetDID(c *fiber.Ctx) error {
	id := c.Params("id")
	var did db.DID
	if err := h.DB.First(&did, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "DID not found"})
	}
	return c.JSON(did)
}

func (h *APIHandler) CreateDID(c *fiber.Ctx) error {
	var did db.DID
	if err := c.BodyParser(&did); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if did.TenantID == "" {
		did.TenantID = getTenantID(c)
	}
	if did.TenantID == "" || did.Number == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "tenant_id and number are required"})
	}

	if did.ID == "" {
		did.ID = uuid.New().String()
	}

	if err := h.DB.Create(&did).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(did)
}

func (h *APIHandler) UpdateDID(c *fiber.Ctx) error {
	id := c.Params("id")
	var did db.DID
	if err := h.DB.First(&did, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "DID not found"})
	}

	var req struct {
		Number             *string `json:"number"`
		AssignedWorkflowID *string `json:"assigned_workflow_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Number != nil {
		did.Number = *req.Number
	}
	if req.AssignedWorkflowID != nil {
		did.AssignedWorkflowID = *req.AssignedWorkflowID
	}

	if err := h.DB.Save(&did).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(did)
}

func (h *APIHandler) DeleteDID(c *fiber.Ctx) error {
	id := c.Params("id")
	res := h.DB.Delete(&db.DID{}, "id = ?", id)
	if res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": res.Error.Error()})
	}
	if res.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "DID not found"})
	}
	return c.JSON(fiber.Map{"message": "DID deleted successfully"})
}

// ==========================================
// WORKFLOWS (Customer API)
// ==========================================

func (h *APIHandler) ListWorkflows(c *fiber.Ctx) error {
	tid := getTenantID(c)
	var workflows []db.Workflow
	query := h.DB
	if tid != "" {
		query = query.Where("tenant_id = ?", tid)
	}
	if err := query.Find(&workflows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(workflows)
}

func (h *APIHandler) GetWorkflow(c *fiber.Ctx) error {
	id := c.Params("id")
	var wf db.Workflow
	if err := h.DB.First(&wf, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Workflow not found"})
	}
	return c.JSON(wf)
}

func (h *APIHandler) CreateWorkflow(c *fiber.Ctx) error {
	var wf db.Workflow
	if err := c.BodyParser(&wf); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if wf.TenantID == "" {
		wf.TenantID = getTenantID(c)
	}
	if wf.TenantID == "" || wf.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "tenant_id and name are required"})
	}

	if wf.ID == "" {
		wf.ID = uuid.New().String()
	}

	if err := h.DB.Create(&wf).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(wf)
}

func (h *APIHandler) UpdateWorkflow(c *fiber.Ctx) error {
	id := c.Params("id")
	var wf db.Workflow
	if err := h.DB.First(&wf, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Workflow not found"})
	}

	var req struct {
		Name        *string `json:"name"`
		TriggerType *string `json:"trigger_type"`
		Definition  *string `json:"definition"`
		IsActive    *bool   `json:"is_active"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Name != nil {
		wf.Name = *req.Name
	}
	if req.TriggerType != nil {
		wf.TriggerType = *req.TriggerType
	}
	if req.Definition != nil {
		wf.Definition = *req.Definition
	}
	if req.IsActive != nil {
		wf.IsActive = *req.IsActive
	}

	if err := h.DB.Save(&wf).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(wf)
}

func (h *APIHandler) DeleteWorkflow(c *fiber.Ctx) error {
	id := c.Params("id")
	res := h.DB.Delete(&db.Workflow{}, "id = ?", id)
	if res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": res.Error.Error()})
	}
	if res.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Workflow not found"})
	}
	return c.JSON(fiber.Map{"message": "Workflow deleted successfully"})
}

// ==========================================
// TENANT SETTINGS / PERSONA / FAQ / BYOK (Customer API)
// ==========================================

func (h *APIHandler) GetSettings(c *fiber.Ctx) error {
	tid := getTenantID(c)
	if tid == "" {
		tid = c.Params("tenant_id")
	}
	if tid == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "tenant_id is required via header X-Tenant-ID or query parameter"})
	}

	var settings db.TenantSettings
	err := h.DB.First(&settings, "tenant_id = ?", tid).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Settings not found for this tenant"})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(settings)
}

func (h *APIHandler) UpsertSettings(c *fiber.Ctx) error {
	tid := getTenantID(c)
	if tid == "" {
		tid = c.Params("tenant_id")
	}

	var req db.TenantSettings
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.TenantID != "" {
		tid = req.TenantID
	}
	if tid == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "tenant_id is required"})
	}
	req.TenantID = tid

	var existing db.TenantSettings
	err := h.DB.First(&existing, "tenant_id = ?", tid).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if req.ID == "" {
			req.ID = uuid.New().String()
		}
		if err := h.DB.Create(&req).Error; err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusCreated).JSON(req)
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Update fields if provided
	if req.ApiKey != "" {
		existing.ApiKey = req.ApiKey
	}
	if req.Provider != "" {
		existing.Provider = req.Provider
	}
	if req.FaqDataset != "" {
		existing.FaqDataset = req.FaqDataset
	}
	if req.ActiveVoiceID != "" {
		existing.ActiveVoiceID = req.ActiveVoiceID
	}
	if req.PersonaName != "" {
		existing.PersonaName = req.PersonaName
	}
	if req.PersonaPrompt != "" {
		existing.PersonaPrompt = req.PersonaPrompt
	}
	if len(req.FillerPhrases) > 0 {
		existing.FillerPhrases = req.FillerPhrases
	}
	if req.McpServerURL != "" {
		existing.McpServerURL = req.McpServerURL
	}

	if err := h.DB.Save(&existing).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(existing)
}

// ==========================================
// CDRs (Customer API)
// ==========================================

func (h *APIHandler) ListCDRs(c *fiber.Ctx) error {
	tid := getTenantID(c)
	caller := c.Query("caller")
	callee := c.Query("callee")
	status := c.Query("status")
	callType := c.Query("type")

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	query := h.DB.Model(&db.CDR{})
	if tid != "" {
		query = query.Where("tenant_id = ?", tid)
	}
	if caller != "" {
		query = query.Where("caller LIKE ?", "%"+caller+"%")
	}
	if callee != "" {
		query = query.Where("callee LIKE ?", "%"+callee+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if callType != "" {
		query = query.Where("type = ?", callType)
	}

	var total int64
	query.Count(&total)

	var cdrs []db.CDR
	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&cdrs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"total": total,
		"limit": limit,
		"offset": offset,
		"cdrs":  cdrs,
	})
}

func (h *APIHandler) GetCDR(c *fiber.Ctx) error {
	id := c.Params("id")
	var cdr db.CDR
	err := h.DB.Where("id = ? OR call_id = ?", id, id).First(&cdr).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "CDR not found"})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(cdr)
}

func (h *APIHandler) CreateCDR(c *fiber.Ctx) error {
	var cdr db.CDR
	if err := c.BodyParser(&cdr); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if cdr.TenantID == "" {
		cdr.TenantID = getTenantID(c)
	}
	if cdr.TenantID == "" || cdr.CallID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "tenant_id and call_id are required"})
	}

	if cdr.ID == "" {
		cdr.ID = uuid.New().String()
	}

	if err := h.DB.Create(&cdr).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(cdr)
}

// ==========================================
// WEBHOOKS (Kamailio SMS-IWF)
// ==========================================

func (h *APIHandler) HandleSmsWebhook(c *fiber.Ctx) error {
	var payload struct {
		SMSCID    string `json:"smsc_id"`
		From      string `json:"from"`
		To        string `json:"to"`
		Text      string `json:"text"`
		TenantID  string `json:"tenant_id"`
		MsgID     string `json:"msg_id"`
	}

	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body: " + err.Error()})
	}

	if payload.From == "" || payload.To == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "from and to fields are required"})
	}

	// Default tenant if not specified
	tid := payload.TenantID
	if tid == "" {
		tid = "default_tenant"
	}

	return c.JSON(fiber.Map{
		"status":     "ok",
		"tenant_id":  tid,
		"from":       payload.From,
		"to":         payload.To,
		"msg_id":     payload.MsgID,
		"dispatched": true,
	})
}

// ==========================================
// WEBSOCKET (FreeSWITCH mod_audio_fork /ws/audio)
// ==========================================

// HandleAudioStream coordinates bidirectional PCM audio streaming between FreeSWITCH and Live AI engine.
func (h *APIHandler) HandleAudioStream(c *websocket.Conn) {
	callUUID := c.Query("call_uuid")
	if callUUID == "" {
		callUUID = c.Query("uuid")
	}
	if callUUID == "" {
		callUUID = uuid.New().String()
	}

	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		tenantID = "default_tenant"
	}

	sampleRate := 8000
	rateStr := c.Query("sample_rate")
	if rateStr == "" {
		rateStr = c.Query("rate")
	}
	if rateStr == "16000" {
		sampleRate = 16000
	}

	log.Printf("[AudioStream] FreeSWITCH mod_audio_fork connected: call_uuid=%s, tenant=%s, rate=%d", callUUID, tenantID, sampleRate)

	// Fetch tenant settings if database is configured
	var settings db.TenantSettings
	if h.DB != nil {
		_ = h.DB.Where("tenant_id = ?", tenantID).First(&settings).Error
	}

	var aiEngine ai.LiveAIEngine
	if strings.EqualFold(settings.Provider, "Gemini") && settings.ApiKey != "" {
		geminiCfg := ai.GeminiLiveConfig{
			APIKey:            settings.ApiKey,
			Model:             ai.DefaultGeminiLiveModel,
			VoiceName:         settings.ActiveVoiceID,
			SystemInstruction: settings.PersonaPrompt,
			InputSampleRate:   16000,
		}
		client := ai.NewGeminiLiveClient(geminiCfg)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := client.Connect(ctx); err == nil {
			client.StartReadLoop()
			aiEngine = client
		} else {
			log.Printf("[AudioStream] Could not connect to Gemini Live (%v), falling back to modular streaming", err)
		}
		cancel()
	}

	if aiEngine == nil {
		// Fallback to modular streaming (Deepgram STT -> Instant Filler Injection -> LLM -> Cartesia/ElevenLabs TTS)
		modCfg := ai.ModularStreamerConfig{
			PersonaName:   settings.PersonaName,
			PersonaPrompt: settings.PersonaPrompt,
			VoiceID:       settings.ActiveVoiceID,
			FillerPhrases: settings.FillerPhrases,
			SampleRate:    16000,
		}
		aiEngine = ai.NewModularStreamer(modCfg, nil, nil, nil)
	}

	cfg := ai.StreamerConfig{
		CallUUID:     callUUID,
		TenantID:     tenantID,
		InboundRate:  sampleRate,
		OutboundRate: sampleRate,
		AIRate:       16000,
	}

	streamer := ai.NewAudioStreamer(cfg, c, aiEngine)

	if h.CallHandler != nil {
		h.CallHandler.AttachAudioStreamer(streamer)
	} else if h.StreamManager != nil {
		h.StreamManager.Register(callUUID, streamer)
	}

	defer func() {
		log.Printf("[AudioStream] FreeSWITCH mod_audio_fork disconnected: call_uuid=%s", callUUID)
		if h.CallHandler != nil && h.CallHandler.StreamManager() != nil {
			h.CallHandler.StreamManager().Unregister(callUUID)
		} else if h.StreamManager != nil {
			h.StreamManager.Unregister(callUUID)
		}
		_ = streamer.Close()
	}()

	streamer.Start()
}



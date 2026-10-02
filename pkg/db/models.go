package db

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Tenant represents an isolated business customer (e.g. Eczane, Kargo, Klinik).
type Tenant struct {
	ID        string    `gorm:"type:varchar(64);primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	Subdomain string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"subdomain"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Settings   *TenantSettings `gorm:"foreignKey:TenantID;references:ID;constraint:OnDelete:CASCADE" json:"settings,omitempty"`
	Extensions []Extension     `gorm:"foreignKey:TenantID;references:ID;constraint:OnDelete:CASCADE" json:"extensions,omitempty"`
	DIDs       []DID           `gorm:"foreignKey:TenantID;references:ID;constraint:OnDelete:CASCADE" json:"dids,omitempty"`
	Workflows  []Workflow      `gorm:"foreignKey:TenantID;references:ID;constraint:OnDelete:CASCADE" json:"workflows,omitempty"`
	CDRs       []CDR           `gorm:"foreignKey:TenantID;references:ID;constraint:OnDelete:CASCADE" json:"cdrs,omitempty"`
}

func (t *Tenant) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return
}

// TenantSettings stores tenant-level BYOK, LLM persona, voice, FAQ, and MCP configuration.
type TenantSettings struct {
	ID            string    `gorm:"type:varchar(64);primaryKey" json:"id"`
	TenantID      string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"tenant_id"`
	ApiKey        string    `gorm:"type:varchar(255)" json:"api_key"`
	Provider      string    `gorm:"type:varchar(50);default:'Gemini'" json:"provider"` // Gemini, OpenAI, Anthropic
	FaqDataset    string    `gorm:"type:text" json:"faq_dataset"`                      // JSON or formatted text
	ActiveVoiceID string    `gorm:"type:varchar(100)" json:"active_voice_id"`
	PersonaName   string    `gorm:"type:varchar(100)" json:"persona_name"`
	PersonaPrompt string    `gorm:"type:text" json:"persona_prompt"`
	FillerPhrases []string  `gorm:"serializer:json" json:"filler_phrases"`
	McpServerURL  string    `gorm:"type:varchar(255)" json:"mcp_server_url"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (ts *TenantSettings) BeforeCreate(tx *gorm.DB) (err error) {
	if ts.ID == "" {
		ts.ID = uuid.New().String()
	}
	return
}

// Extension represents an internal SIP extension (e.g. 101, 102).
type Extension struct {
	ID            string    `gorm:"type:varchar(64);primaryKey" json:"id"`
	TenantID      string    `gorm:"type:varchar(64);index;not null" json:"tenant_id"`
	ExtNumber     string    `gorm:"type:varchar(20);not null" json:"ext_number"`
	Name          string    `gorm:"type:varchar(100);not null" json:"name"`
	Password      string    `gorm:"type:varchar(100);not null" json:"password"`
	WebRTCEnabled bool      `gorm:"default:true" json:"webrtc_enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (e *Extension) BeforeCreate(tx *gorm.DB) (err error) {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	return
}

// DID represents a public telephone number (e.g. 08501110001) mapped to a workflow.
type DID struct {
	ID                 string    `gorm:"type:varchar(64);primaryKey" json:"id"`
	TenantID           string    `gorm:"type:varchar(64);index;not null" json:"tenant_id"`
	Number             string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"number"`
	AssignedWorkflowID string    `gorm:"type:varchar(64)" json:"assigned_workflow_id"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (d *DID) BeforeCreate(tx *gorm.DB) (err error) {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return
}

// Workflow represents an n8n-style visual drag-and-drop workflow with trigger and node definitions.
type Workflow struct {
	ID          string    `gorm:"type:varchar(64);primaryKey" json:"id"`
	TenantID    string    `gorm:"type:varchar(64);index;not null" json:"tenant_id"`
	Name        string    `gorm:"type:varchar(255);not null" json:"name"`
	TriggerType string    `gorm:"type:varchar(20);default:'CALL'" json:"trigger_type"` // CALL, SMS
	Definition  string    `gorm:"type:text" json:"definition"`                         // JSON nodes and connections
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (w *Workflow) BeforeCreate(tx *gorm.DB) (err error) {
	if w.ID == "" {
		w.ID = uuid.New().String()
	}
	return
}

// CDR represents Call Detail Records for reporting, auditing and billing.
type CDR struct {
	ID             string    `gorm:"type:varchar(64);primaryKey" json:"id"`
	TenantID       string    `gorm:"type:varchar(64);index;not null" json:"tenant_id"`
	CallID         string    `gorm:"type:varchar(100);index" json:"call_id"`
	Caller         string    `gorm:"type:varchar(50)" json:"caller"`
	Callee         string    `gorm:"type:varchar(50)" json:"callee"`
	Type           string    `gorm:"type:varchar(20);default:'CALL'" json:"type"` // CALL, SMS
	DurationSec    int       `gorm:"default:0" json:"duration_sec"`
	Status         string    `gorm:"type:varchar(50)" json:"status"` // ANSWERED, BUSY, NO_ANSWER, FAILED
	Cost           float64   `gorm:"type:decimal(10,4);default:0" json:"cost"`
	TranscriptText string    `gorm:"type:text" json:"transcript_text"`
	RecordingURL   string    `gorm:"type:varchar(500)" json:"recording_url"`
	CreatedAt      time.Time `json:"created_at"`
}

func (c *CDR) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return
}

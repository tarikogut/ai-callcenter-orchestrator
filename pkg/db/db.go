package db

import (
	"errors"
	"log"
	"os"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB initializes the database using PostgreSQL or seamlessly falls back to SQLite.
func InitDB(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}

	var db *gorm.DB
	var err error

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	}

	isPostgres := strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://")

	if isPostgres {
		log.Printf("[DB] Connecting to PostgreSQL...")
		db, err = gorm.Open(postgres.Open(dsn), gormConfig)
		if err != nil {
			log.Printf("[DB] WARNING: PostgreSQL connection failed (%v). Falling back to SQLite...", err)
			sqliteFile := os.Getenv("SQLITE_DB_PATH")
			if sqliteFile == "" {
				sqliteFile = "callcenter.db"
			}
			db, err = gorm.Open(sqlite.Open(sqliteFile), gormConfig)
		}
	} else {
		sqliteFile := dsn
		if sqliteFile == "" {
			sqliteFile = os.Getenv("SQLITE_DB_PATH")
			if sqliteFile == "" {
				sqliteFile = "callcenter.db"
			}
		}
		log.Printf("[DB] Connecting to SQLite (%s)...", sqliteFile)
		db, err = gorm.Open(sqlite.Open(sqliteFile), gormConfig)
	}

	if err != nil {
		return nil, err
	}

	// Auto-migrate tables
	if err := AutoMigrate(db); err != nil {
		return nil, err
	}

	// Seed default tenants and data
	if err := SeedDefaultData(db); err != nil {
		log.Printf("[DB] Warning during seeding: %v", err)
	}

	return db, nil
}

// InitTestDB initializes an in-memory SQLite database for unit tests.
func InitTestDB() (*gorm.DB, error) {
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), gormConfig)
	if err != nil {
		return nil, err
	}

	if err := AutoMigrate(db); err != nil {
		return nil, err
	}

	if err := SeedDefaultData(db); err != nil {
		return nil, err
	}

	return db, nil
}

// AutoMigrate migrates all schema models.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Tenant{},
		&TenantSettings{},
		&Extension{},
		&DID{},
		&Workflow{},
		&CDR{},
	)
}

// SeedDefaultData seeds eczane_hayat and jet_kargo tenants if not present.
func SeedDefaultData(db *gorm.DB) error {
	// 1. Seed Eczane Hayat
	var existingEczane Tenant
	err := db.Where("subdomain = ?", "eczane_hayat").First(&existingEczane).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		eczane := Tenant{
			ID:        "tenant-eczane-hayat",
			Name:      "Hayat Eczanesi",
			Subdomain: "eczane_hayat",
		}
		if err := db.Create(&eczane).Error; err != nil {
			return err
		}

		// Settings for Eczane
		eczaneSettings := TenantSettings{
			ID:            "settings-eczane-hayat",
			TenantID:      eczane.ID,
			ApiKey:        "sk-eczane-demo-key",
			Provider:      "Gemini",
			ActiveVoiceID: "Puck",
			PersonaName:   "Ayşe (Eczacı Asistanı)",
			PersonaPrompt: "Sen Hayat Eczanesi'nin yapay zeka sesli asistanı Ayşe'sin. Müşterilere nöbetçi eczane, ilaç stok durumu ve reçete sorgularında nazik ve profesyonelce yardımcı olursun.",
			McpServerURL:  "http://localhost:8001/mcp/stock",
			FillerPhrases: []string{
				"Bir saniye stoklarımıza bakıyorum...",
				"Hemen kontrol ediyorum efendim...",
				"Reçete sistemine bağlanıyorum...",
			},
			FaqDataset: `[{"q":"Çalışma saatleriniz nedir?","a":"Hafta içi 08:30 - 19:00 arası açığız."},{"q":"Nöbetçi misiniz?","a":"Bu hafta sonu Cumartesi günü nöbetçi eczaneyiz."}]`,
		}
		db.Create(&eczaneSettings)

		// Extension 101
		eczaneExt := Extension{
			ID:            "ext-eczane-101",
			TenantID:      eczane.ID,
			ExtNumber:     "101",
			Name:          "Eczane Banko / Danışma",
			Password:      "eczane101pass",
			WebRTCEnabled: true,
		}
		db.Create(&eczaneExt)

		// Workflow for Eczane
		eczaneWorkflow := Workflow{
			ID:          "wf-eczane-main",
			TenantID:    eczane.ID,
			Name:        "Eczane Çağrı Karşılama ve Stok Kontrolü",
			TriggerType: "CALL",
			Definition:  `{"nodes":[{"id":"inbound_1","type":"InboundCallTrigger","data":{"label":"Gelen Arama"}},{"id":"ai_voice_1","type":"AiVoiceAgentNode","data":{"persona":"Ayşe (Eczacı Asistanı)","mcp_tool_url":"http://localhost:8001/mcp/stock"}},{"id":"transfer_1","type":"TransferNode","data":{"destination":"101"}}],"edges":[{"source":"inbound_1","target":"ai_voice_1"},{"source":"ai_voice_1","target":"transfer_1"}]}`,
			IsActive:    true,
		}
		db.Create(&eczaneWorkflow)

		// DID 08501110001
		eczaneDID := DID{
			ID:                 "did-eczane-08501110001",
			TenantID:           eczane.ID,
			Number:             "08501110001",
			AssignedWorkflowID: eczaneWorkflow.ID,
		}
		db.Create(&eczaneDID)

		log.Printf("[DB] Seeded default tenant: %s (%s)", eczane.Name, eczane.Subdomain)
	}

	// 2. Seed Jet Kargo
	var existingKargo Tenant
	err = db.Where("subdomain = ?", "jet_kargo").First(&existingKargo).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		kargo := Tenant{
			ID:        "tenant-jet-kargo",
			Name:      "Jet Kargo",
			Subdomain: "jet_kargo",
		}
		if err := db.Create(&kargo).Error; err != nil {
			return err
		}

		// Settings for Jet Kargo
		kargoSettings := TenantSettings{
			ID:            "settings-jet-kargo",
			TenantID:      kargo.ID,
			ApiKey:        "sk-kargo-demo-key",
			Provider:      "Gemini",
			ActiveVoiceID: "Fenrir",
			PersonaName:   "Can (Kargo Temsilcisi)",
			PersonaPrompt: "Sen Jet Kargo'nun müşteri temsilcisi Can'sın. Kargo takip numarası ile sorgulama ve kurye taleplerine hızlı ve net yanıtlar verirsin.",
			McpServerURL:  "http://localhost:8002/mcp/tracking",
			FillerPhrases: []string{
				"Hemen kargo durumunuza bakıyorum...",
				"Sistemden takip kodunu sorguluyorum...",
				"Kurye rotasını kontrol ediyorum...",
			},
			FaqDataset: `[{"q":"Teslimat süresi ne kadar?","a":"Standart gönderiler 1-2 iş günü içerisinde teslim edilir."},{"q":"Kurye çağırabilir miyim?","a":"Evet, mesai saatleri içinde kurye talep edebilirsiniz."}]`,
		}
		db.Create(&kargoSettings)

		// Extension 201
		kargoExt := Extension{
			ID:            "ext-kargo-201",
			TenantID:      kargo.ID,
			ExtNumber:     "201",
			Name:          "Kargo Müşteri Masası",
			Password:      "kargo201pass",
			WebRTCEnabled: true,
		}
		db.Create(&kargoExt)

		// Workflow for Jet Kargo
		kargoWorkflow := Workflow{
			ID:          "wf-kargo-main",
			TenantID:    kargo.ID,
			Name:        "Jet Kargo Takip & Yönlendirme Akışı",
			TriggerType: "CALL",
			Definition:  `{"nodes":[{"id":"inbound_1","type":"InboundCallTrigger","data":{"label":"Gelen Arama"}},{"id":"ai_voice_1","type":"AiVoiceAgentNode","data":{"persona":"Can (Kargo Temsilcisi)","mcp_tool_url":"http://localhost:8002/mcp/tracking"}},{"id":"transfer_1","type":"TransferNode","data":{"destination":"201"}}],"edges":[{"source":"inbound_1","target":"ai_voice_1"},{"source":"ai_voice_1","target":"transfer_1"}]}`,
			IsActive:    true,
		}
		db.Create(&kargoWorkflow)

		// DID 08502220002
		kargoDID := DID{
			ID:                 "did-kargo-08502220002",
			TenantID:           kargo.ID,
			Number:             "08502220002",
			AssignedWorkflowID: kargoWorkflow.ID,
		}
		db.Create(&kargoDID)

		log.Printf("[DB] Seeded default tenant: %s (%s)", kargo.Name, kargo.Subdomain)
	}

	return nil
}

package rag

import (
	"context"
	"strings"
	"testing"
)

func TestInMemoryEngine(t *testing.T) {
	engine := NewInMemoryEngine()
	ctx := context.Background()

	tenantID := "tenant_pharmacy_01"

	docs := []Document{
		{
			ID:       "doc_1",
			TenantID: tenantID,
			Category: "working_hours",
			Question: "Eczanenin çalışma saatleri nedir?",
			Answer:   "Hafta içi ve Cumartesi günleri 08:30 - 19:00 saatleri arasında açığız. Pazar günleri nöbetçi eczaneler hizmet vermektedir.",
		},
		{
			ID:       "doc_2",
			TenantID: tenantID,
			Category: "pricing",
			Question: "Reçete ve ilaç katkı payı nasıl ödenir?",
			Answer:   "SGK anlaşmalı reçetelerde katkı payı nakit veya kredi kartı ile ödenebilir.",
		},
		{
			ID:       "doc_3",
			TenantID: tenantID,
			Category: "faq",
			Question: "Nöbetçi eczane nasıl bulunur?",
			Answer:   "Gece ve Pazar günleri nöbetçi eczane listesini panomuzdan veya çağrı merkezimizden öğrenebilirsiniz.",
		},
		{
			ID:       "doc_other_tenant",
			TenantID: "tenant_courier_01",
			Category: "faq",
			Question: "Kargo dağıtım saatleri nedir?",
			Answer:   "Kargolarımız 09:00 - 18:00 saatleri arasında teslim edilmektedir.",
		},
	}

	if err := engine.AddDocuments(ctx, docs...); err != nil {
		t.Fatalf("AddDocuments failed: %v", err)
	}

	// Test Search for working hours
	results, err := engine.Search(ctx, tenantID, "çalışma saatleri kaçta açılıyor", 2)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected search results for working hours, got 0")
	}
	if results[0].Document.ID != "doc_1" {
		t.Fatalf("expected doc_1 as top result, got %s", results[0].Document.ID)
	}

	// Test tenant isolation
	courierResults, err := engine.Search(ctx, "tenant_courier_01", "çalışma saatleri", 2)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(courierResults) == 0 || courierResults[0].Document.TenantID != "tenant_courier_01" {
		t.Fatalf("expected courier tenant doc, got: %+v", courierResults)
	}

	// Test FormatPromptContext
	promptCtx, err := engine.FormatPromptContext(ctx, tenantID, "katkı payı ödemesi kredi kartı geçerli mi", 1)
	if err != nil {
		t.Fatalf("FormatPromptContext failed: %v", err)
	}
	if !strings.Contains(promptCtx, "Reçete ve ilaç katkı payı") || !strings.Contains(promptCtx, "kredi kartı") {
		t.Fatalf("expected formatted context to contain pricing FAQ, got: %s", promptCtx)
	}

	// Test DeleteDocument
	if err := engine.DeleteDocument(ctx, tenantID, "doc_1"); err != nil {
		t.Fatalf("DeleteDocument failed: %v", err)
	}
	resultsAfterDel, err := engine.Search(ctx, tenantID, "çalışma saatleri", 2)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	for _, r := range resultsAfterDel {
		if r.Document.ID == "doc_1" {
			t.Fatalf("doc_1 should have been deleted")
		}
	}
}

func TestCosineSimilarity(t *testing.T) {
	v1 := []float32{1.0, 0.0, 0.0}
	v2 := []float32{1.0, 0.0, 0.0}
	v3 := []float32{0.0, 1.0, 0.0}

	sim1 := CosineSimilarity(v1, v2)
	if sim1 < 0.99 {
		t.Errorf("expected ~1.0 similarity for identical vectors, got %f", sim1)
	}

	sim2 := CosineSimilarity(v1, v3)
	if sim2 != 0.0 {
		t.Errorf("expected 0.0 similarity for orthogonal vectors, got %f", sim2)
	}
}

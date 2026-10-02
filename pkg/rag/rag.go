package rag

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"
)

// Document represents a knowledge base or FAQ item
type Document struct {
	ID        string            `json:"id"`
	TenantID  string            `json:"tenant_id"`
	Category  string            `json:"category"` // "faq", "working_hours", "pricing", "policy"
	Question  string            `json:"question"`
	Answer    string            `json:"answer"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Embedding []float32         `json:"embedding,omitempty"`
}

// SearchResult represents a scored document match
type SearchResult struct {
	Document Document `json:"document"`
	Score    float64  `json:"score"`
}

// Engine defines the FAQ & Knowledge Base RAG interface
type Engine interface {
	AddDocuments(ctx context.Context, docs ...Document) error
	DeleteDocument(ctx context.Context, tenantID, id string) error
	Search(ctx context.Context, tenantID string, query string, topK int) ([]SearchResult, error)
	FormatPromptContext(ctx context.Context, tenantID string, query string, topK int) (string, error)
}

// InMemoryEngine implements in-memory text & vector similarity search
type InMemoryEngine struct {
	mu   sync.RWMutex
	docs map[string][]Document // tenantID -> docs
}

// NewInMemoryEngine creates an in-memory RAG engine
func NewInMemoryEngine() *InMemoryEngine {
	return &InMemoryEngine{
		docs: make(map[string][]Document),
	}
}

// AddDocuments indexes documents for a tenant
func (e *InMemoryEngine) AddDocuments(ctx context.Context, docs ...Document) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, doc := range docs {
		tenantDocs := e.docs[doc.TenantID]
		// Replace if already exists with same ID
		found := false
		for i, existing := range tenantDocs {
			if existing.ID == doc.ID {
				tenantDocs[i] = doc
				found = true
				break
			}
		}
		if !found {
			tenantDocs = append(tenantDocs, doc)
		}
		e.docs[doc.TenantID] = tenantDocs
	}
	return nil
}

// DeleteDocument removes a document by ID for a tenant
func (e *InMemoryEngine) DeleteDocument(ctx context.Context, tenantID, id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	tenantDocs := e.docs[tenantID]
	filtered := make([]Document, 0, len(tenantDocs))
	for _, doc := range tenantDocs {
		if doc.ID != id {
			filtered = append(filtered, doc)
		}
	}
	e.docs[tenantID] = filtered
	return nil
}

// Search performs hybrid lexical (BM25/token-overlap) and optional vector similarity search
func (e *InMemoryEngine) Search(ctx context.Context, tenantID string, query string, topK int) ([]SearchResult, error) {
	e.mu.RLock()
	tenantDocs := e.docs[tenantID]
	e.mu.RUnlock()

	if len(tenantDocs) == 0 || topK <= 0 {
		return nil, nil
	}

	queryTokens := tokenize(query)
	results := make([]SearchResult, 0, len(tenantDocs))

	for _, doc := range tenantDocs {
		// Calculate token overlap / lexical score
		docText := doc.Question + " " + doc.Answer + " " + doc.Category
		for _, v := range doc.Metadata {
			docText += " " + v
		}
		docTokens := tokenize(docText)

		lexScore := computeTokenOverlap(queryTokens, docTokens)

		// Extra boost if question directly contains query keywords
		qTokens := tokenize(doc.Question)
		qScore := computeTokenOverlap(queryTokens, qTokens)

		score := (lexScore * 0.4) + (qScore * 0.6)

		if score > 0 {
			results = append(results, SearchResult{
				Document: doc,
				Score:    score,
			})
		}
	}

	// Sort descending by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

// FormatPromptContext formats retrieved knowledge items as a context block for AI agent prompts
func (e *InMemoryEngine) FormatPromptContext(ctx context.Context, tenantID string, query string, topK int) (string, error) {
	results, err := e.Search(ctx, tenantID, query, topK)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString("=== Bilgi Bankası ve Sıkça Sorulan Sorular (Knowledge Base / FAQ) ===\n")
	for i, r := range results {
		doc := r.Document
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		if doc.Category != "" {
			sb.WriteString("[Kategori: " + doc.Category + "]\n")
		}
		if doc.Question != "" {
			sb.WriteString("Soru: " + doc.Question + "\n")
		}
		if doc.Answer != "" {
			sb.WriteString("Cevap: " + doc.Answer + "\n")
		}
		for k, v := range doc.Metadata {
			sb.WriteString(k + ": " + v + "\n")
		}
		if i == len(results)-1 {
			sb.WriteString(strings.Repeat("-", 40) + "\n")
		}
	}
	sb.WriteString("Yukarıdaki bilgileri esas alarak müşterinin sorusunu doğru ve net şekilde yanıtla.\n")

	return sb.String(), nil
}

// Helper tokenization & similarity functions
func tokenize(text string) []string {
	lower := strings.ToLower(text)
	// Replace punctuation with spaces
	replacer := strings.NewReplacer(
		".", " ", ",", " ", "!", " ", "?", " ", ";", " ", ":", " ", "-", " ", "(", " ", ")", " ", "\"", " ",
	)
	cleaned := replacer.Replace(lower)
	fields := strings.Fields(cleaned)
	return fields
}

func computeTokenOverlap(queryTokens, docTokens []string) float64 {
	if len(queryTokens) == 0 || len(docTokens) == 0 {
		return 0.0
	}

	docTokenMap := make(map[string]int)
	for _, dt := range docTokens {
		docTokenMap[dt]++
	}

	matchCount := 0
	for _, qt := range queryTokens {
		if _, exists := docTokenMap[qt]; exists {
			matchCount++
		}
	}

	// Normalized overlap score
	queryRatio := float64(matchCount) / float64(len(queryTokens))
	docRatio := float64(matchCount) / float64(len(docTokens))

	// Harmonic mean / F1-style or weighted overlap
	return (0.7 * queryRatio) + (0.3 * docRatio)
}

// CosineSimilarity computes similarity between two embeddings
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}
	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	if normA == 0 || normB == 0 {
		return 0.0
	}
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

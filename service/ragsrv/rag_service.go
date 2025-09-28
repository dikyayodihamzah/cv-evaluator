package ragsrv

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/dikyayodihamzah/cv-evaluator/config/llmclient"
	"github.com/dikyayodihamzah/cv-evaluator/pkg/constant"
)

type RAGService interface {
	GetRelevantContext(contextType, query string) (string, error)
	IndexDocument(docType, content string) error
	Initialize() error
}

type ragService struct {
	documents    map[string][]Document
	mutex        sync.RWMutex
	openaiClient *llmclient.OpenAIClient
}

type Document struct {
	ID      string
	Type    string
	Content string
	Vector  []float32 // OpenAI embeddings are float32
}

func New() RAGService {
	return &ragService{
		documents:    make(map[string][]Document),
		openaiClient: llmclient.NewOpenAIClient(),
	}
}

func (s *ragService) Initialize() error {
	// Initialize with default job descriptions and scoring rubrics
	defaultJobDesc := constant.JobDescription
	cvEvaluationRubric := constant.CVEvaluationRubric
	projectEvaluationRubric := constant.ProjectEvaluationRubric

	// Index default documents
	s.IndexDocument("job_description", defaultJobDesc)
	s.IndexDocument("cv_evaluation", cvEvaluationRubric)
	s.IndexDocument("project_evaluation", projectEvaluationRubric)

	return nil
}

func (s *ragService) IndexDocument(docType, content string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Generate real vector embedding using OpenAI
	vector, err := s.openaiClient.GetEmbedding(content)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	doc := Document{
		ID:      fmt.Sprintf("%s_%d", docType, len(s.documents[docType])),
		Type:    docType,
		Content: content,
		Vector:  vector,
	}

	s.documents[docType] = append(s.documents[docType], doc)
	return nil
}

func (s *ragService) GetRelevantContext(contextType, query string) (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	docs, exists := s.documents[contextType]
	if !exists || len(docs) == 0 {
		return "", fmt.Errorf("no documents found for context type: %s", contextType)
	}

	// Generate embedding for the query
	queryVector, err := s.openaiClient.GetEmbedding(query)
	if err != nil {
		// Fallback to keyword matching if embedding fails
		bestMatch := s.findBestMatchKeyword(docs, query)
		if bestMatch == nil {
			return "", fmt.Errorf("no relevant context found")
		}
		return bestMatch.Content, nil
	}

	// Use vector similarity for relevance scoring
	bestMatch := s.findBestMatchVector(docs, queryVector)
	if bestMatch == nil {
		return "", fmt.Errorf("no relevant context found")
	}

	return bestMatch.Content, nil
}

func (s *ragService) findBestMatchVector(docs []Document, queryVector []float32) *Document {
	if len(docs) == 0 {
		return nil
	}

	bestScore := -1.0 // Cosine similarity ranges from -1 to 1
	var bestDoc *Document

	for i := range docs {
		if len(docs[i].Vector) == 0 {
			continue // Skip documents without vectors
		}

		score := s.cosineSimilarity(queryVector, docs[i].Vector)
		if score > bestScore {
			bestScore = score
			bestDoc = &docs[i]
		}
	}

	// Fallback to first document if no good match found
	if bestDoc == nil && len(docs) > 0 {
		bestDoc = &docs[0]
	}

	return bestDoc
}

func (s *ragService) findBestMatchKeyword(docs []Document, query string) *Document {
	if len(docs) == 0 {
		return nil
	}

	queryWords := strings.Fields(strings.ToLower(query))
	bestScore := 0.0
	var bestDoc *Document

	for i := range docs {
		score := s.calculateKeywordSimilarity(&docs[i], queryWords)
		if score > bestScore {
			bestScore = score
			bestDoc = &docs[i]
		}
	}

	if bestDoc == nil && len(docs) > 0 {
		bestDoc = &docs[0] // Fallback to first document
	}

	return bestDoc
}

func (s *ragService) cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0.0 || normB == 0.0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func (s *ragService) calculateKeywordSimilarity(doc *Document, queryWords []string) float64 {
	docContent := strings.ToLower(doc.Content)
	matches := 0

	for _, word := range queryWords {
		if strings.Contains(docContent, word) {
			matches++
		}
	}

	if len(queryWords) == 0 {
		return 0.0
	}

	return float64(matches) / float64(len(queryWords))
}

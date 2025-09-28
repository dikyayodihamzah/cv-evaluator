package ragsrv

import (
	"fmt"
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

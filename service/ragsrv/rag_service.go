package ragsrv

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/dikyayodihamzah/cv-evaluator/config/llmclient"
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
	defaultJobDesc := `
Senior Backend Engineer Position

Requirements:
- 5+ years of backend development experience
- Proficiency in Go, Python, or Java
- Experience with cloud platforms (AWS, GCP, Azure)
- Knowledge of microservices architecture
- Database design and optimization (SQL and NoSQL)
- API design and implementation (REST, GraphQL)
- Experience with containerization (Docker, Kubernetes)
- Understanding of CI/CD pipelines
- Strong problem-solving and communication skills

Preferred:
- Experience with AI/ML integration
- Knowledge of LLM and prompt engineering
- RAG (Retrieval Augmented Generation) experience
- Event-driven architecture
- Performance optimization
- Monitoring and observability tools

We're looking for someone who can design scalable systems, mentor junior developers, and drive technical decisions in a fast-paced environment.
`

	cvEvaluationRubric := `
CV Evaluation Criteria:

1. Technical Skills Match (40% weight)
   - Backend technologies alignment
   - Cloud platform experience
   - Database knowledge
   - API development skills
   - AI/LLM exposure (bonus)

2. Experience Level (30% weight)
   - Years of relevant experience
   - Project complexity and scale
   - Leadership and mentoring experience
   - Industry domain knowledge

3. Relevant Achievements (20% weight)
   - Performance improvements
   - System design contributions
   - Team leadership
   - Process improvements

4. Cultural Fit (10% weight)
   - Communication skills
   - Learning attitude
   - Collaboration indicators
   - Innovation mindset

Scoring: Rate each area 1-5, then calculate weighted average for final match rate.
`

	projectEvaluationRubric := `
Project Evaluation Criteria (1-5 scale each):

1. Correctness (25% weight)
   - Meets stated requirements
   - Proper prompt design and engineering
   - LLM chaining implementation
   - RAG integration
   - Error handling implementation

2. Code Quality (20% weight)
   - Clean, readable code
   - Modular architecture
   - Proper abstractions
   - Following best practices
   - Code documentation

3. Resilience (20% weight)
   - Error handling and recovery
   - Retry mechanisms
   - Circuit breakers
   - Graceful degradation
   - Monitoring and logging

4. Documentation (15% weight)
   - Clear README
   - API documentation
   - Architecture decisions
   - Trade-offs explanation
   - Setup instructions

5. Creativity/Bonus (20% weight)
   - Additional features
   - Performance optimizations
   - Security implementations
   - Deployment automation
   - User interface
   - Testing coverage

Final Score: Sum of weighted scores (max 10.0)
`

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

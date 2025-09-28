package ragsrv

import (
	"fmt"
	"testing"

	"github.com/dikyayodihamzah/cv-evaluator/config/llmclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockOpenAIClient for mocking LLM client
type MockOpenAIClient struct {
	mock.Mock
}

func (m *MockOpenAIClient) GetEmbedding(text string) ([]float32, error) {
	args := m.Called(text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]float32), args.Error(1)
}

func (m *MockOpenAIClient) ExtractCVInfo(cvContent string) (*llmclient.CVInfo, error) {
	args := m.Called(cvContent)
	return args.Get(0).(*llmclient.CVInfo), args.Error(1)
}

func (m *MockOpenAIClient) ScoreCV(cvInfo *llmclient.CVInfo, jobDesc, ragContext string) (*llmclient.CVEvaluation, error) {
	args := m.Called(cvInfo, jobDesc, ragContext)
	return args.Get(0).(*llmclient.CVEvaluation), args.Error(1)
}

func (m *MockOpenAIClient) EvaluateProject(projectContent, ragContext string) (*llmclient.ProjectEvaluation, error) {
	args := m.Called(projectContent, ragContext)
	return args.Get(0).(*llmclient.ProjectEvaluation), args.Error(1)
}

// RAGServiceTestSuite contains test cases for RAG service
type RAGServiceTestSuite struct {
	suite.Suite
	mockOpenAIClient *MockOpenAIClient
	ragService       RAGService
}

func (suite *RAGServiceTestSuite) SetupTest() {
	suite.mockOpenAIClient = new(MockOpenAIClient)

	// Create service with nil client for testing non-OpenAI functions
	service := &ragService{
		documents:    make(map[string][]Document),
		openaiClient: nil, // We'll test functions that don't require OpenAI
	}

	suite.ragService = service
}

func (suite *RAGServiceTestSuite) TestCosineSimilarity() {
	service := &ragService{}

	testCases := []struct {
		name     string
		a        []float32
		b        []float32
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1.0, 0.0},
			b:        []float32{0.0, 1.0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1.0, 0.0},
			b:        []float32{-1.0, 0.0},
			expected: -1.0,
		},
		{
			name:     "different length vectors",
			a:        []float32{1.0, 0.0},
			b:        []float32{1.0, 0.0, 1.0},
			expected: 0.0, // Should return 0 for different length vectors
		},
		{
			name:     "zero vectors",
			a:        []float32{0.0, 0.0},
			b:        []float32{0.0, 0.0},
			expected: 0.0,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			result := service.cosineSimilarity(tc.a, tc.b)
			assert.InDelta(t, tc.expected, result, 0.0001)
		})
	}
}

func (suite *RAGServiceTestSuite) TestCalculateKeywordSimilarity() {
	service := &ragService{}

	testCases := []struct {
		name       string
		doc        *Document
		queryWords []string
		expected   float64
	}{
		{
			name: "perfect match",
			doc: &Document{
				Content: "backend engineer with go programming skills",
			},
			queryWords: []string{"backend", "engineer", "go"},
			expected:   1.0, // All 3 words found
		},
		{
			name: "partial match",
			doc: &Document{
				Content: "frontend developer with javascript skills",
			},
			queryWords: []string{"frontend", "backend", "developer"},
			expected:   2.0 / 3.0, // 2 out of 3 words found
		},
		{
			name: "no match",
			doc: &Document{
				Content: "data scientist with python experience",
			},
			queryWords: []string{"backend", "go", "api"},
			expected:   0.0, // No words found
		},
		{
			name: "empty query",
			doc: &Document{
				Content: "any content",
			},
			queryWords: []string{},
			expected:   0.0,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			result := service.calculateKeywordSimilarity(tc.doc, tc.queryWords)
			assert.InDelta(t, tc.expected, result, 0.0001)
		})
	}
}

func (suite *RAGServiceTestSuite) TestFindBestMatchVector() {
	service := &ragService{}

	docs := []Document{
		{
			ID:      "doc1",
			Content: "Backend development",
			Vector:  []float32{1.0, 0.0, 0.0},
		},
		{
			ID:      "doc2",
			Content: "Frontend development",
			Vector:  []float32{0.0, 1.0, 0.0},
		},
		{
			ID:      "doc3",
			Content: "Full stack development",
			Vector:  []float32{0.5, 0.5, 0.0},
		},
	}

	// Query vector closest to doc1
	queryVector := []float32{0.9, 0.1, 0.0}

	result := service.findBestMatchVector(docs, queryVector)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "doc1", result.ID)
}

func (suite *RAGServiceTestSuite) TestFindBestMatchVector_EmptyDocs() {
	service := &ragService{}

	docs := []Document{}
	queryVector := []float32{1.0, 0.0}

	result := service.findBestMatchVector(docs, queryVector)

	assert.Nil(suite.T(), result)
}

func (suite *RAGServiceTestSuite) TestFindBestMatchKeyword() {
	service := &ragService{}

	docs := []Document{
		{
			ID:      "doc1",
			Content: "Backend engineer with Go programming skills",
		},
		{
			ID:      "doc2",
			Content: "Frontend developer with React experience",
		},
		{
			ID:      "doc3",
			Content: "Data scientist with Python and machine learning",
		},
	}

	query := "backend go programming"

	result := service.findBestMatchKeyword(docs, query)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "doc1", result.ID)
}

func (suite *RAGServiceTestSuite) TestFindBestMatchKeyword_EmptyDocs() {
	service := &ragService{}

	docs := []Document{}
	query := "test"

	result := service.findBestMatchKeyword(docs, query)

	assert.Nil(suite.T(), result)
}

// Integration test
func (suite *RAGServiceTestSuite) TestFullWorkflow() {
	// Initialize service
	suite.mockOpenAIClient.On("GetEmbedding", mock.AnythingOfType("string")).
		Return([]float32{0.1, 0.2, 0.3, 0.4, 0.5}, nil).
		Maybe()

	err := suite.ragService.Initialize()
	assert.NoError(suite.T(), err)

	// Index a document
	docContent := "Senior backend engineer with 5+ years Go experience"
	err = suite.ragService.IndexDocument("job_requirements", docContent)
	assert.NoError(suite.T(), err)

	// Query for relevant context
	query := "backend go developer requirements"
	context, err := suite.ragService.GetRelevantContext("job_requirements", query)

	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), context)
}

// Error edge cases
func (suite *RAGServiceTestSuite) TestGetRelevantContext_EmptyDocuments() {
	service := suite.ragService.(*ragService)

	// Add empty document list
	service.documents["empty_type"] = []Document{}

	result, err := suite.ragService.GetRelevantContext("empty_type", "test query")

	assert.Error(suite.T(), err)
	assert.Empty(suite.T(), result)
}

func (suite *RAGServiceTestSuite) TestIndexDocument_MultipleDocuments() {
	docType := "multi_test"

	// Index multiple documents
	for i := 0; i < 3; i++ {
		content := fmt.Sprintf("Document content %d", i)
		embedding := []float32{float32(i), float32(i + 1), float32(i + 2)}

		suite.mockOpenAIClient.On("GetEmbedding", content).
			Return(embedding, nil).Once()

		err := suite.ragService.IndexDocument(docType, content)
		assert.NoError(suite.T(), err)
	}

	// Verify all documents were indexed
	service := suite.ragService.(*ragService)
	docs, exists := service.documents[docType]
	assert.True(suite.T(), exists)
	assert.Len(suite.T(), docs, 3)
}

// Benchmark tests
func BenchmarkRAGService_CosineSimilarity(b *testing.B) {
	service := &ragService{}
	vec1 := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	vec2 := []float32{0.2, 0.3, 0.4, 0.5, 0.6}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.cosineSimilarity(vec1, vec2)
	}
}

func BenchmarkRAGService_KeywordSimilarity(b *testing.B) {
	service := &ragService{}
	doc := &Document{
		Content: "Backend engineer with Go programming skills and Docker experience",
	}
	queryWords := []string{"backend", "go", "docker", "programming"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.calculateKeywordSimilarity(doc, queryWords)
	}
}

func BenchmarkRAGService_FindBestMatch(b *testing.B) {
	service := &ragService{}

	// Create test documents
	docs := make([]Document, 100)
	for i := 0; i < 100; i++ {
		docs[i] = Document{
			ID:      fmt.Sprintf("doc_%d", i),
			Content: fmt.Sprintf("Document content %d", i),
			Vector:  []float32{float32(i % 10), float32((i + 1) % 10), float32((i + 2) % 10)},
		}
	}

	queryVector := []float32{5.0, 6.0, 7.0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.findBestMatchVector(docs, queryVector)
	}
}

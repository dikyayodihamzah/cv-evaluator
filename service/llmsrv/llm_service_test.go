package llmsrv

import (
	"errors"
	"testing"
	"time"

	"github.com/dikyayodihamzah/cv-evaluator/config/llmclient"
	"github.com/dikyayodihamzah/cv-evaluator/pkg/resilience"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockOpenAIClient for mocking LLM client
type MockOpenAIClient struct {
	mock.Mock
}

func (m *MockOpenAIClient) ExtractCVInfo(cvContent string) (*llmclient.CVInfo, error) {
	args := m.Called(cvContent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llmclient.CVInfo), args.Error(1)
}

func (m *MockOpenAIClient) ScoreCV(cvInfo *llmclient.CVInfo, jobDesc, ragContext string) (*llmclient.CVEvaluation, error) {
	args := m.Called(cvInfo, jobDesc, ragContext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llmclient.CVEvaluation), args.Error(1)
}

func (m *MockOpenAIClient) EvaluateProject(projectContent, ragContext string) (*llmclient.ProjectEvaluation, error) {
	args := m.Called(projectContent, ragContext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llmclient.ProjectEvaluation), args.Error(1)
}

func (m *MockOpenAIClient) GetEmbedding(text string) ([]float32, error) {
	args := m.Called(text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]float32), args.Error(1)
}

// MockRAGService for mocking RAG service
type MockRAGService struct {
	mock.Mock
}

func (m *MockRAGService) Initialize() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockRAGService) IndexDocument(docType, content string) error {
	args := m.Called(docType, content)
	return args.Error(0)
}

func (m *MockRAGService) GetRelevantContext(contextType, query string) (string, error) {
	args := m.Called(contextType, query)
	return args.String(0), args.Error(1)
}

// LLMServiceTestSuite contains test cases for LLM service
type LLMServiceTestSuite struct {
	suite.Suite
	mockRAGService   *MockRAGService
	mockOpenAIClient *MockOpenAIClient
	llmService       LLMService
}

func (suite *LLMServiceTestSuite) SetupTest() {
	// Use a simple mock implementation instead of external mocks
	suite.mockRAGService = &MockRAGService{}
	suite.mockOpenAIClient = new(MockOpenAIClient)

	// Create service with nil OpenAI client since we can't properly mock the concrete type
	suite.llmService = &llmService{
		ragService:     suite.mockRAGService,
		openaiClient:   nil, // We'll test functions that don't require OpenAI client
		circuitBreaker: resilience.NewCircuitBreaker(3, 30*time.Second),
		retryConfig:    resilience.DefaultRetryConfig(),
	}
}

func TestLLMServiceSuite(t *testing.T) {
	suite.Run(t, new(LLMServiceTestSuite))
}

func (suite *LLMServiceTestSuite) TestEvaluateCandidate_Success() {
	cvContent := "John Doe, Software Engineer with 5 years of experience"
	projectContent := "Built a REST API using Go and PostgreSQL"
	jobDesc := "Looking for a Senior Backend Engineer"

	// Mock CV info extraction
	mockCVInfo := &llmclient.CVInfo{
		Skills:       []string{"Go", "PostgreSQL", "REST API"},
		Experience:   5,
		Projects:     []string{"REST API Project"},
		Achievements: []string{"Led team of 3 developers"},
	}

	// Mock RAG context retrieval
	suite.mockRAGService.On("GetRelevantContext", "cv_evaluation", jobDesc).
		Return("CV evaluation context", nil)

	suite.mockRAGService.On("GetRelevantContext", "project_evaluation", projectContent).
		Return("Project evaluation context", nil)

	// Mock OpenAI client calls
	suite.mockOpenAIClient.On("ExtractCVInfo", cvContent).
		Return(mockCVInfo, nil)

	mockCVEval := &CVEvaluation{
		MatchRate: 0.85,
		Feedback:  "Strong candidate with relevant experience",
	}
	suite.mockOpenAIClient.On("ScoreCV", mock.Anything, jobDesc, "CV evaluation context").
		Return(mockCVEval, nil)

	mockProjectEval := &ProjectEvaluation{
		Score:    8.5,
		Feedback: "Well-structured project with good technical choices",
	}
	suite.mockOpenAIClient.On("EvaluateProject", projectContent, "Project evaluation context").
		Return(mockProjectEval, nil)

	// Execute test
	result, err := suite.llmService.EvaluateCandidate(cvContent, projectContent, jobDesc)

	// Assertions
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), 0.85, result.CVMatchRate)
	assert.Equal(suite.T(), 8.5, result.ProjectScore)
	assert.Contains(suite.T(), result.OverallSummary, "Excellent candidate fit")
}

func (suite *LLMServiceTestSuite) TestEvaluateCandidate_CVExtractionError() {
	cvContent := "Invalid CV content"
	projectContent := "Valid project content"
	jobDesc := "Job description"

	// Mock CV info extraction failure
	suite.mockOpenAIClient.On("ExtractCVInfo", cvContent).
		Return(nil, errors.New("failed to extract CV info"))

	// Execute test
	result, err := suite.llmService.EvaluateCandidate(cvContent, projectContent, jobDesc)

	// Assertions
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to extract CV info")
}

func (suite *LLMServiceTestSuite) TestEvaluateCVOnly_Success() {
	cvContent := "John Doe, Software Engineer with 5 years of experience"
	jobDesc := "Looking for a Senior Backend Engineer"

	// Mock CV info extraction
	mockCVInfo := &llmclient.CVInfo{
		Skills:       []string{"Go", "PostgreSQL"},
		Experience:   5,
		Projects:     []string{"REST API Project"},
		Achievements: []string{"Led team of 3 developers"},
	}

	// Mock RAG context retrieval
	suite.mockRAGService.On("GetRelevantContext", "cv_evaluation", jobDesc).
		Return("CV evaluation context", nil)

	// Mock OpenAI client calls
	suite.mockOpenAIClient.On("ExtractCVInfo", cvContent).
		Return(mockCVInfo, nil)

	mockCVEval := &CVEvaluation{
		MatchRate: 0.75,
		Feedback:  "Good candidate with some relevant experience",
	}
	suite.mockOpenAIClient.On("ScoreCV", mock.Anything, jobDesc, "CV evaluation context").
		Return(mockCVEval, nil)

	// Execute test
	result, err := suite.llmService.EvaluateCVOnly(cvContent, jobDesc)

	// Assertions
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), 0.75, result.CVMatchRate)
	assert.Equal(suite.T(), 0.0, result.ProjectScore) // Default project score
	assert.Contains(suite.T(), result.ProjectFeedback, "No project file provided")
	assert.Contains(suite.T(), result.OverallSummary, "Good CV match")
}

func (suite *LLMServiceTestSuite) TestEvaluateCVOnly_RAGFailure() {
	cvContent := "John Doe, Software Engineer"
	jobDesc := "Backend Engineer position"

	// Mock CV info extraction
	mockCVInfo := &llmclient.CVInfo{
		Skills:     []string{"Go"},
		Experience: 3,
	}

	// Mock RAG context retrieval failure
	suite.mockRAGService.On("GetRelevantContext", "cv_evaluation", jobDesc).
		Return("", errors.New("RAG service unavailable"))

	// Mock OpenAI client calls
	suite.mockOpenAIClient.On("ExtractCVInfo", cvContent).
		Return(mockCVInfo, nil)

	mockCVEval := &CVEvaluation{
		MatchRate: 0.6,
		Feedback:  "Moderate candidate fit",
	}
	// Note: RAG context should be empty string due to failure
	suite.mockOpenAIClient.On("ScoreCV", mock.Anything, jobDesc, "").
		Return(mockCVEval, nil)

	// Execute test
	result, err := suite.llmService.EvaluateCVOnly(cvContent, jobDesc)

	// Assertions
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), 0.6, result.CVMatchRate)
}

func (suite *LLMServiceTestSuite) TestExtractCVInfo_Success() {
	cvContent := "Experienced software engineer"

	mockCVInfo := &llmclient.CVInfo{
		Skills:       []string{"Go", "Python"},
		Experience:   5,
		Projects:     []string{"Web API", "CLI Tool"},
		Achievements: []string{"Performance improvement"},
	}

	suite.mockOpenAIClient.On("ExtractCVInfo", cvContent).
		Return(mockCVInfo, nil)

	// Create service with mock
	service := &llmService{
		openaiClient:   nil, // Cannot mock concrete type
		circuitBreaker: resilience.NewCircuitBreaker(3, 30*time.Second),
		retryConfig:    resilience.DefaultRetryConfig(),
	}

	result, err := service.ExtractCVInfo(cvContent)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), []string{"Go", "Python"}, result.Skills)
	assert.Equal(suite.T(), 5, result.Experience)
}

func (suite *LLMServiceTestSuite) TestScoreCV_Success() {
	cvInfo := &CVInfo{
		Skills:     []string{"Go", "Docker"},
		Experience: 4,
	}
	jobDesc := "Backend engineer with Go experience"
	ragContext := "Evaluation context"

	mockEval := &CVEvaluation{
		MatchRate: 0.8,
		Feedback:  "Strong technical match",
	}

	suite.mockOpenAIClient.On("ScoreCV", mock.Anything, jobDesc, ragContext).
		Return(mockEval, nil)

	service := &llmService{
		openaiClient:   nil, // Cannot mock concrete type
		circuitBreaker: resilience.NewCircuitBreaker(3, 30*time.Second),
		retryConfig:    resilience.DefaultRetryConfig(),
	}

	result, err := service.ScoreCV(cvInfo, jobDesc, ragContext)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), 0.8, result.MatchRate)
	assert.Equal(suite.T(), "Strong technical match", result.Feedback)
}

func (suite *LLMServiceTestSuite) TestEvaluateProject_Success() {
	projectContent := "REST API with comprehensive testing"
	ragContext := "Project evaluation guidelines"

	mockEval := &ProjectEvaluation{
		Score:    9.0,
		Feedback: "Excellent project structure and testing coverage",
	}

	suite.mockOpenAIClient.On("EvaluateProject", projectContent, ragContext).
		Return(mockEval, nil)

	service := &llmService{
		openaiClient:   nil, // Cannot mock concrete type
		circuitBreaker: resilience.NewCircuitBreaker(3, 30*time.Second),
		retryConfig:    resilience.DefaultRetryConfig(),
	}

	result, err := service.EvaluateProject(projectContent, ragContext)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), 9.0, result.Score)
}

// Test helper functions
func (suite *LLMServiceTestSuite) TestGenerateOverallSummary() {
	service := &llmService{}

	testCases := []struct {
		name           string
		cvEval         *CVEvaluation
		projectEval    *ProjectEvaluation
		expectedPhrase string
	}{
		{
			name:           "excellent candidate",
			cvEval:         &CVEvaluation{MatchRate: 0.9},
			projectEval:    &ProjectEvaluation{Score: 9.0},
			expectedPhrase: "Excellent candidate fit",
		},
		{
			name:           "good candidate",
			cvEval:         &CVEvaluation{MatchRate: 0.8},
			projectEval:    &ProjectEvaluation{Score: 7.5},
			expectedPhrase: "Good candidate fit",
		},
		{
			name:           "moderate candidate",
			cvEval:         &CVEvaluation{MatchRate: 0.6},
			projectEval:    &ProjectEvaluation{Score: 6.0},
			expectedPhrase: "Moderate candidate fit",
		},
		{
			name:           "limited candidate",
			cvEval:         &CVEvaluation{MatchRate: 0.4},
			projectEval:    &ProjectEvaluation{Score: 4.0},
			expectedPhrase: "Limited candidate fit",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			result := service.generateOverallSummary(tc.cvEval, tc.projectEval)
			assert.Contains(t, result, tc.expectedPhrase)
		})
	}
}

func (suite *LLMServiceTestSuite) TestGenerateCVOnlySummary() {
	service := &llmService{}

	testCases := []struct {
		name           string
		cvEval         *CVEvaluation
		expectedPhrase string
	}{
		{
			name:           "strong CV match",
			cvEval:         &CVEvaluation{MatchRate: 0.85},
			expectedPhrase: "Strong CV match",
		},
		{
			name:           "good CV match",
			cvEval:         &CVEvaluation{MatchRate: 0.75},
			expectedPhrase: "Good CV match",
		},
		{
			name:           "moderate CV match",
			cvEval:         &CVEvaluation{MatchRate: 0.65},
			expectedPhrase: "Moderate CV match",
		},
		{
			name:           "limited CV match",
			cvEval:         &CVEvaluation{MatchRate: 0.45},
			expectedPhrase: "Limited CV match",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			result := service.generateCVOnlySummary(tc.cvEval)
			assert.Contains(t, result, tc.expectedPhrase)
			assert.Contains(t, result, "Project evaluation not available")
		})
	}
}

// Error handling tests
func (suite *LLMServiceTestSuite) TestEvaluateCandidate_CVScoringError() {
	cvContent := "Valid CV"
	projectContent := "Valid project"
	jobDesc := "Job description"

	// Mock successful CV extraction
	mockCVInfo := &llmclient.CVInfo{Skills: []string{"Go"}}
	suite.mockOpenAIClient.On("ExtractCVInfo", cvContent).
		Return(mockCVInfo, nil)

	// Mock RAG success
	suite.mockRAGService.On("GetRelevantContext", "cv_evaluation", jobDesc).
		Return("context", nil)
	suite.mockRAGService.On("GetRelevantContext", "project_evaluation", projectContent).
		Return("context", nil)

	// Mock CV scoring failure
	suite.mockOpenAIClient.On("ScoreCV", mock.Anything, jobDesc, "context").
		Return(nil, errors.New("scoring failed"))

	result, err := suite.llmService.EvaluateCandidate(cvContent, projectContent, jobDesc)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to score CV")
}

func (suite *LLMServiceTestSuite) TestEvaluateCandidate_ProjectEvaluationError() {
	cvContent := "Valid CV"
	projectContent := "Valid project"
	jobDesc := "Job description"

	// Mock successful CV processing
	mockCVInfo := &llmclient.CVInfo{Skills: []string{"Go"}}
	suite.mockOpenAIClient.On("ExtractCVInfo", cvContent).
		Return(mockCVInfo, nil)

	mockCVEval := &CVEvaluation{MatchRate: 0.8, Feedback: "Good"}
	suite.mockOpenAIClient.On("ScoreCV", mock.Anything, jobDesc, "context").
		Return(mockCVEval, nil)

	// Mock RAG success
	suite.mockRAGService.On("GetRelevantContext", "cv_evaluation", jobDesc).
		Return("context", nil)
	suite.mockRAGService.On("GetRelevantContext", "project_evaluation", projectContent).
		Return("context", nil)

	// Mock project evaluation failure
	suite.mockOpenAIClient.On("EvaluateProject", projectContent, "context").
		Return(nil, errors.New("project evaluation failed"))

	result, err := suite.llmService.EvaluateCandidate(cvContent, projectContent, jobDesc)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to evaluate project")
}

// Benchmark tests
func BenchmarkLLMService_GenerateOverallSummary(b *testing.B) {
	service := &llmService{}
	cvEval := &CVEvaluation{MatchRate: 0.8}
	projectEval := &ProjectEvaluation{Score: 8.0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.generateOverallSummary(cvEval, projectEval)
	}
}

func BenchmarkLLMService_GenerateCVOnlySummary(b *testing.B) {
	service := &llmService{}
	cvEval := &CVEvaluation{MatchRate: 0.75}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.generateCVOnlySummary(cvEval)
	}
}

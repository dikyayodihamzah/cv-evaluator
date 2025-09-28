package evalsrv

import (
	"context"
	"mime/multipart"
	"testing"

	"github.com/dikyayodihamzah/cv-evaluator/pkg/model/cvweb"
	"github.com/dikyayodihamzah/cv-evaluator/service/llmsrv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockFileService for mocking file operations
type MockFileService struct {
	mock.Mock
}

func (m *MockFileService) Upload(ctx context.Context, path string, files []*multipart.FileHeader) ([]string, error) {
	args := m.Called(ctx, path, files)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockFileService) ExtractText(filePath string) (string, error) {
	args := m.Called(filePath)
	return args.String(0), args.Error(1)
}

// MockLLMService for mocking LLM operations
type MockLLMService struct {
	mock.Mock
}

func (m *MockLLMService) EvaluateCandidate(cvContent, projectContent, jobDesc string) (*cvweb.CVResponse, error) {
	args := m.Called(cvContent, projectContent, jobDesc)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cvweb.CVResponse), args.Error(1)
}

func (m *MockLLMService) EvaluateCVOnly(cvContent, jobDesc string) (*cvweb.CVResponse, error) {
	args := m.Called(cvContent, jobDesc)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cvweb.CVResponse), args.Error(1)
}

func (m *MockLLMService) ExtractCVInfo(cvContent string) (*llmsrv.CVInfo, error) {
	args := m.Called(cvContent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llmsrv.CVInfo), args.Error(1)
}

func (m *MockLLMService) ScoreCV(cvInfo *llmsrv.CVInfo, jobDesc, ragContext string) (*llmsrv.CVEvaluation, error) {
	args := m.Called(cvInfo, jobDesc, ragContext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llmsrv.CVEvaluation), args.Error(1)
}

func (m *MockLLMService) EvaluateProject(projectContent, ragContext string) (*llmsrv.ProjectEvaluation, error) {
	args := m.Called(projectContent, ragContext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llmsrv.ProjectEvaluation), args.Error(1)
}

// EvaluationServiceTestSuite contains test cases for evaluation service
type EvaluationServiceTestSuite struct {
	suite.Suite
	mockFileService *MockFileService
	mockLLMService  *MockLLMService
	evalService     EvaluationService
}

func (suite *EvaluationServiceTestSuite) SetupTest() {
	suite.mockFileService = &MockFileService{}
	suite.mockLLMService = &MockLLMService{}

	suite.evalService = New(suite.mockFileService, suite.mockLLMService)
}

func TestEvaluationServiceSuite(t *testing.T) {
	suite.Run(t, new(EvaluationServiceTestSuite))
}

func (suite *EvaluationServiceTestSuite) TestGetResult_JobNotFound() {
	nonExistentJobID := "non-existent-job-id"

	result, err := suite.evalService.GetResult(nonExistentJobID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Equal(suite.T(), ErrJobNotFound, err)
}

func (suite *EvaluationServiceTestSuite) TestGetResult_CompletedJob() {
	// Start evaluation
	req := cvweb.EvaluationRequest{
		CVFile:         "cv-files/test-cv.pdf",
		JobDescription: "Test job description",
	}

	jobID, err := suite.evalService.StartEvaluation(req)
	assert.NoError(suite.T(), err)

	// Manually set job as completed for testing
	service := suite.evalService.(*evaluationService)
	service.mutex.Lock()
	job := service.jobs[jobID]
	job.Status = "completed"
	job.Result = &cvweb.CVResponse{
		CVMatchRate:     0.85,
		CVFeedback:      "Good match",
		ProjectScore:    8.5,
		ProjectFeedback: "Excellent project",
		OverallSummary:  "Recommended candidate",
	}
	service.mutex.Unlock()

	// Get result
	result, err := suite.evalService.GetResult(jobID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "completed", result.Status)
	assert.NotNil(suite.T(), result.Result)

	cvResponse := result.Result
	assert.Equal(suite.T(), 0.85, cvResponse.CVMatchRate)
	assert.Equal(suite.T(), 8.5, cvResponse.ProjectScore)
}

func (suite *EvaluationServiceTestSuite) TestGetResult_FailedJob() {
	// Start evaluation
	req := cvweb.EvaluationRequest{
		CVFile:         "cv-files/test-cv.pdf",
		JobDescription: "Test job description",
	}

	jobID, err := suite.evalService.StartEvaluation(req)
	assert.NoError(suite.T(), err)

	// Manually set job as failed for testing
	service := suite.evalService.(*evaluationService)
	service.mutex.Lock()
	job := service.jobs[jobID]
	job.Status = "failed"
	errorMsg := "Test error message"
	job.Error = &errorMsg
	service.mutex.Unlock()

	// Get result
	result, err := suite.evalService.GetResult(jobID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "failed", result.Status)
	assert.NotNil(suite.T(), result.Error)
	assert.Equal(suite.T(), "Test error message", *result.Error)
}

func (suite *EvaluationServiceTestSuite) TestMarkJobFailed() {
	// Start evaluation
	req := cvweb.EvaluationRequest{
		CVFile:         "cv-files/test-cv.pdf",
		JobDescription: "Test job description",
	}

	jobID, err := suite.evalService.StartEvaluation(req)
	assert.NoError(suite.T(), err)

	// Test marking job as failed
	errorMsg := "Test error message"
	service := suite.evalService.(*evaluationService)
	service.markJobFailed(jobID, errorMsg)

	// Verify job was marked as failed
	service.mutex.RLock()
	job := service.jobs[jobID]
	service.mutex.RUnlock()

	assert.Equal(suite.T(), "failed", job.Status)
	assert.NotNil(suite.T(), job.Error)
	assert.Equal(suite.T(), errorMsg, *job.Error)
}

// Concurrent access tests
func (suite *EvaluationServiceTestSuite) TestConcurrentJobAccess() {
	// Start multiple evaluations concurrently
	numJobs := 10
	jobIDs := make([]string, numJobs)

	for i := 0; i < numJobs; i++ {
		req := cvweb.EvaluationRequest{
			CVFile:         "cv-files/test-cv.pdf",
			JobDescription: "Test job description",
		}

		jobID, err := suite.evalService.StartEvaluation(req)
		assert.NoError(suite.T(), err)
		jobIDs[i] = jobID
	}

	// Verify all jobs exist
	for _, jobID := range jobIDs {
		result, err := suite.evalService.GetResult(jobID)
		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.Equal(suite.T(), jobID, result.ID)
	}
}

func (suite *EvaluationServiceTestSuite) TestJobDataIntegrity() {
	req := cvweb.EvaluationRequest{
		CVFile:             "cv-files/test-cv.pdf",
		ProjectFile:        "cv-files/test-project.pdf",
		JobDescription:     "Backend engineer position",
		JobDescriptionFile: "cv-files/job-desc.txt",
	}

	jobID, err := suite.evalService.StartEvaluation(req)
	assert.NoError(suite.T(), err)

	// Verify job data integrity
	service := suite.evalService.(*evaluationService)
	service.mutex.RLock()
	job := service.jobs[jobID]
	service.mutex.RUnlock()

	assert.Equal(suite.T(), req.CVFile, job.Request.CVFile)
	assert.Equal(suite.T(), req.ProjectFile, job.Request.ProjectFile)
	assert.Equal(suite.T(), req.JobDescription, job.Request.JobDescription)
	assert.Equal(suite.T(), req.JobDescriptionFile, job.Request.JobDescriptionFile)
	assert.False(suite.T(), job.Created.IsZero())
	assert.False(suite.T(), job.Updated.IsZero())
}

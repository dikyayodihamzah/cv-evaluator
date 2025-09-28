package evalsrv

import (
	"errors"
	"sync"
	"time"

	"github.com/dikyayodihamzah/cv-evaluator/pkg/model"
	"github.com/dikyayodihamzah/cv-evaluator/pkg/model/cvweb"
	"github.com/dikyayodihamzah/cv-evaluator/service/filesrv"
	"github.com/dikyayodihamzah/cv-evaluator/service/llmsrv"
	"github.com/google/uuid"
)

type EvaluationService interface {
	StartEvaluation(req cvweb.EvaluationRequest) (string, error)
	GetResult(jobID string) (*model.BaseResponse, error)
}

type evaluationService struct {
	fileService filesrv.FileService
	llmService  llmsrv.LLMService
	jobs        map[string]*Job
	mutex       sync.RWMutex
}

type Job struct {
	ID      string
	Status  string
	Request cvweb.EvaluationRequest
	Result  *cvweb.CVResponse
	Error   *string
	Created time.Time
	Updated time.Time
}

func New(fileService filesrv.FileService, llmService llmsrv.LLMService) EvaluationService {
	return &evaluationService{
		fileService: fileService,
		llmService:  llmService,
		jobs:        make(map[string]*Job),
	}
}

func (s *evaluationService) StartEvaluation(req cvweb.EvaluationRequest) (string, error) {
	jobID := uuid.New().String()

	job := &Job{
		ID:      jobID,
		Status:  "queued",
		Request: req,
		Created: time.Now(),
		Updated: time.Now(),
	}

	s.mutex.Lock()
	s.jobs[jobID] = job
	s.mutex.Unlock()

	// Start async processing
	go s.processEvaluation(jobID)

	return jobID, nil
}

func (s *evaluationService) GetResult(jobID string) (*model.BaseResponse, error) {
	s.mutex.RLock()
	job, exists := s.jobs[jobID]
	s.mutex.RUnlock()

	if !exists {
		return nil, ErrJobNotFound
	}

	response := &model.BaseResponse{
		ID:     job.ID,
		Status: job.Status,
	}

	switch job.Status {
	case "completed":
		response.Result = job.Result
	case "failed":
		response.Error = job.Error
	}

	return response, nil
}

func (s *evaluationService) processEvaluation(jobID string) {
	s.updateJobStatus(jobID, "processing")

	// Simulate processing time
	time.Sleep(2 * time.Second)

	s.mutex.RLock()
	job := s.jobs[jobID]
	s.mutex.RUnlock()

	// Extract content from all CV file
	cvContent, err := s.fileService.ExtractText(job.Request.CVFile)
	if err != nil {
		s.markJobFailed(jobID, "Failed to extract CV content from file "+job.Request.CVFile+": "+err.Error())
		return
	}

	// Get job description content
	var jobDescription string
	if job.Request.JobDescriptionFile != "" {
		// Extract job description from file
		jobDescContent, err := s.fileService.ExtractText(job.Request.JobDescriptionFile)
		if err != nil {
			s.markJobFailed(jobID, "Failed to extract job description content: "+err.Error())
			return
		}
		jobDescription = jobDescContent
	} else {
		// Use provided job description text
		jobDescription = job.Request.JobDescription
	}

	// For project content, we'll extract it from the CV files (treating them as complete documents)
	// In a real implementation, you might want to separate CV and project content
	projectContent := cvContent // Using CV content as project content for now

	// Process with LLM
	result, err := s.llmService.EvaluateCandidate(cvContent, projectContent, jobDescription)
	if err != nil {
		s.markJobFailed(jobID, "LLM evaluation failed: "+err.Error())
		return
	}

	s.mutex.Lock()
	job.Status = "completed"
	job.Result = result
	job.Updated = time.Now()
	s.mutex.Unlock()
}

func (s *evaluationService) updateJobStatus(jobID, status string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if job, exists := s.jobs[jobID]; exists {
		job.Status = status
		job.Updated = time.Now()
	}
}

func (s *evaluationService) markJobFailed(jobID, errorMsg string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if job, exists := s.jobs[jobID]; exists {
		job.Status = "failed"
		job.Error = &errorMsg
		job.Updated = time.Now()
	}
}

var ErrJobNotFound = errors.New("job not found")

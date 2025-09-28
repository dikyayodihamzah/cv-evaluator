package evalsrv

import (
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

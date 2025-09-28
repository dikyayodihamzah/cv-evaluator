package evalsrv

import (
	"errors"
	"time"

	"github.com/dikyayodihamzah/cv-evaluator/pkg/log"
	"github.com/dikyayodihamzah/cv-evaluator/pkg/model/cvweb"
)

func (s *evaluationService) processEvaluation(jobID string) {
	logger := log.WithContext("EvaluationService", "ProcessEvaluation")
	start := time.Now()

	logger.Info("Starting evaluation processing for job %s", jobID)
	s.updateJobStatus(jobID, "processing")

	// Simulate processing time
	time.Sleep(2 * time.Second)

	s.mutex.RLock()
	job := s.jobs[jobID]
	s.mutex.RUnlock()

	logger.Debug("Processing evaluation request - CV file: %s, Project file: %s",
		job.Request.CVFile, job.Request.ProjectFile)

	// Extract content from all CV file
	logger.Debug("Extracting content from CV file: %s", job.Request.CVFile)
	cvContent, err := s.fileService.ExtractText(job.Request.CVFile)
	if err != nil {
		logger.Error("Failed to extract CV content from file %s: %v", job.Request.CVFile, err)
		s.markJobFailed(jobID, "Failed to extract CV content from file "+job.Request.CVFile+": "+err.Error())
		return
	}
	logger.Debug("CV content extracted successfully: %d characters", len(cvContent))

	// Get job description content
	logger.Debug("Processing job description")
	var jobDescription string
	if job.Request.JobDescriptionFile != "" {
		logger.Debug("Extracting job description from file: %s", job.Request.JobDescriptionFile)
		// Extract job description from file
		jobDescContent, err := s.fileService.ExtractText(job.Request.JobDescriptionFile)
		if err != nil {
			logger.Error("Failed to extract job description from file %s: %v", job.Request.JobDescriptionFile, err)
			s.markJobFailed(jobID, "Failed to extract job description content: "+err.Error())
			return
		}
		jobDescription = jobDescContent
		logger.Debug("Job description extracted from file: %d characters", len(jobDescription))
	} else {
		// Use provided job description text
		jobDescription = job.Request.JobDescription
		logger.Debug("Using provided job description text: %d characters", len(jobDescription))
	}

	// Get project content - only if project file is provided
	logger.Debug("Determining evaluation mode based on project file presence")
	var result *cvweb.CVResponse

	if job.Request.ProjectFile != "" {
		logger.Info("Full evaluation mode: CV + Project file")
		logger.Debug("Extracting project content from file: %s", job.Request.ProjectFile)
		// Extract project content from separate project file
		projContent, err := s.fileService.ExtractText(job.Request.ProjectFile)
		if err != nil {
			logger.Error("Failed to extract project content from file %s: %v", job.Request.ProjectFile, err)
			s.markJobFailed(jobID, "Failed to extract project content from file "+job.Request.ProjectFile+": "+err.Error())
			return
		}
		logger.Debug("Project content extracted: %d characters", len(projContent))

		// Process with LLM (full evaluation including project)
		logger.Debug("Starting full LLM evaluation (CV + Project)")
		result, err = s.llmService.EvaluateCandidate(cvContent, projContent, jobDescription)
		if err != nil {
			logger.Error("Full LLM evaluation failed: %v", err)
			s.markJobFailed(jobID, "LLM evaluation failed: "+err.Error())
			return
		}
		logger.Info("Full evaluation completed successfully")
	} else {
		logger.Info("CV-only evaluation mode: No project file provided")
		// No project file provided - evaluate only CV and provide default project info
		logger.Debug("Starting CV-only LLM evaluation")
		var err error
		result, err = s.llmService.EvaluateCVOnly(cvContent, jobDescription)
		if err != nil {
			logger.Error("CV-only LLM evaluation failed: %v", err)
			s.markJobFailed(jobID, "LLM evaluation failed: "+err.Error())
			return
		}
		logger.Info("CV-only evaluation completed successfully")
	}

	logger.Debug("Finalizing evaluation job")
	s.mutex.Lock()
	job.Status = "completed"
	job.Result = result
	job.Updated = time.Now()
	s.mutex.Unlock()

	logger.Info("Evaluation job %s completed successfully", jobID)
	logger.WithDuration(start)
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

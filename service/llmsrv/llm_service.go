package llmsrv

import (
	"fmt"
	"time"

	"github.com/dikyayodihamzah/cv-evaluator/config/llmclient"
	"github.com/dikyayodihamzah/cv-evaluator/pkg/log"
	"github.com/dikyayodihamzah/cv-evaluator/pkg/model/cvweb"
	"github.com/dikyayodihamzah/cv-evaluator/pkg/resilience"
	"github.com/dikyayodihamzah/cv-evaluator/service/ragsrv"
)

type LLMService interface {
	EvaluateCandidate(cvContent, projectContent, jobDesc string) (*cvweb.CVResponse, error)
	EvaluateCVOnly(cvContent, jobDesc string) (*cvweb.CVResponse, error)
	ExtractCVInfo(cvContent string) (*CVInfo, error)
	ScoreCV(cvInfo *CVInfo, jobDesc string, ragContext string) (*CVEvaluation, error)
	EvaluateProject(projectContent, ragContext string) (*ProjectEvaluation, error)
}

type llmService struct {
	ragService     ragsrv.RAGService
	openaiClient   *llmclient.OpenAIClient
	circuitBreaker *resilience.CircuitBreaker
	retryConfig    resilience.RetryConfig
}

type CVInfo struct {
	Skills       []string `json:"skills"`
	Experience   int      `json:"experience_years"`
	Projects     []string `json:"projects"`
	Achievements []string `json:"achievements"`
}

type CVEvaluation struct {
	MatchRate float64 `json:"match_rate"`
	Feedback  string  `json:"feedback"`
}

type ProjectEvaluation struct {
	Score    float64 `json:"score"`
	Feedback string  `json:"feedback"`
}

func New(ragService ragsrv.RAGService) LLMService {
	return &llmService{
		ragService:     ragService,
		openaiClient:   llmclient.NewOpenAIClient(),
		circuitBreaker: resilience.NewCircuitBreaker(3, 30*time.Second),
		retryConfig:    resilience.DefaultRetryConfig(),
	}
}

func (s *llmService) EvaluateCandidate(cvContent, projectContent, jobDesc string) (*cvweb.CVResponse, error) {
	logger := log.WithContext("LLMService", "EvaluateCandidate")
	start := time.Now()

	logger.Info("Starting full candidate evaluation (CV + Project)")
	logger.Debug("Content sizes - CV: %d chars, Project: %d chars, Job desc: %d chars",
		len(cvContent), len(projectContent), len(jobDesc))

	// Step 1: Extract structured info from CV
	logger.Debug("Step 1: Extracting structured CV information")
	cvInfo, err := s.ExtractCVInfo(cvContent)
	if err != nil {
		logger.Error("Failed to extract CV info: %v", err)
		return nil, fmt.Errorf("failed to extract CV info: %w", err)
	}
	logger.Debug("CV info extracted - Skills: %d, Experience: %d years, Projects: %d",
		len(cvInfo.Skills), cvInfo.Experience, len(cvInfo.Projects))

	// Step 2: Get relevant context from RAG
	logger.Debug("Step 2: Retrieving RAG context")
	cvRagContext, err := s.ragService.GetRelevantContext("cv_evaluation", jobDesc)
	if err != nil {
		logger.Warn("Failed to get CV RAG context, continuing without: %v", err)
		cvRagContext = "" // Continue without RAG if it fails
	} else {
		logger.Debug("CV RAG context retrieved: %d characters", len(cvRagContext))
	}

	projectRagContext, err := s.ragService.GetRelevantContext("project_evaluation", projectContent)
	if err != nil {
		logger.Warn("Failed to get project RAG context, continuing without: %v", err)
		projectRagContext = "" // Continue without RAG if it fails
	} else {
		logger.Debug("Project RAG context retrieved: %d characters", len(projectRagContext))
	}

	// Step 3: Score CV against job requirements
	logger.Debug("Step 3: Scoring CV against job requirements")
	cvEval, err := s.ScoreCV(cvInfo, jobDesc, cvRagContext)
	if err != nil {
		logger.Error("Failed to score CV: %v", err)
		return nil, fmt.Errorf("failed to score CV: %w", err)
	}
	logger.Debug("CV scoring completed - Match rate: %.2f", cvEval.MatchRate)

	// Step 4: Evaluate project
	logger.Debug("Step 4: Evaluating project")
	projectEval, err := s.EvaluateProject(projectContent, projectRagContext)
	if err != nil {
		logger.Error("Failed to evaluate project: %v", err)
		return nil, fmt.Errorf("failed to evaluate project: %w", err)
	}
	logger.Debug("Project evaluation completed - Score: %.1f", projectEval.Score)

	// Step 5: Generate overall summary
	logger.Debug("Step 5: Generating overall summary")
	overallSummary := s.generateOverallSummary(cvEval, projectEval)

	logger.Info("Full candidate evaluation completed successfully")
	logger.WithDuration(start)

	return &cvweb.CVResponse{
		CVMatchRate:     cvEval.MatchRate,
		CVFeedback:      cvEval.Feedback,
		ProjectScore:    projectEval.Score,
		ProjectFeedback: projectEval.Feedback,
		OverallSummary:  overallSummary,
	}, nil
}

func (s *llmService) EvaluateCVOnly(cvContent, jobDesc string) (*cvweb.CVResponse, error) {
	logger := log.WithContext("LLMService", "EvaluateCVOnly")
	start := time.Now()

	logger.Info("Starting CV-only evaluation (no project file)")
	logger.Debug("Content sizes - CV: %d chars, Job desc: %d chars", len(cvContent), len(jobDesc))

	// Step 1: Extract structured info from CV
	logger.Debug("Step 1: Extracting structured CV information")
	cvInfo, err := s.ExtractCVInfo(cvContent)
	if err != nil {
		logger.Error("Failed to extract CV info: %v", err)
		return nil, fmt.Errorf("failed to extract CV info: %w", err)
	}
	logger.Debug("CV info extracted - Skills: %d, Experience: %d years, Projects: %d",
		len(cvInfo.Skills), cvInfo.Experience, len(cvInfo.Projects))

	// Step 2: Get relevant context from RAG for CV evaluation
	logger.Debug("Step 2: Retrieving RAG context for CV evaluation")
	cvRagContext, err := s.ragService.GetRelevantContext("cv_evaluation", jobDesc)
	if err != nil {
		logger.Warn("Failed to get CV RAG context, continuing without: %v", err)
		cvRagContext = "" // Continue without RAG if it fails
	} else {
		logger.Debug("CV RAG context retrieved: %d characters", len(cvRagContext))
	}

	// Step 3: Score CV against job requirements
	logger.Debug("Step 3: Scoring CV against job requirements")
	cvEval, err := s.ScoreCV(cvInfo, jobDesc, cvRagContext)
	if err != nil {
		logger.Error("Failed to score CV: %v", err)
		return nil, fmt.Errorf("failed to score CV: %w", err)
	}
	logger.Debug("CV scoring completed - Match rate: %.2f", cvEval.MatchRate)

	// Step 4: Provide default project information since no project file was provided
	logger.Debug("Step 4: Setting default project information (no project file)")
	defaultProjectScore := 0.0
	defaultProjectFeedback := "No project file provided for evaluation. To get a comprehensive project assessment, please upload a separate project file containing technical documentation, code samples, or project reports."

	// Step 5: Generate CV-only summary
	logger.Debug("Step 5: Generating CV-only summary")
	overallSummary := s.generateCVOnlySummary(cvEval)

	logger.Info("CV-only evaluation completed successfully")
	logger.WithDuration(start)

	return &cvweb.CVResponse{
		CVMatchRate:     cvEval.MatchRate,
		CVFeedback:      cvEval.Feedback,
		ProjectScore:    defaultProjectScore,
		ProjectFeedback: defaultProjectFeedback,
		OverallSummary:  overallSummary,
	}, nil
}

func (s *llmService) ExtractCVInfo(cvContent string) (*CVInfo, error) {
	var result *CVInfo

	// Use circuit breaker and retry logic
	if err := s.circuitBreaker.Execute(func() error {
		return resilience.WithExponentialBackoff(s.retryConfig, func() error {
			// Call real OpenAI API
			openaiResult, err := s.openaiClient.ExtractCVInfo(cvContent)
			if err != nil {
				return fmt.Errorf("OpenAI API call failed: %w", err)
			}

			// Convert from OpenAI client result to service result
			result = &CVInfo{
				Skills:       openaiResult.Skills,
				Experience:   openaiResult.Experience,
				Projects:     openaiResult.Projects,
				Achievements: openaiResult.Achievements,
			}

			return nil
		})
	}); err != nil {
		return nil, fmt.Errorf("failed to extract CV info: %w", err)
	}

	return result, nil
}

func (s *llmService) ScoreCV(cvInfo *CVInfo, jobDesc, ragContext string) (*CVEvaluation, error) {
	var result *CVEvaluation

	// Use circuit breaker and retry logic
	if err := s.circuitBreaker.Execute(func() error {
		return resilience.WithExponentialBackoff(s.retryConfig, func() error {
			// Convert to OpenAI client format
			openaiCVInfo := &llmclient.CVInfo{
				Skills:       cvInfo.Skills,
				Experience:   cvInfo.Experience,
				Projects:     cvInfo.Projects,
				Achievements: cvInfo.Achievements,
			}

			// Call real OpenAI API
			openaiResult, err := s.openaiClient.ScoreCV(openaiCVInfo, jobDesc, ragContext)
			if err != nil {
				return fmt.Errorf("OpenAI API call failed: %w", err)
			}

			// Convert from OpenAI client result to service result
			result = &CVEvaluation{
				MatchRate: openaiResult.MatchRate,
				Feedback:  openaiResult.Feedback,
			}

			return nil
		})
	}); err != nil {
		return nil, fmt.Errorf("failed to score CV: %w", err)
	}

	return result, nil
}

func (s *llmService) EvaluateProject(projectContent, ragContext string) (*ProjectEvaluation, error) {
	var result *ProjectEvaluation

	// Use circuit breaker and retry logic
	if err := s.circuitBreaker.Execute(func() error {
		return resilience.WithExponentialBackoff(s.retryConfig, func() error {
			// Call real OpenAI API
			openaiResult, err := s.openaiClient.EvaluateProject(projectContent, ragContext)
			if err != nil {
				return fmt.Errorf("OpenAI API call failed: %w", err)
			}

			// Convert from OpenAI client result to service result
			result = &ProjectEvaluation{
				Score:    openaiResult.Score,
				Feedback: openaiResult.Feedback,
			}

			return nil
		})
	}); err != nil {
		return nil, fmt.Errorf("failed to evaluate project: %w", err)
	}

	return result, nil
}

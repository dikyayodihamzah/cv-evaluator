package llmsrv

import (
	"fmt"
	"time"

	"github.com/dikyayodihamzah/cv-evaluator/config/llmclient"
	"github.com/dikyayodihamzah/cv-evaluator/pkg/model/cvweb"
	"github.com/dikyayodihamzah/cv-evaluator/pkg/resilience"
	"github.com/dikyayodihamzah/cv-evaluator/service/ragsrv"
)

type LLMService interface {
	EvaluateCandidate(cvContent, projectContent, jobDesc string) (*cvweb.CVResponse, error)
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
	// Step 1: Extract structured info from CV
	cvInfo, err := s.ExtractCVInfo(cvContent)
	if err != nil {
		return nil, fmt.Errorf("failed to extract CV info: %w", err)
	}

	// Step 2: Get relevant context from RAG
	cvRagContext, err := s.ragService.GetRelevantContext("cv_evaluation", jobDesc)
	if err != nil {
		cvRagContext = "" // Continue without RAG if it fails
	}

	projectRagContext, err := s.ragService.GetRelevantContext("project_evaluation", projectContent)
	if err != nil {
		projectRagContext = "" // Continue without RAG if it fails
	}

	// Step 3: Score CV against job requirements
	cvEval, err := s.ScoreCV(cvInfo, jobDesc, cvRagContext)
	if err != nil {
		return nil, fmt.Errorf("failed to score CV: %w", err)
	}

	// Step 4: Evaluate project
	projectEval, err := s.EvaluateProject(projectContent, projectRagContext)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate project: %w", err)
	}

	// Step 5: Generate overall summary
	overallSummary := s.generateOverallSummary(cvEval, projectEval)

	return &cvweb.CVResponse{
		CVMatchRate:     cvEval.MatchRate,
		CVFeedback:      cvEval.Feedback,
		ProjectScore:    projectEval.Score,
		ProjectFeedback: projectEval.Feedback,
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

func (s *llmService) generateOverallSummary(cvEval *CVEvaluation, projectEval *ProjectEvaluation) string {
	if cvEval.MatchRate >= 0.8 && projectEval.Score >= 8.0 {
		return "Excellent candidate fit. Strong technical background with proven project delivery capabilities."
	} else if cvEval.MatchRate >= 0.7 && projectEval.Score >= 7.0 {
		return "Good candidate fit. Solid foundation with room for growth in specific areas."
	} else if cvEval.MatchRate >= 0.6 && projectEval.Score >= 6.0 {
		return "Moderate candidate fit. Shows potential but would benefit from additional training and experience."
	} else {
		return "Limited candidate fit. Significant gaps in required skills and experience."
	}
}

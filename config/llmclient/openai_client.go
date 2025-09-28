package llmclient

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/dikyayodihamzah/cv-evaluator/pkg/env"
	"github.com/sashabaranov/go-openai"
)

type OpenAIClient struct {
	client *openai.Client
	model  string
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

func NewOpenAIClient() *OpenAIClient {
	// Support both OpenAI and DeepSeek API keys
	apiKey := env.GetString("DEEPSEEK_API_KEY")
	if apiKey == "" {
		apiKey = env.GetString("OPENAI_API_KEY")
		if apiKey == "" {
			panic("Either DEEPSEEK_API_KEY or OPENAI_API_KEY is required")
		}
	}

	config := openai.DefaultConfig(apiKey)

	// Set base URL based on provider
	baseURL := env.GetString("LLM_BASE_URL")
	if baseURL == "" {
		if env.GetString("DEEPSEEK_API_KEY") != "" {
			baseURL = "https://api.deepseek.com"
		} else {
			baseURL = "https://api.openai.com/v1"
		}
	}
	config.BaseURL = baseURL

	// Set model based on provider
	model := env.GetString("LLM_MODEL")
	if model == "" {
		if env.GetString("DEEPSEEK_API_KEY") != "" {
			model = "deepseek-chat"
		} else {
			model = "gpt-4"
		}
	}

	return &OpenAIClient{
		client: openai.NewClientWithConfig(config),
		model:  model,
	}
}

func (c *OpenAIClient) ExtractCVInfo(cvContent string) (*CVInfo, error) {
	prompt := fmt.Sprintf(`
Please analyze the following CV content and extract structured information in JSON format.

CV Content:
%s

Extract the following information:
1. Skills (list of technical skills, programming languages, frameworks, tools)
2. Experience (number of years of professional experience)
3. Projects (list of notable projects mentioned)
4. Achievements (list of quantifiable achievements with metrics)

Return the result as a JSON object with the following structure:
{
  "skills": ["skill1", "skill2", ...],
  "experience_years": number,
  "projects": ["project1", "project2", ...],
  "achievements": ["achievement1", "achievement2", ...]
}

Only return the JSON object, no additional text.`, cvContent)

	resp, err := c.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: c.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are an expert HR assistant that extracts structured information from CVs. Always respond with valid JSON only.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature: 0.1,
			MaxTokens:   1000,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)

	// Remove markdown code blocks if present
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var cvInfo CVInfo
	if err := json.Unmarshal([]byte(content), &cvInfo); err != nil {
		return nil, fmt.Errorf("failed to parse CV info JSON: %w, content: %s", err, content)
	}

	return &cvInfo, nil
}

func (c *OpenAIClient) ScoreCV(cvInfo *CVInfo, jobDesc, ragContext string) (*CVEvaluation, error) {
	prompt := fmt.Sprintf(`
You are an expert recruiter evaluating a candidate's CV against a job description.

Context from Job Requirements Database:
%s

Job Description:
%s

Candidate Profile:
- Skills: %s
- Experience: %d years
- Projects: %s
- Achievements: %s

Please evaluate the candidate's fit for this role and provide:
1. A match rate between 0.0 and 1.0 (where 1.0 is perfect match)
2. Detailed feedback on strengths and areas for improvement

Focus on these evaluation criteria:
- Technical Skills Match (40%% weight): backend, databases, APIs, cloud, AI/LLM exposure
- Experience Level (30%% weight): years, project complexity, leadership
- Relevant Achievements (20%% weight): impact, scale, quantifiable results
- Cultural Fit (10%% weight): communication, learning attitude, collaboration

Return the result as a JSON object:
{
  "match_rate": 0.85,
  "feedback": "Detailed feedback about the candidate's fit..."
}

Only return the JSON object, no additional text.`,
		ragContext,
		jobDesc,
		strings.Join(cvInfo.Skills, ", "),
		cvInfo.Experience,
		strings.Join(cvInfo.Projects, ", "),
		strings.Join(cvInfo.Achievements, ", "))

	resp, err := c.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: c.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are an expert recruiter that evaluates candidates objectively. Always respond with valid JSON only.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature: 0.2,
			MaxTokens:   800,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API for CV scoring: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI for CV scoring")
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var evaluation CVEvaluation
	if err := json.Unmarshal([]byte(content), &evaluation); err != nil {
		return nil, fmt.Errorf("failed to parse CV evaluation JSON: %w, content: %s", err, content)
	}

	// Validate match rate is within bounds
	if evaluation.MatchRate > 1.0 {
		evaluation.MatchRate = 1.0
	}
	if evaluation.MatchRate < 0.0 {
		evaluation.MatchRate = 0.0
	}

	return &evaluation, nil
}

func (c *OpenAIClient) EvaluateProject(projectContent, ragContext string) (*ProjectEvaluation, error) {
	prompt := fmt.Sprintf(`
You are an expert technical reviewer evaluating a project submission.

Evaluation Criteria from Database:
%s

Project Report:
%s

Please evaluate this project based on the following criteria (1-5 scale each):
1. Correctness (25%% weight): meets requirements, prompt design, LLM chaining, RAG, error handling
2. Code Quality (20%% weight): clean, modular, testable, best practices
3. Resilience (20%% weight): handles failures, retries, monitoring, graceful degradation
4. Documentation (15%% weight): clear README, API docs, architecture decisions, trade-offs
5. Creativity/Bonus (20%% weight): additional features, performance optimization, security, deployment

Provide an overall score out of 10 and detailed feedback.

Return the result as a JSON object:
{
  "score": 8.5,
  "feedback": "Detailed feedback about the project's strengths and areas for improvement..."
}

Only return the JSON object, no additional text.`, ragContext, projectContent)

	resp, err := c.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: c.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are an expert technical reviewer that evaluates projects objectively based on defined criteria. Always respond with valid JSON only.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature: 0.2,
			MaxTokens:   1000,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API for project evaluation: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI for project evaluation")
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var evaluation ProjectEvaluation
	if err := json.Unmarshal([]byte(content), &evaluation); err != nil {
		return nil, fmt.Errorf("failed to parse project evaluation JSON: %w, content: %s", err, content)
	}

	// Validate score is within bounds
	if evaluation.Score > 10.0 {
		evaluation.Score = 10.0
	}
	if evaluation.Score < 0.0 {
		evaluation.Score = 0.0
	}

	return &evaluation, nil
}

func (c *OpenAIClient) GetEmbedding(text string) ([]float32, error) {
	// Check if we're using DeepSeek (which may not support embeddings)
	if env.GetString("DEEPSEEK_API_KEY") != "" {
		// For DeepSeek, create a simple hash-based embedding as fallback
		return c.createSimpleEmbedding(text), nil
	}

	embeddingModel := env.GetString("EMBEDDING_MODEL", "text-embedding-ada-002")

	resp, err := c.client.CreateEmbeddings(
		context.Background(),
		openai.EmbeddingRequest{
			Input: []string{text},
			Model: openai.EmbeddingModel(embeddingModel),
		},
	)

	if err != nil {
		// Fallback to simple embedding if OpenAI embeddings fail
		return c.createSimpleEmbedding(text), nil
	}

	if len(resp.Data) == 0 {
		return c.createSimpleEmbedding(text), nil
	}

	return resp.Data[0].Embedding, nil
}

// createSimpleEmbedding creates a basic embedding based on text characteristics
func (c *OpenAIClient) createSimpleEmbedding(text string) []float32 {
	// Create a 768-dimensional embedding (smaller than OpenAI's 1536)
	embedding := make([]float32, 768)

	// Simple word-based feature extraction
	words := strings.Fields(strings.ToLower(text))

	// Basic features
	for i, word := range words {
		if i >= 768 {
			break
		}
		// Simple hash-based value
		hash := 0
		for _, char := range word {
			hash = hash*31 + int(char)
		}
		if hash < 0 {
			hash = -hash
		}
		embedding[i] = float32(hash%1000) / 1000.0
	}

	// Normalize the embedding
	var magnitude float32
	for _, val := range embedding {
		magnitude += val * val
	}
	if magnitude > 0 {
		magnitude = 1.0 / float32(math.Sqrt(float64(magnitude)))
		for i := range embedding {
			embedding[i] *= magnitude
		}
	}

	return embedding
}

package cvweb

type CVResponse struct {
	CVMatchRate     float64 `json:"cv_match_rate"`
	CVFeedback      string  `json:"cv_feedback"`
	ProjectScore    float64 `json:"project_score"`
	ProjectFeedback string  `json:"project_feedback"`
	OverallSummary  string  `json:"overall_summary"`
}

type EvaluationRequest struct {
	CVFile             string `json:"cv_file" validate:"required"`
	ProjectFile        string `json:"project_file,omitempty"`
	JobDescription     string `json:"job_description,omitempty"`
	JobDescriptionFile string `json:"job_description_file,omitempty"`
}

type JobStatus struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

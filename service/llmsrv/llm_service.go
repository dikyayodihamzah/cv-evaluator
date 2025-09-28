package llmsrv

type LLMService interface {
}

type llmService struct {
}

func New() LLMService {
	return &llmService{}
}

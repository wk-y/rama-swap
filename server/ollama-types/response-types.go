package ollamatypes

type ChatResponse struct {
	Model     string  `json:"model"`
	CreatedAt string  `json:"created_at"`
	Message   Message `json:"message"`
	Done      bool    `json:"done"`
}

type ChatFinalResponse struct {
	ChatResponse
	TotalDuration      int `json:"total_duration"`
	LoadDuration       int `json:"load_duration"`
	PromptEvalCount    int `json:"prompt_eval_count"`
	PromptEvalDuration int `json:"prompt_eval_duration"`
	EvalCount          int `json:"eval_count"`
	EvalDuration       int `json:"eval_duration"`
}

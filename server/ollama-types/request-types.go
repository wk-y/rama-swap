package ollamatypes

type ChatRequest struct {
	Model    *string   `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	Options  *Options  `json:"options"`
}

type Options struct {
	NumCtx        *int64   `json:"num_ctx"`
	RepeatLastN   *int64   `json:"repeat_last_n"`
	RepeatPenalty *float64 `json:"repeat_penalty"`
	Temperature   *float64 `json:"temperature"`
	Seed          *int64   `json:"seed"`
	Stop          string   `json:"stop"`
	NumPredict    *int64   `json:"num_predict"`
	TopK          *int64   `json:"top_k"`
	TopP          *float64 `json:"top_p"`
	MinP          *float64 `json:"min_p"`
}

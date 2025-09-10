package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/openai/openai-go/v2"
	ollamatypes "github.com/wk-y/rama-swap/server/ollama-types"
)

func (s *Server) ollamaTags(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)

	ramaModels, err := s.ramalama.GetModels()
	if err != nil {
		log.Printf("Failed to get models: %v\n", ramaModels)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("E_MODEL_GET\n"))
	}

	var models struct {
		Models []ollamatypes.Model `json:"models"`
	}
	for _, ramaModel := range ramaModels {
		models.Models = append(models.Models, ollamatypes.Model{
			Name:       ramaModel.Name,
			Model:      ramaModel.Name,
			ModifiedAt: ramaModel.Modified,
			Size:       ramaModel.Size,
			Details: ollamatypes.ModelDetails{ // todo: fetch real data from ramalama inspect
				Format: "gguf",
			},
		})
	}

	w.Header().Add("Content-Type", "application/json; charset=utf-8")

	err = json.NewEncoder(w).Encode(models)
	if err != nil {
		log.Printf("Failed to reply: %v\n", err)
	}
}

func (s *Server) ollamaVersion(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)

	var version struct {
		Version string `json:"version"`
	}

	version.Version = "0.11.10" // arbitrary

	w.Header().Add("Content-Type", "application/json; charset=utf-8")

	err := json.NewEncoder(w).Encode(version)
	if err != nil {
		log.Printf("Failed to reply: %v\n", err)
	}
}

func (s *Server) ollamaChat(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)

	var requestJson ollamatypes.ChatRequest
	requestJson.Stream = true // default value

	rDecoder := json.NewDecoder(r.Body)
	err := rDecoder.Decode(&requestJson)
	if err != nil || requestJson.Model == nil {
		log.Println("Bad chat request:", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid request JSON\n"))
		return
	}

	model := *requestJson.Model

	backendModel, err := s.StartModel(model)
	if err != nil {
		log.Printf("Failed to start model %s: %v\n", backendModel, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("E_MODEL_START"))
	}

	<-backendModel.ready

	client := backendModel.newClient()
	params := ollamaTranslateParams(requestJson)

	completionStartTime := time.Now().UTC()
	stream := client.Chat.Completions.NewStreaming(r.Context(), params)
	defer stream.Close()

	responseEncoder := json.NewEncoder(w)
	responseController := http.NewResponseController(w)

	// only written to in non-streaming mode to send all tokens in the final response
	var accumulator strings.Builder

	// time the first token was generated
	var firstCreatedAt *time.Time

	// number of tokens generated
	var evalCount int64

	for stream.Next() {
		event := stream.Current()
		if len(event.Choices) > 0 {
			evalCount++

			if firstCreatedAt == nil {
				time := time.Unix(event.Created, 0)
				firstCreatedAt = &time
			}

			if requestJson.Stream == false {
				accumulator.WriteString(event.Choices[0].Delta.Content)
				continue
			}

			ollamaEvent := ollamatypes.ChatResponse{
				Model:     model,
				CreatedAt: time.Unix(event.Created, 0).Format(time.RFC3339),
				Message: ollamatypes.Message{
					Role:    "assistant",
					Content: event.Choices[0].Delta.Content,
				},
				Done: false,
			}

			err = responseEncoder.Encode(ollamaEvent)
			if err != nil {
				log.Printf("Failed to send delta: %v\n", err)
				return
			}

			// Flush to reduce stream latency. Whether it succeeds isn't important.
			_ = responseController.Flush()
		}
	}

	if err := stream.Err(); err != nil {
		log.Println("Error during response stream:", err)
		// keep going to send final response
	}

	completionFinishTime := time.Now().UTC()

	if firstCreatedAt == nil {
		firstCreatedAt = &completionFinishTime
	}

	responseEncoder.Encode(ollamatypes.ChatFinalResponse{
		ChatResponse: ollamatypes.ChatResponse{
			Model:     model,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Message: ollamatypes.Message{
				Role:    "assistant",
				Content: accumulator.String(),
			},
			Done: true,
		},
		TotalDuration: completionFinishTime.Sub(completionStartTime).Nanoseconds(),
		EvalDuration:  completionFinishTime.Sub(*firstCreatedAt).Nanoseconds(),
		EvalCount:     evalCount,
	})
}

func ollamaTranslateParams(request ollamatypes.ChatRequest) (completion openai.ChatCompletionNewParams) {
	completion.Model = *request.Model

	completion.Messages = make([]openai.ChatCompletionMessageParamUnion, len(request.Messages))
	for i, message := range request.Messages {
		switch message.Role {
		case "user":
			completion.Messages[i] = openai.UserMessage(message.Content)
		case "system":
			completion.Messages[i] = openai.SystemMessage(message.Content)
		default: // fallback to assistant message type
			completion.Messages[i] = openai.AssistantMessage(message.Content)
		}
	}

	if request.Options != nil {
		ollamaAddOptions(&completion, *request.Options)
	}

	return
}

// ollamaAddOptions sets completion's options to ollama options, where possible.
// Options that are not set will not be changed.
func ollamaAddOptions(completion *openai.ChatCompletionNewParams, options ollamatypes.Options) {
	if options.Temperature != nil {
		completion.Temperature = openai.Opt(*options.Temperature)
	}

	if options.Seed != nil {
		completion.Seed = openai.Opt(*options.Seed)
	}

	if options.NumPredict != nil {
		completion.MaxCompletionTokens = openai.Opt(*options.NumPredict)
	}

	if options.TopP != nil {
		completion.TopP = openai.Opt(*options.TopP)
	}
}

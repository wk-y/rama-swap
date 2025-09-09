package server

import (
	"encoding/json"
	"log"
	"net/http"
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
	messages := make([]openai.ChatCompletionMessageParamUnion, len(requestJson.Messages))
	for i, message := range requestJson.Messages {
		switch message.Role {
		case "user":
			messages[i] = openai.UserMessage(message.Content)
		case "system":
			messages[i] = openai.SystemMessage(message.Content)
		default: // fallback to assistant message type
			messages[i] = openai.AssistantMessage(message.Content)
		}
	}

	stream := client.Chat.Completions.NewStreaming(r.Context(), openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    model,
	})

	responseEncoder := json.NewEncoder(w)
	responseController := http.NewResponseController(w)
	defer stream.Close()
	for stream.Next() {
		event := stream.Current()
		if len(event.Choices) > 0 {
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

	responseEncoder.Encode(ollamatypes.ChatFinalResponse{
		ChatResponse: ollamatypes.ChatResponse{
			Model:     model,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Message: ollamatypes.Message{
				Role:    "assistant",
				Content: "",
			},
			Done: true,
			// todo: fill out all the other fields
		},
	})

	w.Header().Add("Content-Type", "application/json; charset=utf-8")
}

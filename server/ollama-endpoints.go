package server

import (
	"encoding/json"
	"log"
	"net/http"
)

type OllamaModel struct {
	Name       string             `json:"name"`
	Model      string             `json:"model"`
	ModifiedAt string             `json:"modified_at"` // timestamp
	Size       int                `json:"size"`
	Digest     string             `json:"digest"`
	Details    OllamaModelDetails `json:"details"`
}

type OllamaModelDetails struct {
	ParentModel       string   `json:"parent_model"`
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

func (s *Server) ollamaTags(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)

	ramaModels, err := s.ramalama.GetModels()
	if err != nil {
		log.Printf("Failed to get models: %v\n", ramaModels)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("E_MODEL_GET\n"))
	}

	var models struct {
		Models []OllamaModel `json:"models"`
	}
	for _, ramaModel := range ramaModels {
		models.Models = append(models.Models, OllamaModel{
			Name:       ramaModel.Name,
			Model:      ramaModel.Name,
			ModifiedAt: ramaModel.Modified,
			Size:       ramaModel.Size,
			Details: OllamaModelDetails{ // todo: fetch real data from ramalama inspect
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

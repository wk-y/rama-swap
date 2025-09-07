package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/wk-y/rama-swap/ramalama"
)

type Server struct {
	ramalama *ramalama.Ramalama
}

func (s *Server) HandleHttp(mux *http.ServeMux) {
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json; charset=utf-8")
		ramaModels, err := s.ramalama.GetModels()
		if err != nil {
			log.Printf("Failed to get models: %v\n", ramaModels)
		}
		models, err := convertModelList(ramaModels)
		if err != nil {
			log.Printf("Failed to convert models: %v\n", models)
		}
		err = json.NewEncoder(w).Encode(models)
		if err != nil {
			log.Printf("Failed to reply: %v\n", models)
		}
	})
}

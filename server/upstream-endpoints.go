package server

import (
	"fmt"
	"html"
	"net/http"
)

func (s *Server) serveUpstreamSelect(w http.ResponseWriter, r *http.Request) {
	models, err := s.ramalama.GetModels()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write([]byte("<html><body>"))
	w.Write([]byte("<p>Models:</p>"))
	// write the list of models
	w.Write([]byte("<ul>"))
	for _, model := range models {
		fmt.Fprintf(w, `<li><a href="/upstream/%s">%s</a></li>`, html.EscapeString(s.ModelNameMangler(model.Name)), html.EscapeString(model.Name))
	}
	w.Write([]byte("</ul>"))
	w.Write([]byte("</html></body>"))
}

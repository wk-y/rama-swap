package server

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"net/http/httputil"
)

func (s *Server) serveUpstream(w http.ResponseWriter, r *http.Request) {
	name, err := s.demangle(r.PathValue("model"))
	if err != nil {
		log.Printf("Demangling model name failed: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid model name"))
		return
	}

	backend, err := s.scheduler.Lock(r.Context(), name)
	if err != nil {
		log.Println(err)
		w.Write([]byte(fmt.Sprint(err)))
		return
	}
	defer s.scheduler.Unlock(backend)

	<-backend.ready
	proxy := httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			*pr.Out.URL = *pr.In.URL
			pr.Out.URL.Host = fmt.Sprintf("127.0.0.1:%v", backend.port)
			pr.Out.URL.Path = "/" + r.PathValue("rest")
			pr.Out.URL.Scheme = "http"
		},
	}
	proxy.ServeHTTP(w, r)
}

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

func (s *Server) demangle(name string) (string, error) {
	if s.ModelNameMangler == nil {
		return name, nil
	}

	s.demangleCacheLock.RLock()
	cached, ok := s.demangleCache[name]
	s.demangleCacheLock.RUnlock()

	if ok {
		return cached, nil
	}

	models, err := s.ramalama.GetModels()
	if err != nil {
		return "", fmt.Errorf("failed to get models: %v", err)
	}

	s.demangleCacheLock.Lock()
	defer s.demangleCacheLock.Unlock()

	for _, model := range models {
		s.demangleCache[s.ModelNameMangler(model.Name)] = model.Name
	}

	demangled, ok := s.demangleCache[name]
	if ok {
		return demangled, nil
	}

	return "", fmt.Errorf("model not found")
}

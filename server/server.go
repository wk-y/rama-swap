package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"sync"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/wk-y/rama-swap/internal/util"
	"github.com/wk-y/rama-swap/ramalama"
)

type Server struct {
	ModelNameMangler func(string) string
	BasePort         int // starting port number to use for underlying instances

	ramalama  ramalama.Ramalama
	scheduler ModelScheduler

	demangleCacheLock sync.RWMutex
	demangleCache     map[string]string
}

func NewServer(r ramalama.Ramalama) *Server {
	return &Server{
		ramalama:      r,
		scheduler:     NewFcfsScheduler(r, 49170),
		demangleCache: map[string]string{},
	}
}

type backend struct {
	sync.RWMutex
	ready  chan struct{}
	exited chan struct{}
	// The port the backend is currently running on.
	// A value of zero indicates no port is currently assigned.
	port     int
	portLock sync.RWMutex
	err      error
	cancel   func()
}

func (s *Server) HandleHttp(mux *http.ServeMux) {
	// OpenAI-compatible endpoints
	mux.HandleFunc("/v1/models", s.handleModels)
	mux.HandleFunc("POST /v1/chat/completions", s.handleChatCompletions)
	mux.HandleFunc("POST /v1/completions", s.handleCompletions)

	// Ollama-compatible endpoints
	mux.HandleFunc("/api/version", s.ollamaVersion)
	mux.HandleFunc("/api/tags", s.ollamaTags)
	mux.HandleFunc("/api/chat", s.ollamaChat)

	// llama-swap style endpoint
	mux.HandleFunc("/upstream/{model}/{rest...}", s.serveUpstream)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Unhandled endpoint ", r.URL)
		w.WriteHeader(http.StatusNotFound)
	})
}

func (s *Server) proxyEndpoint(w http.ResponseWriter, r *http.Request, modelFinder func(body io.Reader) (model string, err error)) {
	var decoderRead bytes.Buffer
	tee := io.TeeReader(r.Body, &decoderRead)

	model, err := modelFinder(tee)

	if err != nil {
		log.Println("Failed to determine model for request:", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing or invalid 'model' key"))
		return
	}

	backend, err := s.scheduler.Lock(r.Context(), model)
	if err != nil {
		log.Printf("Failed to start model %s: %v\n", model, err)
		return
	}
	defer s.scheduler.Unlock(backend)

	proxy := httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			*pr.Out.URL = *pr.In.URL
			pr.Out.URL.Host = fmt.Sprintf("127.0.0.1:%v", backend.port)
			pr.Out.URL.Scheme = "http"
			pr.Out.Body = util.ReadCloserWrapper{
				Reader: io.MultiReader(&decoderRead, pr.In.Body),
				Closer: pr.In.Body.Close,
			}
		},
	}
	proxy.ServeHTTP(w, r)
}

func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	s.proxyEndpoint(w, r, func(body io.Reader) (model string, err error) {
		var modelGet struct {
			Model *string
		}

		err = json.NewDecoder(body).Decode(&modelGet)
		if err != nil {
			return
		}

		if modelGet.Model == nil {
			return "", fmt.Errorf("missing model key")
		}

		return *modelGet.Model, nil
	})
}

func (s *Server) handleCompletions(w http.ResponseWriter, r *http.Request) {
	s.proxyEndpoint(w, r, func(body io.Reader) (model string, err error) {
		var modelGet struct {
			Model *string
		}

		err = json.NewDecoder(body).Decode(&modelGet)
		if err != nil {
			return
		}

		if modelGet.Model == nil {
			return "", fmt.Errorf("missing model key")
		}

		return *modelGet.Model, nil
	})
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	internalServerError := func(reason string) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error: "))
		w.Write([]byte(reason))
	}

	ramaModels, err := s.ramalama.GetModels()
	if err != nil {
		log.Printf("Failed to get models: %v\n", ramaModels)
		internalServerError("E_MODEL_GET")
		return
	}

	models, err := convertModelList(ramaModels)
	if err != nil {
		log.Printf("Failed to convert models: %v\n", models)
		internalServerError("E_MODEL_LIST_CONVERT")
		return
	}

	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	err = json.NewEncoder(w).Encode(models)

	if err != nil {
		log.Printf("Failed to reply: %v\n", err)
	}
}

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

func (b *backend) healthCheck() bool {
	// /health is more accurate but might be llama-server specific
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%v/health", b.port))
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// WithClient runs callback with a client configured to use the backend.
// Because the backend's port may be freed and reused by another backend,
// it is not safe to save the client given to callback.
func (b *backend) WithClient(callback func(openai.Client) error) error {
	b.portLock.RLock()
	defer b.portLock.RUnlock()

	if b.port == 0 { // port was freed
		return errors.New("backend is dead")
	}

	client := openai.NewClient(
		option.WithAPIKey(""),
		option.WithOrganization(""),
		option.WithProject(""),
		option.WithWebhookSecret(""),
		option.WithBaseURL(fmt.Sprintf("http://127.0.0.1:%v", b.port)),
	)

	return callback(client)
}

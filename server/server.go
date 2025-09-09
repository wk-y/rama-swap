package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/wk-y/rama-swap/internal/util"
	"github.com/wk-y/rama-swap/ramalama"
)

type Server struct {
	ModelNameMangler func(string) string
	BasePort         int // starting port number to use for underlying instances

	ramalama          ramalama.Ramalama
	backendsLock      sync.RWMutex
	backends          map[string]*backend
	demangleCacheLock sync.RWMutex
	demangleCache     map[string]string
	portManager       portManager
}

func NewServer(r ramalama.Ramalama) *Server {
	return &Server{
		ramalama:      r,
		backends:      map[string]*backend{},
		demangleCache: map[string]string{},
		portManager:   *newPortManager(49170),
	}
}

type backend struct {
	sync.RWMutex
	ready  chan struct{}
	exited bool
	port   int
	err    error
	cancel func()
}

func (s *Server) HandleHttp(mux *http.ServeMux) {
	// OpenAI-compatible endpoints
	mux.HandleFunc("/v1/models", s.handleModels)
	mux.HandleFunc("POST /v1/chat/completions", s.handleChatCompletions)
	mux.HandleFunc("POST /v1/completions", s.handleCompletions)

	// Ollama-compatible endpoints
	mux.HandleFunc("/api/version", s.ollamaVersion)
	mux.HandleFunc("/api/tags", s.ollamaTags)

	// llama-swap style endpoint
	mux.HandleFunc("/upstream/{model}/{rest...}", s.serveUpstream)
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

	backend, err := s.StartModel(model)
	if err != nil {
		log.Printf("Failed to start model %s: %v\n", model, err)
		return
	}
	<-backend.ready

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
	backend, err := s.StartModel(name)

	if err != nil {
		log.Println(err)
		w.Write([]byte(fmt.Sprint(err)))
		return
	}

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

func (s *Server) StartModel(name string) (*backend, error) {
	s.backendsLock.Lock()
	defer s.backendsLock.Unlock()

	back, ok := s.backends[name]
	if ok && !back.exited {
		return back, nil
	}

	// stop all other backends
	for k, backend := range s.backends {
		backend.cancel()
		delete(s.backends, k)
	}

	back = &backend{}
	back.port = s.portManager.ReservePort()

	ctx, cancel := context.WithCancel(context.Background())
	cmd := s.ramalama.ServeCommand(ctx, ramalama.ServeArgs{
		Model: name,
		Port:  back.port,
	})
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start ramalama: %v\n", err)
	}

	switch runtime.GOOS {
	case "linux":
		// By default, Go sends SIGKILL, which causes ramalama to exit without stopping the container.
		// Instead, let ramalama gracefully exit by sending SIGINT
		cmd.Cancel = func() error {
			return cmd.Process.Signal(os.Interrupt)
		}
	default:
		log.Println("[WARN] Graceful shutdown of ramalama not supported for OS, switching may not work correctly")
	}

	back.cancel = cancel
	back.ready = make(chan struct{})

	// waits for ready
	go func() {
		defer close(back.ready)

		for !back.healthCheck() {
			back.Lock()
			dead := back.exited
			back.Unlock()

			if dead {
				break
			}

			time.Sleep(time.Second) // fixme
		}
	}()

	// waits for exit
	go func() {
		err := cmd.Wait()
		back.cancel()

		back.Lock()
		back.err = err
		back.exited = true
		s.portManager.ReleasePort(back.port)
		back.Unlock()
	}()

	s.backends[name] = back
	return back, nil
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

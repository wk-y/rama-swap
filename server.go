package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"

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
		portManager:   *newPortManager(20050),
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

	mux.HandleFunc("/upstream/{model}/{rest...}", s.serveUpstream)
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

	back = &backend{}
	back.exited = false
	back.err = nil
	back.port = s.portManager.ReservePort()

	ctx, cancel := context.WithCancel(context.Background())
	cmd := s.ramalama.ServeCommand(ctx, ramalama.ServeArgs{
		Model: name,
		Port:  back.port,
	})

	err := cmd.Start()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start ramalama: %v\n", err)
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

			time.Sleep(1000) // fixme
		}
	}()

	// waits for exit
	go func() {
		err := cmd.Wait()

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
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%v/v1/models", b.port))
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}

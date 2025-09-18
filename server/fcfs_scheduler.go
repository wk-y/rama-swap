package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/wk-y/rama-swap/ramalama"
)

// fcfsScheduler is a ModelScheduler that implements (roughly) first-come-first-serve
// access with at most one model loaded at a time.
type fcfsScheduler struct {
	port         int // port to attach the backend to
	lock         sync.Mutex
	ramalama     ramalama.Ramalama
	backend      *backend
	backendModel string
	idle         sync.Cond
	backendUsers sync.WaitGroup

	// cached set of valid model names
	ramalamaModelsCache     map[string]struct{}
	ramalamaModelsCacheLock sync.Mutex
}

// Lock implements ModelScheduler.
func (f *fcfsScheduler) Lock(ctx context.Context, model string) (*backend, error) {
	exists, err := f.modelExists(model)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New("nonexistent model")
	}

	f.lock.Lock()
	defer f.lock.Unlock()

	select {
	case <-ctx.Done():
		return nil, errors.New("context cancelled")
	default:
	}

	if f.backend != nil && f.backendModel == model {
		f.backendUsers.Add(1)
		return f.backend, nil
	}

	f.backendUsers.Wait()

	if f.backend != nil {
		f.backend.cancel()
		<-f.backend.exited
		f.backend = nil
	}

	backend, err := f.startBackend(model)
	if err != nil {
		return nil, err
	}
	f.backend = backend
	f.backendModel = model

	select {
	case <-ctx.Done():
		return nil, errors.New("context cancelled")
	case <-backend.ready:
		f.backendUsers.Add(1)
		return f.backend, nil
	}
}

// Unlock implements ModelScheduler.
func (f *fcfsScheduler) Unlock(backend *backend) {
	if f.backend == backend {
		f.backendUsers.Done()
	}
}

func (f *fcfsScheduler) modelExists(modelName string) (bool, error) {
	f.ramalamaModelsCacheLock.Lock()
	defer f.ramalamaModelsCacheLock.Unlock()

	_, ok := f.ramalamaModelsCache[modelName]
	if ok {
		return true, nil
	}

	models, err := f.ramalama.GetModels()
	if err != nil {
		return false, err
	}

	f.ramalamaModelsCache = make(map[string]struct{}, len(models))
	for _, model := range models {
		f.ramalamaModelsCache[model.Name] = struct{}{}
	}

	_, ok = f.ramalamaModelsCache[modelName]
	return ok, nil
}

func (f *fcfsScheduler) startBackend(modelName string) (*backend, error) {
	back := &backend{}
	back.port = f.port

	ctx, cancel := context.WithCancel(context.Background())
	back.cancel = cancel

	cmd := f.ramalama.ServeCommand(ctx, ramalama.ServeArgs{
		Model: modelName,
		Port:  back.port,
	})
	cmd.Stderr = os.Stderr

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

	err := cmd.Start()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start ramalama: %v\n", err)
	}

	back.ready = make(chan struct{})
	back.exited = make(chan struct{})

	// waits for ready
	go func() {
		defer close(back.ready)

		for !back.healthCheck() {
			select {
			case <-back.exited:
				return
			default:
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

		back.portLock.Lock()
		back.port = 0
		back.portLock.Unlock()

		close(back.exited) // must be after portLock unlock

		back.Unlock()
	}()

	return back, nil
}

func NewFcfsScheduler(ramalama ramalama.Ramalama, port int) *fcfsScheduler {
	return &fcfsScheduler{
		ramalama:            ramalama,
		port:                port,
		ramalamaModelsCache: map[string]struct{}{},
	}
}

var _ ModelScheduler = (*fcfsScheduler)(nil)

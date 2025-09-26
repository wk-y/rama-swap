package scheduler

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"sync"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

type backend struct {
	sync.RWMutex
	Ready    chan struct{}
	Exited   chan struct{}
	port     int
	portLock sync.RWMutex
	err      error
	cancel   func()
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

func (b *backend) Proxy() *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			*pr.Out.URL = *pr.In.URL
			pr.Out.URL.Host = fmt.Sprintf("127.0.0.1:%v", b.port)
			pr.Out.URL.Scheme = "http"
		},
	}
}

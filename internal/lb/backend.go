package lb

import (
	"net/http/httputil"
	"net/url"
	"sync"
)

type Backend struct {
	URL   *url.URL
	Proxy *httputil.ReverseProxy
	Alive bool
	mu    sync.RWMutex
}

func (b *Backend) SetAlive(alive bool) {
	b.mu.Lock()
	b.Alive = alive
	b.mu.Unlock()
}

func (b *Backend) isAlive() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Alive
}

func NewBackends() []*Backend {
	rawURLs := []string{
		"http://localhost:9001",
		"http://localhost:9002",
		"http://localhost:9003",
	}
	var backends []*Backend

	for _, raw := range rawURLs {
		u, _ := url.Parse(raw)

		backends = append(backends, &Backend{
			URL:   u,
			Proxy: httputil.NewSingleHostReverseProxy(u),
		})
	}

	return backends
}

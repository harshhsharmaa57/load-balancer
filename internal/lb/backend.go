package lb

import (
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
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

// NewBackends creates the backend pool from BACKEND_URLS env var (comma-separated)
// or falls back to default localhost URLs for local development.
func NewBackends() []*Backend {
	rawURLs := []string{
		"http://localhost:9001",
		"http://localhost:9002",
		"http://localhost:9003",
	}

	// Allow override via environment variable (e.g. Docker Compose)
	if env := os.Getenv("BACKEND_URLS"); env != "" {
		rawURLs = strings.Split(env, ",")
	}

	var backends []*Backend

	for _, raw := range rawURLs {
		raw = strings.TrimSpace(raw)
		u, _ := url.Parse(raw)

		backends = append(backends, &Backend{
			URL:   u,
			Proxy: httputil.NewSingleHostReverseProxy(u),
		})
	}

	return backends
}

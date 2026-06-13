package main

import (
	"net/http/httputil"
	"net/url"
	"sync"
)

type Backend struct {
	URL   *url.URL
	Proxy *httputil.ReverseProxy
	Alive bool
	mu    sync.RWMutex // protects Alive
}

func (b *Backend) SetAlive(alive bool) {
	b.mu.Lock() // only 1 can write
	b.Alive = alive
	b.mu.Unlock()
}

func (b *Backend) isAlive() bool {
	b.mu.RLock() // shared read (many can read at once)
	defer b.mu.RUnlock()
	return b.Alive
}

func reverseProxy() []*Backend {
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

	// fmt.Println(len(backends))
	// fmt.Println(backends[0].URL)
	// fmt.Println(backends[0].URL.Host)

	// for i := 0; i <= 10; i++ {

	// 	fmt.Println(nextBackend(backends).URL.Host)
	// }

}

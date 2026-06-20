package lb

import (
	"sync/atomic"
	"time"
)

var counter uint64

func NextBackend(backends []*Backend) *Backend {
	n := uint64(len(backends))
	for i := uint64(0); i < n; i++ {
		next := atomic.AddUint64(&counter, 1)
		b := backends[next%n]

		if b.isAlive() {
			return b
		}
		n = n - uint64(1)
	}
	return nil
}

func StartHealthCheck(backends []*Backend) {
	ticker := time.NewTicker(5 * time.Second)

	go func() {
		for range ticker.C {
			for _, b := range backends {
				alive := isBackendAlive(b.URL)
				b.SetAlive(alive)
			}
		}
	}()
}

package main

import "sync/atomic"

var counter uint64

func nextBackend(backends []*Backend) *Backend {
	// atomic.AddUint64 does: counter++ but thread-safe
	n := uint64(len(backends))
	for i := uint64(0); i < n; i++ {
		next := atomic.AddUint64(&counter, 1)
		b := backends[next%n]

		if b.isAlive() {
			return b
		} else {
			n = n - uint64(1)
		}
	}
	return nil
}

package main

import "sync/atomic"

var counter uint64

func nextBackend(backends []*Backend) *Backend {
	// atomic.AddUint64 does: counter++ but thread-safe
	next := atomic.AddUint64(&counter, 1)-1
	idx := next % uint64(len(backends))

	return backends[idx]
}

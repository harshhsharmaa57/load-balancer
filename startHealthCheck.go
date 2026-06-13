package main

import (
	"log"
	"time"
)

func startHealthCheck(backends []*Backend) {
	ticker := time.NewTicker(5 * time.Second)

	go func() {
		for range ticker.C {
			for _, b := range backends {
				alive := isBackendAlive(b.URL)
				b.SetAlive(alive)
				log.Printf("%s alive=%v", b.URL.Host, alive)
			}
		}
	}()

}

package main

import (
	"log"
	"net/http"
	"time"

	"github.com/harshhsharmaa57/load-balancer/internal/lb"
)

func main() {
	backends := lb.NewBackends()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		b := lb.NextBackend(backends)

		if b == nil {
			http.Error(w, "no healthy backends", http.StatusServiceUnavailable)
			return
		}

		b.Proxy.ServeHTTP(w, r)

		log.Printf("[%s] %s %s  →  %s  (%v)",
			time.Now().Format("15:04:05"),
			r.Method,
			r.URL.Path,
			b.URL.Host,
			time.Since(start),
		)
	})

	lb.StartHealthCheck(backends)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      http.DefaultServeMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("LB starting on :8080")
	log.Fatal(srv.ListenAndServe())
}

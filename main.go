package main

import (
	"log"
	"net/http"
	"time"
)

// type Backend struct {
// 	URL   string
// 	Alive bool
// }

// func (u *Backend) IsAlive() bool {
// 	return u.Alive
// }

// func (b *Backend) SetAlive(v bool) {
// 	b.Alive = v
// }

// func checkBackend(name string) {
// 	time.Sleep(1 * time.Second) // simulate a network ping
// 	fmt.Println(name, "is up")
// }

func main() {
	// server1 := &Backend{URL: "http://localhost:9001", Alive: true}
	// server2 := &Backend{URL: "http://localhost:9001", Alive: false}
	// server3 := &Backend{URL: "http://localhost:9001", Alive: true}

	// fmt.Println(server1.IsAlive())
	// fmt.Println(server2.IsAlive())
	// fmt.Println(server3.IsAlive())

	// go checkBackend("backend-1") // starts in background
	// go checkBackend("backend-2") // starts in background
	// go checkBackend("backend-3") // starts in background

	// time.Sleep(2 * time.Second) // wait for all to finish
	// fmt.Println("all checked")
	// fmt.Println(time.Second)
	backends := reverseProxy()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		b := nextBackend(backends)

		if b == nil {
			http.Error(w, "no healthy backends", 503)
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
	startHealthCheck(backends)
	srv := &http.Server{
		Addr:    ":8080",
		Handler: http.DefaultServeMux,

		ReadTimeout:  5 * time.Second,  // time to read the request
		WriteTimeout: 10 * time.Second, // time to write the response
		IdleTimeout:  60 * time.Second, // keep-alive connection idle
	}

	log.Println("LB starting on :8080")
	log.Fatal(srv.ListenAndServe())

}

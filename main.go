package main

import (
	"fmt"
	"log"
	"net/http"
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
		b := nextBackend(backends)
		if b == nil {
			http.Error(w, "No Healthy connection", 502)
			return
		}

		log.Printf("routing → %s", b.URL.Host)

		b.Proxy.ServeHTTP(w, r)
	})
	startHealthCheck(backends)
	fmt.Println("Server running at port http://localhost:8080/")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

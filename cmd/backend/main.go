package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Args[1]

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Response from backend %s", port)
	})

	log.Printf("Backend is running at port %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

package main

import (
	"log"
	"net/http"
)

func main() {
	// Serve static files (HTML, CSS, JS for the dashboard)
	http.Handle("/", http.FileServer(http.Dir("./dashboard")))

	// Start the dashboard server
	log.Println("Dashboard running at http://localhost:8081")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatalf("Failed to start dashboard: %v", err)
	}
}

package main

import (
	"log"
	"net/http"
	"os"

	"realtime-monitor/internal/monitor"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := monitor.New(".")
	log.Printf("monitor server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, server.Handler()); err != nil {
		log.Fatal(err)
	}
}

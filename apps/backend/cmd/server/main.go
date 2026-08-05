package main

import (
	"log"
	"net/http"
	"os"

	"github.com/hey-amanthakur/charrade/apps/backend/internal/server"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("charrade server listening on %s", addr)
	if err := http.ListenAndServe(addr, server.New()); err != nil {
		log.Fatal(err)
	}
}

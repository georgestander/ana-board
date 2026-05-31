package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/georgestander/ana-board/internal/server"
)

func main() {
	addr := os.Getenv("ANA_BOARD_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}

	app, err := server.NewServer()
	if err != nil {
		log.Fatalf("create server: %v", err)
	}

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           app.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("ana-board listening on http://%s", addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}

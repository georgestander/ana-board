package main

import (
	"log"
	"net/http"
	"os"
	"strings"
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

	for _, extraAddr := range extraAddrs(os.Getenv("ANA_BOARD_EXTRA_ADDRS")) {
		go func(extraAddr string) {
			extraServer := &http.Server{
				Addr:              extraAddr,
				Handler:           app.Routes(),
				ReadHeaderTimeout: 5 * time.Second,
			}

			log.Printf("ana-board listening on http://%s", extraAddr)
			if err := extraServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("serve %s: %v", extraAddr, err)
			}
		}(extraAddr)
	}

	log.Printf("ana-board listening on http://%s", addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}

func extraAddrs(raw string) []string {
	parts := strings.Split(raw, ",")
	addrs := make([]string, 0, len(parts))
	seen := map[string]bool{}

	for _, part := range parts {
		addr := strings.TrimSpace(part)
		if addr == "" || seen[addr] {
			continue
		}

		seen[addr] = true
		addrs = append(addrs, addr)
	}

	return addrs
}

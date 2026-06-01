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

	additionalAddrs := extraAddrs(os.Getenv("ANA_BOARD_EXTRA_ADDRS"))
	trustedOrigins := server.TrustedOriginsForAddrs(append([]string{addr}, additionalAddrs...)...)
	configuredOrigins, err := server.ParseTrustedOrigins(os.Getenv("ANA_BOARD_TRUSTED_ORIGINS"))
	if err != nil {
		log.Fatalf("trusted origins: %v", err)
	}
	trustedOrigins = append(trustedOrigins, configuredOrigins...)

	app, err := server.NewServer(server.WithTrustedOrigins(trustedOrigins))
	if err != nil {
		log.Fatalf("create server: %v", err)
	}

	httpServer := newHTTPServer(addr, app.Routes())

	for _, extraAddr := range additionalAddrs {
		go func(extraAddr string) {
			extraServer := newHTTPServer(extraAddr, app.Routes())

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

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
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

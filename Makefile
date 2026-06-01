.PHONY: build test vet check clean

GOCACHE ?= $(CURDIR)/.gocache
GOMODCACHE ?= $(CURDIR)/.gomodcache

build:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -o bin/ana-board ./cmd/ana-board
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -o bin/ana-boardctl ./cmd/ana-boardctl
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -o bin/ana-board-mcp ./cmd/ana-board-mcp
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -o bin/ana-board-codex-bridge ./cmd/ana-board-codex-bridge

test:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go test ./...

vet:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go vet ./...

check: test vet
	node --check web/static/board.js
	node --check web/static/admin.js

clean:
	rm -rf bin .gocache .gomodcache

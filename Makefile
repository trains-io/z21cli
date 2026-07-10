.PHONY: help build test clean

.DEFAULT_GOAL := help

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS = -X github.com/trains-io/z21cli/cmd.Version=$(VERSION) \
          -X github.com/trains-io/z21cli/cmd.Commit=$(COMMIT) \
          -X github.com/trains-io/z21cli/cmd.BuildDate=$(DATE)

help: ## Show targets
	@grep -E '^[a-zA-Z0-9_-]+:.*##' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*## "}; {printf "  %-18s %s\n", $$1, $$2}'

build: ## Build bin/z21
	go build -ldflags "$(LDFLAGS)" -o bin/z21 .

test: ## Run unit tests
	go test ./... -count=1

clean: ## Remove build artifacts
	rm -rf bin/

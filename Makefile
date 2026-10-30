.DEFAULT_GOAL := help

.PHONY: help test lint fuzz build test-hardware test-reconnect

help: ## List make targets
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: ## Run go vet and unit tests with the race detector
	go vet ./...
	go test -race ./...

lint: ## Check gofmt -s and run golangci-lint (see .golangci.yml)
	@unformatted="$$(gofmt -s -l .)"; \
	if [ -n "$$unformatted" ]; then echo "needs gofmt -s -w:"; echo "$$unformatted"; exit 1; fi
	golangci-lint run ./...
	golangci-lint run --build-tags hardware ./...

FUZZTIME ?= 15s

fuzz: ## Run each fuzz target for FUZZTIME (default 15s)
	@for f in $$(go test -list '^Fuzz' . | grep '^Fuzz'); do \
		go test -run '^$$' -fuzz "^$$f\$$" -fuzztime $(FUZZTIME) . || exit 1; \
	done

build: ## Compile the package
	go build ./...

test-hardware: ## Run hardware tests (needs a Nano and a Square attached)
	go test -tags hardware -race -count=1 -v -run '^TestHardware' ./...

test-reconnect: ## Hardware reconnect test (you unplug and replug the Nano when told)
	BLINKSTICK_UNPLUG=1 go test -tags hardware -count=1 -v -run '^TestHardwareReconnect$$' ./...

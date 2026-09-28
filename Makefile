.DEFAULT_GOAL := help

.PHONY: help test build test-hardware

help: ## List make targets
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: ## Run go vet and unit tests with the race detector
	go vet ./...
	go test -race ./...

build: ## Compile the package
	go build ./...

test-hardware: ## Run hardware tests (needs a Nano and a Square attached)
	go test -tags hardware -race -count=1 -v -run '^TestHardware' ./...

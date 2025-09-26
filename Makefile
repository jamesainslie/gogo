.PHONY: build test lint run clean install deps help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=gogo
BINARY_PATH=./cmd/gogo
MAIN_PATH=./cmd/gogo/main.go

# Build flags
LDFLAGS=-ldflags "-X main.version=$(shell git describe --tags --always --dirty 2>/dev/null || echo 'dev')"

# Default target
all: deps build

check: test lint build ## Run all checks: test, lint, and build

build: ## Build the binary
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)

test: ## Run tests with coverage
	$(GOTEST) -v ./... -coverprofile=coverage.out -covermode=atomic

lint: ## Run linter
	golangci-lint run

run: ## Run the application
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH) && ./$(BINARY_NAME)

clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out

deps: ## Install dependencies
	$(GOMOD) download
	$(GOMOD) tidy

install: build ## Install the binary
	mv $(BINARY_NAME) $(GOPATH)/bin/

## Display help
help:
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

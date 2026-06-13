.PHONY: help check-go-version build run dev test test-race coverage ci lint vet fmt tidy docker docker-run clean

APP_NAME = $(shell head -1 go.mod | cut -d "/" -f2-)
BUILD ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

BINARY ?= bin/tyvm
GO ?= go
GO_MIN_MAJOR ?= 1
GO_MIN_MINOR ?= 22
GO_MIN_PATCH ?= 0

PORT ?= 8080
DB_PATH ?= tyvm.db
DOCKER_IMAGE ?= tyvm
DOCKER_TAG ?= latest

.DEFAULT_GOAL := help

help:  ## Show this help
	@echo "$(APP_NAME):$(BUILD)"
	@perl -nle'print $& if m{^[a-zA-Z_-]+:.*?## .*$$}' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

check-go-version:  ## Verify installed Go meets minimum version
	@current="$$( $(GO) env GOVERSION 2>/dev/null | sed 's/^go//' )"; \
	current_major="$${current%%.*}"; \
	current_minor="$${current#*.}"; \
	current_minor="$${current_minor%%.*}"; \
	current_patch="$${current#*.*.}"; \
	if [ "$$current_patch" = "$$current" ]; then current_patch=0; fi; \
	current_patch="$${current_patch%%.*}"; \
	if [ -z "$$current_major" ] || [ -z "$$current_minor" ] || [ -z "$$current_patch" ]; then \
		echo "Unable to detect Go version via '$(GO) env GOVERSION'."; \
		exit 1; \
	fi; \
	if [ "$$current_major" -lt "$(GO_MIN_MAJOR)" ] || { [ "$$current_major" -eq "$(GO_MIN_MAJOR)" ] && { [ "$$current_minor" -lt "$(GO_MIN_MINOR)" ] || { [ "$$current_minor" -eq "$(GO_MIN_MINOR)" ] && [ "$$current_patch" -lt "$(GO_MIN_PATCH)" ]; }; }; }; then \
		echo "Go $$current detected. tyvm requires Go $(GO_MIN_MAJOR).$(GO_MIN_MINOR).$(GO_MIN_PATCH)+."; \
		echo "Upgrade Go and retry."; \
		exit 1; \
	fi

build: check-go-version  ## Build ./bin/tyvm
	@mkdir -p $(dir $(BINARY))
	$(GO) build -o $(BINARY) .

run: build  ## Build and run locally (PORT, DB_PATH overridable)
	PORT=$(PORT) DB_PATH=$(DB_PATH) ./$(BINARY)

dev: check-go-version  ## Run via 'go run .' (no build artifact)
	PORT=$(PORT) DB_PATH=$(DB_PATH) $(GO) run .

test: check-go-version  ## Run unit tests
	$(GO) test ./...

test-race: check-go-version  ## Run tests with race detector + coverage.out
	$(GO) test -v -race -coverprofile=coverage.out ./...

coverage:  ## Print coverage summary (requires coverage.out)
	$(GO) tool cover -func=coverage.out

ci: check-go-version  ## Build + vet + race tests + coverage
	$(GO) build ./... && $(GO) vet ./... && $(GO) test -v -race -coverprofile=coverage.out ./...

lint:  ## Run golangci-lint if installed
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed; skipping"; \
	fi

vet: check-go-version  ## Run go vet
	$(GO) vet ./...

fmt: check-go-version  ## Format all Go packages
	$(GO) fmt ./...

tidy: check-go-version  ## Run go mod tidy
	$(GO) mod tidy

docker:  ## Build docker image $(DOCKER_IMAGE):$(DOCKER_TAG)
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: docker  ## Run docker image with ./data mounted at /data
	@mkdir -p data
	docker run --rm -p $(PORT):8080 -v $(PWD)/data:/data $(DOCKER_IMAGE):$(DOCKER_TAG)

clean:  ## Remove build artifacts
	rm -rf bin coverage.out

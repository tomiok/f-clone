.DEFAULT_GOAL := help

BIN_DIR  := bin
APP_MAIN := ./cmd/app/main

GOLANGCI_LINT_VERSION := latest

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

## --- Build ---

.PHONY: build
build: ## Build the binary into ./bin
	go build -o $(BIN_DIR)/app $(APP_MAIN)

.PHONY: run
run: ## Run the app
	go run $(APP_MAIN)

.PHONY: docker
docker: ## Build the container image f-clone
	docker build -t f-clone .

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)

## --- Quality ---

.PHONY: test
test: ## Run tests
	go test ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run ./...

.PHONY: tidy
tidy: ## Tidy go.mod / go.sum
	go mod tidy

.PHONY: check
check: vet lint test ## Run vet, lint and tests

## --- Tooling ---

.PHONY: install-tools
install-tools: ## Install golangci-lint
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

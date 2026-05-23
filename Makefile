BINARY := lane
BIN_DIR := bin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: help
help: ## Show available targets
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build the binary to ./bin/lane
	@mkdir -p $(BIN_DIR)
	@go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY) ./cmd/lane

.PHONY: install
install: ## Install to GOPATH/bin
	@go install $(LDFLAGS) ./cmd/lane

.PHONY: test
test: ## Run all tests with race detector
	@go test ./... -race -count=1

.PHONY: test-cover
test-cover: ## Run tests and open HTML coverage report
	@go test ./... -race -count=1 -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html

.PHONY: lint
lint: ## Run golangci-lint
	@golangci-lint run ./...

.PHONY: fmt
fmt: ## Format code and run go vet
	@gofmt -w .
	@go vet ./...

.PHONY: tidy
tidy: ## Tidy go.mod
	@go mod tidy

.PHONY: clean
clean: ## Remove build artifacts
	@rm -rf $(BIN_DIR) coverage.out coverage.html

.PHONY: run
run: ## Build and run (usage: make run ARGS="list")
	@go run ./cmd/lane $(ARGS)

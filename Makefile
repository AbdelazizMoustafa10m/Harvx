# ─── Harvx Makefile ──────────────────────────────────────────────────────────
# Primary developer interface for building, testing, and linting Harvx.
# Use: make help | make all | make build | make run ARGS='generate .'

# ─── Build Metadata ──────────────────────────────────────────────────────────
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE       ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GO_VERSION ?= $(shell go version)

# ─── Paths ───────────────────────────────────────────────────────────────────
BINARY     := harvx
BIN_DIR    := bin
DIST_DIR   := dist
CMD_DIR    := ./cmd/harvx

# ─── Linker Flags ────────────────────────────────────────────────────────────
LDFLAGS_PKG := github.com/harvx/harvx/internal/buildinfo

LDFLAGS := -s -w \
	-X '$(LDFLAGS_PKG).Version=$(VERSION)' \
	-X '$(LDFLAGS_PKG).Commit=$(COMMIT)' \
	-X '$(LDFLAGS_PKG).Date=$(DATE)' \
	-X '$(LDFLAGS_PKG).GoVersion=$(GO_VERSION)'

# ─── Phony Targets ───────────────────────────────────────────────────────────
.PHONY: all build run test test-verbose test-cover lint fmt vet tidy clean install snapshot help

.DEFAULT_GOAL := help

# ─── Default ─────────────────────────────────────────────────────────────────
all: fmt vet lint test build ## Run fmt, vet, lint, test, build in sequence

# ─── Build ───────────────────────────────────────────────────────────────────
build: ## Compile harvx into bin/harvx with version metadata
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(CMD_DIR)
	@echo "Built $(BIN_DIR)/$(BINARY) ($(VERSION))"

run: build ## Build and run harvx (pass ARGS='...' for arguments)
	@./$(BIN_DIR)/$(BINARY) $(ARGS)

# ─── Test ────────────────────────────────────────────────────────────────────
test: ## Run all tests with race detection
	go test -race -count=1 ./...

test-verbose: ## Run all tests with verbose output
	go test -race -count=1 -v ./...

test-cover: ## Run tests with coverage and generate HTML report
	@mkdir -p $(BIN_DIR)
	go test -race -count=1 -coverprofile=$(BIN_DIR)/coverage.out ./...
	go tool cover -html=$(BIN_DIR)/coverage.out -o $(BIN_DIR)/coverage.html
	@echo "Coverage report: $(BIN_DIR)/coverage.html"

# ─── Lint ────────────────────────────────────────────────────────────────────
lint: ## Run golangci-lint (install separately if not found)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found. Install it with:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		echo "Or see: https://golangci-lint.run/welcome/install/"; \
		exit 1; \
	fi

# ─── Format ──────────────────────────────────────────────────────────────────
fmt: ## Format Go source files with gofmt and goimports
	gofmt -w .
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi

# ─── Vet ─────────────────────────────────────────────────────────────────────
vet: ## Run go vet static analysis
	go vet ./...

# ─── Module ──────────────────────────────────────────────────────────────────
tidy: ## Run go mod tidy
	go mod tidy

# ─── Clean ───────────────────────────────────────────────────────────────────
clean: ## Remove bin/ and dist/ directories and build artifacts
	rm -rf $(BIN_DIR) $(DIST_DIR)
	@echo "Cleaned build artifacts"

# ─── Install ─────────────────────────────────────────────────────────────────
install: ## Install harvx to $GOPATH/bin with version metadata
	CGO_ENABLED=0 go install -ldflags "$(LDFLAGS)" $(CMD_DIR)
	@echo "Installed harvx to $$(go env GOPATH)/bin/harvx"

# ─── Release ─────────────────────────────────────────────────────────────────
snapshot: ## GoReleaser snapshot build (safe: no publish)
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "goreleaser not installed. See: https://goreleaser.com/install/"; \
		exit 1; \
	fi
	goreleaser release --snapshot --clean

# ─── Help ────────────────────────────────────────────────────────────────────
help: ## Show available targets with descriptions
	@echo "Harvx Development Targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Build metadata:"
	@echo "  VERSION=$(VERSION)"
	@echo "  COMMIT=$(COMMIT)"

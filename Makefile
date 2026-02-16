# Makefile for Harvx — Go CLI tool
# Use: make help | make ci | make build | make run ARGS='generate .'

BINARY_NAME := harvx
MAIN_PKG    := ./cmd/harvx
BIN_DIR     := bin
DIST_DIR    := dist

# Version metadata (injected via ldflags)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
	-X github.com/harvx/harvx/internal/cli.version=$(VERSION) \
	-X github.com/harvx/harvx/internal/cli.commit=$(COMMIT) \
	-X github.com/harvx/harvx/internal/cli.date=$(DATE)

.DEFAULT_GOAL := help

# ─── Help ─────────────────────────────────────────────

.PHONY: help
help: ## Show available targets
	@echo "Harvx Makefile"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""

# ─── Quality ──────────────────────────────────────────

.PHONY: fmt
fmt: ## Auto-format all Go files
	gofmt -w .

.PHONY: vet
vet: ## Run go vet static analysis
	go vet ./...

.PHONY: test
test: ## Run all tests with race detector
	go test -race -count=1 ./...

.PHONY: tidy
tidy: ## Run go mod tidy
	go mod tidy

.PHONY: ci
ci: ## Run full pipeline checks (fmt + vet + lint + tidy + test)
	./run_pipeline_checks.sh --with-build

# ─── Build ────────────────────────────────────────────

.PHONY: build
build: ## Build binary to bin/harvx with version info
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_PKG)

.PHONY: run
run: ## Run the CLI (pass ARGS='...' for arguments)
	go run -ldflags "$(LDFLAGS)" $(MAIN_PKG) $(ARGS)

.PHONY: install
install: ## Install harvx to $GOPATH/bin
	CGO_ENABLED=0 go install -ldflags "$(LDFLAGS)" $(MAIN_PKG)

# ─── Release ──────────────────────────────────────────

.PHONY: snapshot
snapshot: ## GoReleaser snapshot build (safe: no publish)
	@if ! command -v goreleaser &>/dev/null; then \
		echo "goreleaser not installed. See: https://goreleaser.com/install/"; \
		exit 1; \
	fi
	goreleaser release --snapshot --clean

# ─── Cleanup ──────────────────────────────────────────

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) $(DIST_DIR)

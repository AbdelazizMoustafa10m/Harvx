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
.PHONY: all build run test test-verbose test-cover test-integration lint fmt vet tidy clean install snapshot release-snapshot release-check completions man bench bench-compare bench-update-baseline fuzz golden-update help

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

test-integration: ## Run integration tests against OSS test repos (build tag: integration)
	go test -tags integration -race -count=1 -timeout=5m -v ./tests/integration/

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
clean: ## Remove bin/, dist/, completions/, and man/ directories and build artifacts
	rm -rf $(BIN_DIR) $(DIST_DIR) completions man
	@echo "Cleaned build artifacts"

# ─── Completions ────────────────────────────────────────────────────────────
completions: build ## Generate shell completion scripts in completions/
	@mkdir -p completions
	./$(BIN_DIR)/$(BINARY) completion bash > completions/harvx.bash
	./$(BIN_DIR)/$(BINARY) completion zsh > completions/_harvx
	./$(BIN_DIR)/$(BINARY) completion fish > completions/harvx.fish
	@echo "Generated shell completions in completions/"

# ─── Man Pages ──────────────────────────────────────────────────────────────
man: build ## Generate man pages in man/
	./$(BIN_DIR)/$(BINARY) docs man --output-dir ./man

# ─── Benchmarks ─────────────────────────────────────────────────────────────
BENCH_DIR      := ./internal/benchmark
BENCH_BASELINE := testdata/benchmarks/baseline.txt
BENCH_CURRENT  := $(BIN_DIR)/bench-current.txt

bench: ## Run all benchmarks (build tag: bench)
	@mkdir -p $(BIN_DIR)
	go test -tags bench -bench=. -benchmem -count=5 -run=^$$ $(BENCH_DIR) | tee $(BENCH_CURRENT)
	@echo "Results saved to $(BENCH_CURRENT)"

bench-compare: bench ## Compare current benchmarks against baseline using benchstat
	@if ! command -v benchstat >/dev/null 2>&1; then \
		echo "benchstat not found. Install with:"; \
		echo "  go install golang.org/x/perf/cmd/benchstat@latest"; \
		exit 1; \
	fi
	@if [ ! -f $(BENCH_BASELINE) ]; then \
		echo "No baseline found at $(BENCH_BASELINE). Run 'make bench-update-baseline' first."; \
		exit 1; \
	fi
	benchstat $(BENCH_BASELINE) $(BENCH_CURRENT)

bench-update-baseline: bench ## Save current benchmark results as the new baseline
	@mkdir -p $$(dirname $(BENCH_BASELINE))
	cp $(BENCH_CURRENT) $(BENCH_BASELINE)
	@echo "Baseline updated: $(BENCH_BASELINE)"

# ─── Fuzz Testing ──────────────────────────────────────────────────────────
FUZZ_TIME ?= 30s

fuzz: ## Run all fuzz tests (FUZZ_TIME=30s per test, configurable)
	@echo "Running fuzz tests ($(FUZZ_TIME) per target)..."
	go test -fuzz=FuzzRedactContent    -fuzztime=$(FUZZ_TIME) -run=^$$ ./internal/security/
	go test -fuzz=FuzzRedactHighEntropy -fuzztime=$(FUZZ_TIME) -run=^$$ ./internal/security/
	go test -fuzz=FuzzRedactEnvFile    -fuzztime=$(FUZZ_TIME) -run=^$$ ./internal/security/
	go test -fuzz=FuzzRedactMixedContent -fuzztime=$(FUZZ_TIME) -run=^$$ ./internal/security/
	go test -fuzz=FuzzParseConfig      -fuzztime=$(FUZZ_TIME) -run=^$$ ./internal/config/
	go test -fuzz=FuzzProfileInheritance -fuzztime=$(FUZZ_TIME) -run=^$$ ./internal/config/
	go test -fuzz=FuzzGlobPattern      -fuzztime=$(FUZZ_TIME) -run=^$$ ./internal/config/
	@echo "All fuzz tests completed without crashes"

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

release-snapshot: ## GoReleaser snapshot build with all 5 platform binaries
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "goreleaser not installed. See: https://goreleaser.com/install/"; \
		exit 1; \
	fi
	goreleaser build --snapshot --clean

release-check: ## Validate .goreleaser.yaml configuration
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "goreleaser not installed. See: https://goreleaser.com/install/"; \
		exit 1; \
	fi
	goreleaser check

# ─── Golden Tests ───────────────────────────────────────────────────────────
golden-update: ## Regenerate golden test files
	go test ./internal/golden/ -update -count=1

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

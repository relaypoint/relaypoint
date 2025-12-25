# Relaypoint Makefile
# Local development Makefile - not used in CI/CD pipelines

BINARY_NAME := relaypoint
BUILD_DIR := dist
CMD_DIR := ./cmd/relaypoint
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOVET := $(GOCMD) vet
GOFMT := gofmt
GOMOD := $(GOCMD) mod
GORUN := $(GOCMD) run

# Colors for output
CYAN := \033[0;36m
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

.PHONY: all build clean test fmt lint fmt vet run help
.PHONY: build-all build-linux build-darwin build-windows
.PHONY: dev mock-backends integration-test bench
.PHONY: deps deps-update deps-tidy deps-verify
.PHONY: install uninstall

# Default target
all: clean lint test build

# help: Show this help message
help:
	@echo ""
	@echo "$(CYAN)Relaypoint$(NC) - Development Commands"
	@echo ""
	@echo "$(YELLOW)Usage:$(NC)"
	@echo "	 make $(GREEN)<target>$(NC)"
	@echo ""
	@echo "$(YELLOW)Build:$(NC)"
	@echo "  $(GREEN)build$(NC)          Build binary for current OS/arch"
	@echo "  $(GREEN)build-all$(NC)      Build binaries for all platforms"
	@echo "  $(GREEN)build-linux$(NC)    Build for Linux (amd64 + arm64)"
	@echo "  $(GREEN)build-darwin$(NC)   Build for macOS (amd64 + arm64)"
	@echo "  $(GREEN)build-windows$(NC)  Build for Windows (amd64)"
	@echo "  $(GREEN)install$(NC)        Install binary to /usr/local/bin"
	@echo "  $(GREEN)uninstall$(NC)      Remove binary from /usr/local/bin"
	@echo "  $(GREEN)clean$(NC)          Remove build artifacts"
	@echo ""
	@echo "$(YELLOW)Development:$(NC)"
	@echo "  $(GREEN)run$(NC)            Run gateway with default config"
	@echo "  $(GREEN)dev$(NC)            Run gateway with hot reload (requires air)"
	@echo "  $(GREEN)mock-backends$(NC)  Start mock backend services"
	@echo "  $(GREEN)demo$(NC)           Start full demo environment"
	@echo ""
	@echo "$(YELLOW)Testing:$(NC)"
	@echo "  $(GREEN)test$(NC)           Run unit tests"
	@echo "  $(GREEN)test-verbose$(NC)   Run tests with verbose output"
	@echo "  $(GREEN)test-race$(NC)      Run tests with race detector"
	@echo "  $(GREEN)test-cover$(NC)     Run tests with coverage report"
	@echo "  $(GREEN)integration-test$(NC) Run integration tests"
	@echo "  $(GREEN)bench$(NC)          Run benchmarks"
	@echo ""
	@echo "$(YELLOW)Code Quality:$(NC)"
	@echo "  $(GREEN)lint$(NC)           Run golangci-lint"
	@echo "  $(GREEN)fmt$(NC)            Format code with gofmt"
	@echo "  $(GREEN)fmt-check$(NC)      Check if code is formatted"
	@echo "  $(GREEN)vet$(NC)            Run go vet"
	@echo "  $(GREEN)check$(NC)          Run all checks (fmt, vet, lint)"
	@echo ""
	@echo "$(YELLOW)Dependencies:$(NC)"
	@echo "  $(GREEN)deps$(NC)           Download dependencies"
	@echo "  $(GREEN)deps-update$(NC)    Update all dependencies"
	@echo "  $(GREEN)deps-tidy$(NC)      Tidy go.mod"
	@echo "  $(GREEN)deps-verify$(NC)    Verify dependencies"
	@echo ""

# ========================= Build =========================

## build: Build binary for current OS/arch
build:
	@echo "$(CYAN)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "$(GREEN)✓ Build completed: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

## build-all: Build binaries for all platforms
build-all: build-linux build-darwin build-windows
	@echo "$(GREEN)✓ All builds completed$(NC)"
	@ls -lh $(BUILD_DIR)/

## build-linux: Build for Linux
build-linux:
	@echo "$(CYAN)Building Linux binaries...$(NC)"
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	@echo "$(GREEN)✓ Linux builds completed$(NC)"

## build-darwin: Build for macOS
build-darwin:
	@echo "$(CYAN)Building macOS binaries...$(NC)"
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	@echo "$(GREEN)✓ macOS builds completed$(NC)"

## build-windows: Build for Windows
build-windows:
	@echo "$(CYAN)Building Windows binaries...$(NC)"
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
	@echo "$(GREEN)✓ Windows builds completed$(NC)"

## install: Install binary to /usr/local/bin
install: build
	@echo "$(CYAN)Installing $(BINARY_NAME) to /usr/local/bin...$(NC)"
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "$(GREEN)✓ Installed to /usr/local/bin/$(BINARY_NAME)$(NC)"

## uninstall: Remove binary from /usr/local/bin
uninstall:
	@echo "$(CYAN)Uninstalling $(BINARY_NAME) from /usr/local/bin...$(NC)"
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "$(GREEN)✓ Uninstalled from /usr/local/bin/$(BINARY_NAME)$(NC)"

# ========================= Clean =========================

## clean: Remove build artifacts
clean:
	@echo "$(CYAN)Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@rm -f ${BINARY_NAME} mock_backend
	@echo "$(GREEN)✓ Clean completed$(NC)"

# ==================== Development ====================

## run: Run relaypoint with default config
run: build
	@echo "$(CYAN)Starting $(BINARY_NAME)...$(NC)"
	$(BUILD_DIR)/$(BINARY_NAME) -config relaypoint.yml

## run-config: Run relaypoint with custom config (usage: make run-config CONFIG=path/to/config.yaml)
run-config: build
	@echo "$(CYAN)Starting $(BINARY_NAME) with config: $(CONFIG)...$(NC)"
	$(BUILD_DIR)/$(BINARY_NAME) -config $(CONFIG)

## dev: Run relaypoint with hot reload (requires air)
dev:
	@command -v air >/dev/null 2>&1 || { echo >&2 "$(RED)✗ air is not installed. Please install it first: go install github.com/air-verse/air@latest$(NC)"; exit 1; }
	@echo "$(CYAN)Starting development server with hot reload...$(NC)"
	air -c .air.toml 2> /dev/null || air

## mock-backends: Start mock backend services
mock-backends:
	@echo "$(CYAN)Building mock backend...$(NC)"
	@$(GOBUILD) -o mock_backend ./test/mock_backend.go
	@echo "$(CYAN)Starting mock backends on ports 3001-3003...$(NC)"
	@./mock_backend -port 3001 -name "backend-1" &
	@./mock_backend -port 3002 -name "backend-2" &
	@./mock_backend -port 3003 -name "backend-3" &
	@echo "$(GREEN)✓ Mock backends running$(NC)"
	@echo "  - http://localhost:3001 (backend-1)"
	@echo "  - http://localhost:3002 (backend-2)"
	@echo "  - http://localhost:3003 (backend-3)"

## stop-backends: Stop mock backend services
stop-backends:
	@echo "$(CYAN)Stopping mock backend services...$(NC)"
	@pkill -f mock_backend 2>/dev/null || true
	@echo "$(GREEN)✓ Mock backends stopped$(NC)"

## demo: Start full demo environment (mock backends + relaypoint)
demo: build mock-backends
	@sleep 1
	@echo "$(CYAN)Starting Relaypoint...$(NC)"
	@$(BUILD_DIR)/$(BINARY_NAME) -config test/test_config.yml &
	@sleep 2
	@echo ""
	@echo "$(GREEN)✓ Demo environment running!$(NC)"
	@echo ""
	@echo "$(YELLOW)Endpoints:$(NC)"
	@echo "  Relaypoint:  http://localhost:8080"
	@echo "  Metrics:  http://localhost:9090/metrics"
	@echo "  Health:   http://localhost:8080/health"
	@echo ""
	@echo "$(YELLOW)Try:$(NC)"
	@echo "  curl http://localhost:8080/api/v1/users"
	@echo "  curl http://localhost:8080/api/v1/orders"
	@echo ""
	@echo "$(YELLOW)Stop with:$(NC) make demo-stop"

## demo-stop: Stop demo environment
demo-stop: stop-backends
	@echo "$(CYAN)Stopping Relaypoint...$(NC)"
	@pkill -f "$(BINARY_NAME)" 2>/dev/null || true
	@echo "$(GREEN)✓ Demo stopped$(NC)"

# ========================= Testing =========================

## test: Run unit tests
test:
	@echo "$(CYAN)Running unit tests...$(NC)"
	@$(GOTEST) ./...
	@echo "$(GREEN)✓ Tests passed$(NC)"

## test-verbose: Run tests with verbose output
test-verbose:
	@echo "$(CYAN)Running unit tests (verbose)...$(NC)"
	@$(GOTEST) -v ./...

## test-race: Run tests with race detector
test-race:
	@echo "$(CYAN)Running unit tests (race detector)...$(NC)"
	@$(GOTEST) -race ./...
	@echo "$(GREEN)✓ No race conditions detected$(NC)"

## test-cover: Run tests with coverage report
test-cover:
	@echo "$(CYAN)Running unit tests with coverage...$(NC)"
	@$(GOTEST) -coverprofile=coverage.out -covermode=atomic ./...
	@$(GOCMD) tool cover -func=coverage.out
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage report generated: coverage.html$(NC)"

## test-short: Run tests with short flag
test-short:
	@echo "$(CYAN)Running unit tests (short)...$(NC)"
	@$(GOTEST) -short ./...

## integration-test: Run integration tests
integration-test: build
	@echo "$(CYAN)Running integration tests...$(NC)"
	@$(GOBUILD) -o mock_backend ./test/mock_backend.go
	@./mock_backend -port 3001 -name "backend-1" &
	@./mock_backend -port 3002 -name "backend-2" &
	@sleep 1
	@$(BUILD_DIR)/$(BINARY_NAME) -config test/test_config.yaml &
	@sleep 2
	@echo "Testing basic routing..."
	@curl -sf http://localhost:8080/api/v1/users > /dev/null && echo "$(GREEN)✓ Basic routing$(NC)" || echo "$(RED)✗ Basic routing failed$(NC)"
	@echo "Testing health endpoint..."
	@curl -sf http://localhost:8080/health > /dev/null && echo "$(GREEN)✓ Health endpoint$(NC)" || echo "$(RED)✗ Health endpoint failed$(NC)"
	@echo "Testing metrics endpoint..."
	@curl -sf http://localhost:9090/metrics > /dev/null && echo "$(GREEN)✓ Metrics endpoint$(NC)" || echo "$(RED)✗ Metrics endpoint failed$(NC)"
	@pkill -f mock_backend 2>/dev/null || true
	@pkill -f "$(BINARY_NAME)" 2>/dev/null || true
	@echo "$(GREEN)✓ Integration tests complete$(NC)"

## bench: Run benchmarks
bench:
	@echo "$(CYAN)Running benchmarks...$(NC)"
	@$(GOTEST) -bench=. -benchmem ./...

## bench-cpu: Run benchmarks with CPU profiling
bench-cpu:
	@echo "$(CYAN)Running benchmarks with CPU profiling...$(NC)"
	@$(GOTEST) -bench=. -cpuprofile=cpu.prof ./internal/router
	@echo "$(GREEN)View with: go tool pprof cpu.prof$(NC)"

## bench-mem: Run benchmarks with memory profiling
bench-mem:
	@echo "$(CYAN)Running benchmarks with memory profiling...$(NC)"
	@$(GOTEST) -bench=. -memprofile=mem.prof ./internal/router
	@echo "$(GREEN)View with: go tool pprof mem.prof$(NC)"

# ===================== Code Quality =====================

## lint: Run golangci-lint
lint:
	@echo "$(CYAN)Running golangci-lint...$(NC)"
	@command -v golangci-lint >/dev/null 2>&1 || { echo "$(YELLOW)Installing golangci-lint...$(NC)"; go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; }
	@golangci-lint run ./...
	@echo "$(GREEN)✓ Lint passed$(NC)"

## lint-fix: Run golangci-lint with auto-fix
lint-fix:
	@echo "$(CYAN)Running golangci-lint with auto-fix...$(NC)"
	golangci-lint run --fix ./...
	@echo "$(GREEN)✓ Lint fixes applied$(NC)"

## fmt: Format code with gofmt
fmt:
	@echo "$(CYAN)Formatting code with gofmt...$(NC)"
	@$(GOFMT) -s -w .
	@echo "$(GREEN)✓ Code formatted$(NC)"

## fmt-check: Check if code is formatted
fmt-check:
	@echo "$(CYAN)Checking code formatting with gofmt...$(NC)"
	@test -z "$$($(GOFMT) -l .)" || { echo "$(RED)Code not formatted. Run: make fmt$(NC)"; $(GOFMT) -d .; exit 1; }
	@echo "$(GREEN)✓ Code is properly formatted$(NC)"

## vet: Run go vet
vet:
	@echo "$(CYAN)Running go vet...$(NC)"
	@$(GOVET) ./...
	@echo "$(GREEN)✓ go vet passed$(NC)"

## check: Run all checks (fmt, vet, lint)
check: fmt-check vet lint
	@echo "$(GREEN)✓ All checks passed$(NC)"

## security: Run security scan (requires gosec)
security:
	@echo "$(CYAN)Running security scan...$(NC)"
	@command -v gosec >/dev/null 2>&1 || { echo "$(YELLOW)Installing gosec...$(NC)"; go install github.com/securego/gosec/v2/cmd/gosec@latest; }
	gosec -quiet ./...
	@echo "$(GREEN)✓ Security scan passed$(NC)"

# ===================== Dependencies =====================

## deps: Download dependencies
deps:
	@echo "$(CYAN)Downloading dependencies...$(NC)"
	@$(GOMOD) download
	@echo "$(GREEN)✓ Dependencies downloaded$(NC)"

## deps-update: Update all dependencies
deps-update:
	@echo "$(CYAN)Updating dependencies...$(NC)"
	@$(GOCMD) get -u ./...
	@$(GOMOD) tidy
	@echo "$(GREEN)✓ Dependencies updated$(NC)"

## deps-tidy: Tidy go.mod
deps-tidy:
	@echo "$(CYAN)Tidying go.mod...$(NC)"
	@$(GOMOD) tidy
	@echo "$(GREEN)✓ go.mod tidied$(NC)"

## deps-verify: Verify dependencies
deps-verify:
	@echo "$(CYAN)Verifying dependencies...$(NC)"
	@$(GOMOD) verify
	@echo "$(GREEN)✓ Dependencies verified$(NC)"

## deps-graph: Generate dependencies graph
deps-graph:
	@echo "$(CYAN)Generating dependency graph...$(NC)"
	@command -v modgraphviz >/dev/null 2>&1 || { echo "$(YELLOW)Installing modgraphviz...$(NC)"; go install github.com/lucasepe/modgraphviz@latest; }
	$(GOCMD) mod graph | modgraphviz | dot -Tpng -o deps.png
	@echo "$(GREEN)✓ Dependency graph: deps.png$(NC)"

# ==================== Release ====================

## release-dry: Dry run release (shows what would be built)
release-dry:
	@echo "$(CYAN)Release dry run for version $(VERSION)...$(NC)"
	@echo "Would build:"
	@echo "  - $(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz"
	@echo "  - $(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz"
	@echo "  - $(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz"
	@echo "  - $(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz"
	@echo "  - $(BINARY_NAME)-$(VERSION)-windows-amd64.zip"

## release: Build release binaries and packages
release: clean build-all
	@echo "$(CYAN)Creating release archives...$(NC)"
	@mkdir -p $(BUILD_DIR)/release
	@# Linux amd64
	@tar -czvf $(BUILD_DIR)/release/relaypoint-$(VERSION)-linux-amd64.tar.gz \
		-C $(BUILD_DIR) $(BINARY_NAME)-linux-amd64 \
		-C $(PWD) README.md relaypoint.yml examples/
	@# Linux arm64
	@tar -czvf $(BUILD_DIR)/release/relaypoint-$(VERSION)-linux-arm64.tar.gz \
		-C $(BUILD_DIR) $(BINARY_NAME)-linux-arm64 \
		-C $(PWD) README.md relaypoint.yml examples/
	@# Darwin amd64
	@tar -czvf $(BUILD_DIR)/release/relaypoint-$(VERSION)-darwin-amd64.tar.gz \
		-C $(BUILD_DIR) $(BINARY_NAME)-darwin-amd64 \
		-C $(PWD) README.md relaypoint.yml examples/
	@# Darwin arm64
	@tar -czvf $(BUILD_DIR)/release/relaypoint-$(VERSION)-darwin-arm64.tar.gz \
		-C $(BUILD_DIR) $(BINARY_NAME)-darwin-arm64 \
		-C $(PWD) README.md relaypoint.yml examples/
	@# Windows
	@cd $(BUILD_DIR) && zip -j release/relaypoint-$(VERSION)-windows-amd64.zip \
		$(BINARY_NAME)-windows-amd64.exe \
		../README.md ../relaypoint.yml
	@echo "$(GREEN)✓ Release archives created in $(BUILD_DIR)/release/$(NC)"
	@ls -lh $(BUILD_DIR)/release/
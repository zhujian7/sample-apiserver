# MyTest API Server Makefile

# Variables
BINARY_NAME := mytest-apiserver
DOCKER_IMAGE := quay.io/zhujian/mytest-apiserver
VERSION ?= dev
GO_VERSION := 1.24
NAMESPACE := my-apiserver-system

# Go related variables
GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/bin
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt

# Docker related variables
DOCKER := docker
DOCKER_BUILD_ARGS := --platform linux/amd64

# Kubernetes related variables
KUBECTL := kubectl
KIND := kind
CLUSTER_NAME := kind

# Build flags
LDFLAGS := -w -s
BUILD_FLAGS := -ldflags "$(LDFLAGS)"

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

.PHONY: help
help: ## Show this help message
	@echo "$(BLUE)MyTest API Server - Available Commands:$(NC)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "$(BLUE)Usage:$(NC)\n  make $(GREEN)<target>$(NC)\n\n$(BLUE)Targets:$(NC)\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: all
all: clean fmt vet test build ## Run all checks and build

# Development targets
.PHONY: fmt
fmt: ## Format Go code
	@echo "$(YELLOW)Formatting Go code...$(NC)"
	@$(GOFMT) ./...

.PHONY: vet
vet: ## Run go vet
	@echo "$(YELLOW)Running go vet...$(NC)"
	@$(GOCMD) vet ./...

.PHONY: lint
lint: ## Run golangci-lint (requires golangci-lint to be installed)
	@echo "$(YELLOW)Running golangci-lint...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "$(RED)golangci-lint not found. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(NC)"; \
	fi

.PHONY: deps
deps: ## Download dependencies
	@echo "$(YELLOW)Downloading dependencies...$(NC)"
	@$(GOMOD) download
	@$(GOMOD) verify

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "$(YELLOW)Updating dependencies...$(NC)"
	@$(GOMOD) tidy
	@$(GOGET) -u ./...

# Build targets
.PHONY: build
build: ## Build the binary
	@echo "$(YELLOW)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(GOBIN)
	@$(GOBUILD) $(BUILD_FLAGS) -o $(GOBIN)/$(BINARY_NAME) .
	@echo "$(GREEN)Binary built: $(GOBIN)/$(BINARY_NAME)$(NC)"

.PHONY: build-linux
build-linux: ## Build binary for Linux
	@echo "$(YELLOW)Building $(BINARY_NAME) for Linux...$(NC)"
	@mkdir -p $(GOBIN)
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(GOBIN)/$(BINARY_NAME)-linux .
	@echo "$(GREEN)Linux binary built: $(GOBIN)/$(BINARY_NAME)-linux$(NC)"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@$(GOCLEAN)
	@rm -rf $(GOBIN)
	@rm -rf coverage/
	@echo "$(GREEN)Clean completed$(NC)"

# Test targets
.PHONY: test
test: ## Run all tests
	@echo "$(YELLOW)Running all tests...$(NC)"
	@./test.sh all

.PHONY: test-unit
test-unit: ## Run unit tests only
	@echo "$(YELLOW)Running unit tests...$(NC)"
	@./test.sh unit

.PHONY: test-integration
test-integration: ## Run integration tests only
	@echo "$(YELLOW)Running integration tests...$(NC)"
	@./test.sh integration

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "$(YELLOW)Running tests with coverage...$(NC)"
	@./test.sh coverage

.PHONY: test-race
test-race: ## Run tests with race detection
	@echo "$(YELLOW)Running tests with race detection...$(NC)"
	@$(GOTEST) -race -v ./pkg/...

.PHONY: benchmark
benchmark: ## Run benchmarks
	@echo "$(YELLOW)Running benchmarks...$(NC)"
	@$(GOTEST) -bench=. -benchmem ./pkg/...

# Docker targets
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "$(YELLOW)Building Docker image $(DOCKER_IMAGE):$(VERSION)...$(NC)"
	@$(DOCKER) build $(DOCKER_BUILD_ARGS) -t $(DOCKER_IMAGE):$(VERSION) .
	@$(DOCKER) tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest
	@echo "$(GREEN)Docker image built: $(DOCKER_IMAGE):$(VERSION)$(NC)"

.PHONY: docker-push
docker-push: ## Push Docker image to registry
	@echo "$(YELLOW)Pushing Docker image $(DOCKER_IMAGE):$(VERSION)...$(NC)"
	@$(DOCKER) push $(DOCKER_IMAGE):$(VERSION)
	@$(DOCKER) push $(DOCKER_IMAGE):latest
	@echo "$(GREEN)Docker image pushed$(NC)"

.PHONY: docker-run
docker-run: ## Run Docker container locally
	@echo "$(YELLOW)Running Docker container...$(NC)"
	@$(DOCKER) run --rm -p 8443:8443 $(DOCKER_IMAGE):$(VERSION)

# Kubernetes targets
.PHONY: kind-create
kind-create: ## Create Kind cluster
	@echo "$(YELLOW)Creating Kind cluster...$(NC)"
	@./deploy/kind/setup.sh
	@echo "$(GREEN)Kind cluster created$(NC)"

.PHONY: kind-delete
kind-delete: ## Delete Kind cluster
	@echo "$(YELLOW)Deleting Kind cluster...$(NC)"
	@$(KIND) delete cluster --name $(CLUSTER_NAME)
	@echo "$(GREEN)Kind cluster deleted$(NC)"

.PHONY: deploy
deploy: ## Deploy to Kubernetes
	@echo "$(YELLOW)Deploying to Kubernetes...$(NC)"
	@./deploy/deploy.sh install
	@echo "$(GREEN)Deployment completed$(NC)"

.PHONY: undeploy
undeploy: ## Remove deployment from Kubernetes
	@echo "$(YELLOW)Removing deployment from Kubernetes...$(NC)"
	@./deploy/deploy.sh uninstall
	@echo "$(GREEN)Undeployment completed$(NC)"

.PHONY: status
status: ## Check deployment status
	@echo "$(YELLOW)Checking deployment status...$(NC)"
	@./deploy/deploy.sh status

.PHONY: logs
logs: ## Show API server logs
	@echo "$(YELLOW)Showing API server logs...$(NC)"
	@$(KUBECTL) logs -n $(NAMESPACE) -l app=mytest-apiserver --tail=50 -f

# Development workflow targets
.PHONY: dev-setup
dev-setup: deps kind-create deploy ## Set up complete development environment
	@echo "$(GREEN)Development environment ready!$(NC)"
	@echo "$(BLUE)Quick test:$(NC)"
	@echo "  kubectl get apiservice v1alpha1.things.myorg.io"
	@echo "  kubectl api-resources | grep things.myorg.io"

.PHONY: dev-teardown
dev-teardown: undeploy kind-delete ## Tear down development environment
	@echo "$(GREEN)Development environment cleaned up$(NC)"

.PHONY: dev-restart
dev-restart: undeploy docker-build deploy ## Rebuild and redeploy for development
	@echo "$(GREEN)Development restart completed$(NC)"

.PHONY: quick-test
quick-test: ## Quick end-to-end test with sample resources
	@echo "$(YELLOW)Running quick end-to-end test...$(NC)"
	@kubectl apply -f - <<EOF || true\n\
	apiVersion: things.myorg.io/v1alpha1\n\
	kind: Widget\n\
	metadata:\n\
	  name: test-widget\n\
	  namespace: default\n\
	spec:\n\
	  name: "Test Widget"\n\
	  description: "Quick test widget"\n\
	  size: 42\n\
	EOF
	@kubectl apply -f - <<EOF || true\n\
	apiVersion: things.myorg.io/v1alpha1\n\
	kind: Gadget\n\
	metadata:\n\
	  name: test-gadget\n\
	  namespace: default\n\
	spec:\n\
	  type: "sensor"\n\
	  version: "v1.0"\n\
	  enabled: true\n\
	  priority: 10\n\
	EOF
	@echo "$(BLUE)Created test resources:$(NC)"
	@$(KUBECTL) get widgets,gadgets
	@echo "$(YELLOW)Cleaning up test resources...$(NC)"
	@$(KUBECTL) delete widget test-widget --ignore-not-found=true
	@$(KUBECTL) delete gadget test-gadget --ignore-not-found=true
	@echo "$(GREEN)Quick test completed$(NC)"

# Release targets
.PHONY: release-build
release-build: clean fmt vet test build-linux docker-build ## Build release artifacts
	@echo "$(GREEN)Release build completed$(NC)"

.PHONY: release
release: release-build docker-push ## Create and push a release
	@echo "$(GREEN)Release $(VERSION) completed$(NC)"

# CI targets
.PHONY: ci-test
ci-test: deps fmt vet test-unit ## Run CI tests (no integration tests)
	@echo "$(GREEN)CI tests completed$(NC)"

.PHONY: ci-build
ci-build: ci-test build docker-build ## Full CI build
	@echo "$(GREEN)CI build completed$(NC)"

# Info targets
.PHONY: info
info: ## Show project information
	@echo "$(BLUE)Project Information:$(NC)"
	@echo "  Binary: $(BINARY_NAME)"
	@echo "  Docker Image: $(DOCKER_IMAGE):$(VERSION)"
	@echo "  Go Version: $(GO_VERSION)"
	@echo "  Namespace: $(NAMESPACE)"
	@echo "  Cluster: $(CLUSTER_NAME)"

.PHONY: check-tools
check-tools: ## Check if required tools are installed
	@echo "$(YELLOW)Checking required tools...$(NC)"
	@command -v go >/dev/null 2>&1 || (echo "$(RED)❌ Go not installed$(NC)" && exit 1)
	@command -v docker >/dev/null 2>&1 || (echo "$(RED)❌ Docker not installed$(NC)" && exit 1)
	@command -v kubectl >/dev/null 2>&1 || (echo "$(RED)❌ kubectl not installed$(NC)" && exit 1)
	@command -v kind >/dev/null 2>&1 || (echo "$(RED)❌ Kind not installed$(NC)" && exit 1)
	@echo "$(GREEN)✅ All required tools are installed$(NC)"

# Default target
.DEFAULT_GOAL := help
KUBECONFIG_PATH := ${PWD}/kubeconfig
CONFIG_FILE := ./etc/debug-config.yaml

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: setup
setup: pre-commit golangci-lint goimports swaggo ## Setup development environment with all tools and hooks.
	@echo "Development environment setup completed!"

##@ Development

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

.PHONY: vet
vet: fmt ## Run go vet.
	@echo "Running go vet..."
	go vet ./...

.PHONY: imports
imports: goimports ## Run goimports on all go files.
	@echo "Running goimports..."
	$(GOIMPORTS) -w -local github.com/raids-lab/crater .

# if $(GOIMPORTS) -l -local github.com/raids-lab/crater .go, then error
.PHONY: import-check
import-check: goimports ## Check if goimports is needed.
	@echo "Running goimports..."
	@if [ -n "$$($(GOIMPORTS) -l -local github.com/raids-lab/crater .)" ]; then \
		echo "goimports needs to be run, please run 'make imports'"; \
		exit 1; \
	fi

.PHONY: lint
lint: fmt imports import-check golangci-lint ## Lint go files.
	@echo "Linting go files..."
	$(GOLANGCI_LINT) run --timeout 5m

.PHONY: curd
curd: ## Generate Gorm CURD code.
	@echo "Generating CURD code..."
	go run cmd/gorm-gen/curd/generate.go

.PHONY: migrate
migrate: ## Migrate database.
	@echo "Migrating database..."
	go run cmd/gorm-gen/models/migrate.go

.PHONY: docs
docs: swaggo ## Generate docs docs.
	@echo "Generating swagger docs..."
	$(SWAGGO) init -g cmd/crater/main.go -q --parseInternal --parseDependency

.PHONY: run
run: fmt docs imports pre-commit ## Run a controller from your host.
	export KUBECONFIG="$(KUBECONFIG_PATH)" && \
	go run cmd/crater/main.go --config-file "$(CONFIG_FILE)"

.PHONY: pre-commit-check
pre-commit-check: pre-commit ## Run pre-commit hook manually.
	@echo "Running pre-commit hook..."
	@$(GIT_HOOKS_DIR)/pre-commit && echo "Pre-commit hook completed successfully! You can now commit with confidence." || echo "Pre-commit hook failed! Please fix the issues and try again."

##@ Build

.PHONY: build
build: fmt lint docs ## Build manager binary.
	go build -ldflags="-w -s" -o bin/controller cmd/crater/main.go

.PHONY: build-migrate
build-migrate: fmt lint ## Build migration binary.
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/migrate cmd/gorm-gen/models/migrate.go

##@ Development Tools

## Location to install tools to
LOCATION ?= $(PWD)/bin
$(LOCATION):
	mkdir -p $(LOCATION)

## Tool Binaries
GOLANGCI_LINT ?= $(LOCATION)/golangci-lint
GOIMPORTS ?= $(LOCATION)/goimports
SWAGGO ?= $(LOCATION)/swag
HACK_DIR ?= $(PWD)/hack

## Tool Versions
GOLANGCI_LINT_VERSION ?= v2.2.1
SWAGGO_VERSION ?= v1.16.3
GOIMPORTS_VERSION ?= v0.28.0

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Install golangci-lint
$(GOLANGCI_LINT): $(LOCATION)
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."
	GOBIN=$(LOCATION) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: goimports
goimports: $(GOIMPORTS) ## Install goimports
$(GOIMPORTS): $(LOCATION)
	@echo "Installing goimports $(GOIMPORTS_VERSION)..."
	GOBIN=$(LOCATION) go install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)

.PHONY: swaggo
swaggo: $(SWAGGO) ## Install swaggo
$(SWAGGO): $(LOCATION)
	@echo "Installing swag $(SWAGGO_VERSION)..."
	GOBIN=$(LOCATION) go install github.com/swaggo/swag/cmd/swag@$(SWAGGO_VERSION)

##@ Git Hooks

## Location to install git hooks to
GIT_HOOKS_DIR ?= $(PWD)/.git/hooks
$(GIT_HOOKS_DIR):
	@echo "Creating git hooks directory..."
	@mkdir -p $(GIT_HOOKS_DIR)

.PHONY: pre-commit
pre-commit: $(GIT_HOOKS_DIR)/pre-commit ## Install git pre-commit hook.
$(GIT_HOOKS_DIR)/pre-commit: $(GIT_HOOKS_DIR) .githook/pre-commit
	@echo "Installing git pre-commit hook..."
	@cp .githook/pre-commit $(GIT_HOOKS_DIR)/pre-commit
	@chmod +x $(GIT_HOOKS_DIR)/pre-commit
	@echo "Git pre-commit hook installed successfully!"

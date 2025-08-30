KUBECONFIG_PATH := $(if $(KUBECONFIG),$(KUBECONFIG),${PWD}/kubeconfig)
CONFIG_FILE := ./etc/debug-config.yaml

# 颜色定义
RED := \033[31m
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
MAGENTA := \033[35m
CYAN := \033[36m
WHITE := \033[37m
RESET := \033[0m

ifeq ($(wildcard $(KUBECONFIG_PATH)),)
$(warning KUBECONFIG file not found at: $(KUBECONFIG_PATH))
$(warning Please ensure the kubeconfig file exists or set KUBECONFIG environment variable)
endif

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

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

.PHONY: show-kubeconfig
show-kubeconfig: ## Display current KUBECONFIG path
	@echo "$(CYAN)KUBECONFIG environment variable:$(RESET) $(if $(KUBECONFIG),$(KUBECONFIG),$(YELLOW)Not set$(RESET))"
	@echo "$(CYAN)Using KUBECONFIG path:$(RESET) $(KUBECONFIG_PATH)"
	@if [ -f "$(KUBECONFIG_PATH)" ]; then \
		echo "$(GREEN)✅ KUBECONFIG file exists$(RESET)"; \
	else \
		echo "$(RED)❌ KUBECONFIG file does NOT exist$(RESET)"; \
	fi

.PHONY: prepare
prepare: ## Prepare development environment with updated configs
	@echo "$(BLUE)Preparing development environment...$(RESET)"
	@if [ ! -f .debug.env ]; then \
		echo "$(YELLOW)Creating .debug.env file...$(RESET)"; \
		echo 'CRATER_BE_PORT=:8088' > .debug.env; \
		echo "$(GREEN)✅ .debug.env created successfully!$(RESET)"; \
	else \
		echo "$(GREEN)✅ .debug.env already exists$(RESET)"; \
	fi

##@ Development

.PHONY: fmt
fmt:
	@echo "$(BLUE)Formatting code...$(RESET)"
	go fmt ./...

.PHONY: vet
vet: fmt ## Run go vet.
	@echo "$(BLUE)Running go vet...$(RESET)"
	go vet ./...

.PHONY: imports
imports: goimports ## Run goimports on all go files.
	@echo "$(BLUE)Running goimports...$(RESET)"
	$(GOIMPORTS) -w -local github.com/raids-lab/crater .

# if $(GOIMPORTS) -l -local github.com/raids-lab/crater .go, then error
.PHONY: import-check
import-check: goimports ## Check if goimports is needed.
	@echo "$(BLUE)Checking imports...$(RESET)"
	@if [ -n "$$($(GOIMPORTS) -l -local github.com/raids-lab/crater .)" ]; then \
		echo "$(RED)❌ goimports needs to be run, please run 'make imports'$(RESET)"; \
		exit 1; \
	else \
		echo "$(GREEN)✅ Imports are properly formatted$(RESET)"; \
	fi

.PHONY: lint
lint: fmt imports import-check golangci-lint ## Lint go files.
	@echo "$(BLUE)Linting go files...$(RESET)"
	$(GOLANGCI_LINT) run --timeout 5m

.PHONY: curd
curd: ## Generate Gorm CURD code.
	@echo "$(MAGENTA)Generating CURD code...$(RESET)"
	go run cmd/gorm-gen/curd/generate.go
	@echo "$(GREEN)✅ CURD code generated successfully$(RESET)"

.PHONY: migrate
migrate: ## Migrate database.
	@echo "$(MAGENTA)Migrating database...$(RESET)"
	go run cmd/gorm-gen/models/migrate.go
	@echo "$(GREEN)✅ Database migration completed$(RESET)"

.PHONY: docs
docs: swaggo ## Generate docs docs.
	@echo "$(MAGENTA)Generating swagger docs...$(RESET)"
	$(SWAGGO) init -g cmd/crater/main.go -q --parseInternal --parseDependency
	@echo "$(GREEN)✅ Swagger docs generated$(RESET)"

.PHONY: run
run: prepare show-kubeconfig fmt docs imports pre-commit ## Run a controller from your host.
	@echo "$(GREEN)🚀 Starting crater backend server...$(RESET)"
	KUBECONFIG="$(KUBECONFIG_PATH)" go run cmd/crater/main.go

.PHONY: pre-commit-check
pre-commit-check: pre-commit ## Run pre-commit hook manually.
	@echo "$(BLUE)Running pre-commit hook...$(RESET)"
	@$(GIT_HOOKS_DIR)/pre-commit && echo "$(GREEN)✅ Pre-commit hook completed successfully! You can now commit with confidence.$(RESET)" || echo "$(RED)❌ Pre-commit hook failed! Please fix the issues and try again.$(RESET)"

##@ Build

.PHONY: build
build: fmt lint docs ## Build manager binary.
	@echo "$(YELLOW)🔨 Building controller binary...$(RESET)"
	go build -ldflags="-w -s" -o bin/controller cmd/crater/main.go
	@echo "$(GREEN)✅ Controller binary built successfully: bin/controller$(RESET)"

.PHONY: build-migrate
build-migrate: fmt lint ## Build migration binary.
	@echo "$(YELLOW)🔨 Building migration binary...$(RESET)"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/migrate cmd/gorm-gen/models/migrate.go
	@echo "$(GREEN)✅ Migration binary built successfully: bin/migrate$(RESET)"

##@ Development Tools

## Location to install tools to
LOCALBIN ?= $(PWD)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
GOIMPORTS ?= $(LOCALBIN)/goimports
SWAGGO ?= $(LOCALBIN)/swag
HACK_DIR ?= $(PWD)/hack

## Tool Versions
GOLANGCI_LINT_VERSION ?= v2.2.1
SWAGGO_VERSION ?= v1.16.6
GOIMPORTS_VERSION ?= v0.28.0

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Install golangci-lint
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

.PHONY: goimports
goimports: $(GOIMPORTS) ## Install goimports
$(GOIMPORTS): $(LOCALBIN)
	$(call go-install-tool,$(GOIMPORTS),golang.org/x/tools/cmd/goimports,$(GOIMPORTS_VERSION))

.PHONY: swaggo
swaggo: $(SWAGGO) ## Install swaggo
$(SWAGGO): $(LOCALBIN)
	$(call go-install-tool,$(SWAGGO),github.com/swaggo/swag/cmd/swag,$(SWAGGO_VERSION))

##@ Git Hooks

## Location to install git hooks to
GIT_HOOKS_DIR ?= $(shell git rev-parse --git-path hooks)
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

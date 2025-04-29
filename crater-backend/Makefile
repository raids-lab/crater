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
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

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
lint: fmt import-check golangci-lint ## Lint go files.
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

.PHONY: api-docs
swagger: ## Generate swagger docs.
	@echo "Generating swagger docs..."
	$(SWAGGO) init

.PHONY: run
run: fmt api-docs  ## Run a controller from your host.
	export KUBECONFIG="$(KUBECONFIG_PATH)" && \
	go run main.go --config-file "$(CONFIG_FILE)"

##@ Build

.PHONY: build
build: fmt lint swagger # generate vet ## Build manager binary.
	go build -ldflags="-w -s" -o bin/controller main.go

.PHONY: build-migrate
build-migrate: fmt lint
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/migrate cmd/gorm-gen/models/migrate.go

.PHONY: docker-build
docker-build: fmt swagger ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

.PHONY: build-backend
build-backend:
	docker build -t ${IMG} -f Dockerfile.
	docker push ${IMG}

##@ Build Dependencies

## Location to install dependencies to
LOCATION ?= $(PWD)/bin
$(LOCATION):
	mkdir -p $(LOCATION)

## Tool Binaries
GOLANGCI_LINT ?= $(LOCATION)/golangci-lint
GOIMPORTS ?= $(LOCATION)/goimports
SWAGGO ?= $(LOCATION)/swag
HACK_DIR ?= $(PWD)/hack

## Tool Versions
GOLANGCI_LINT_VERSION ?= v1.61.0
SWAGGO_VERSION ?= v1.16.3
GOIMPORTS_VERSION ?= v0.28.0

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Install golangci-lint
$(GOLANGCI_LINT): $(LOCATION)
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."
	GOBIN=$(LOCATION) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

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

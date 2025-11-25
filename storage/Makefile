
BINARY=mywebdav
MAIN_FILE=main.go

.PHONY: help
help:
	@echo "Usage:"
	@echo "  make run              - Run the application"
	@echo "  make fmt              - Format Go code"
	@echo "  make build            - Build the binary"
	@echo "  make clean            - Clean up build artifacts"
	@echo "  make pre-commit-check - Run pre-commit checks (golangci-lint)"

.PHONY: run
run: fmt
	@echo "Running application "
	go run $(MAIN_FILE)

.PHONY: fmt
fmt:
	@echo "Formatting Go code..."
	go fmt ./...

.PHONY: build
build: fmt
	@echo "Building binary..."
	go build -o $(BINARY) $(MAIN_FILE)

.PHONY: clean
clean:
	@echo "Cleaning up..."
	rm -f $(BINARY)

.PHONY: pre-commit-check
pre-commit-check: ## Run pre-commit checks (golangci-lint).
	@echo "Checking if golangci-lint is available..."
	@GOLANGCI_LINT=""; \
	if [ -f "../backend/bin/golangci-lint" ]; then \
		GOLANGCI_LINT="../backend/bin/golangci-lint"; \
		echo "Using golangci-lint from backend repository: $$GOLANGCI_LINT"; \
	elif command -v golangci-lint >/dev/null 2>&1; then \
		GOLANGCI_LINT="golangci-lint"; \
		echo "Using golangci-lint from system PATH"; \
	fi; \
	if [ -z "$$GOLANGCI_LINT" ]; then \
		echo "❌ Error: golangci-lint is not found"; \
		echo ""; \
		echo "Please install golangci-lint using one of the following methods:"; \
		echo "  1. Install in backend repository: cd ../backend && make golangci-lint"; \
		echo "  2. Install locally (see storage/README.md for details) (not recommended):"; \
		exit 1; \
	fi; \
	echo "Running golangci-lint..."; \
	if ! $$GOLANGCI_LINT run --fix; then \
		echo "❌ Storage lint failed!"; \
		exit 1; \
	fi; \
	echo "✅ Storage checks passed!"
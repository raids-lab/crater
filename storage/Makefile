
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
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "❌ Error: golangci-lint is not installed or not in PATH"; \
		echo "   Please install golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0"; \
		exit 1; \
	fi
	@echo "Running golangci-lint..."
	@if ! golangci-lint run --fix; then \
		echo "❌ Storage lint failed!"; \
		exit 1; \
	fi
	@echo "✅ Storage checks passed!"
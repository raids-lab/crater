PORT := $(shell if [ -f .env ]; then grep PORT .env | cut -d '=' -f2; else echo "7320"; fi)

BINARY=mywebdav
MAIN_FILE=main.go

.PHONY: help
help:
	@echo "Usage:"
	@echo "  make run        - Run the application"
	@echo "  make fmt        - Format Go code"
	@echo "  make build      - Build the binary"
	@echo "  make clean      - Clean up build artifacts"

.PHONY: run
run: fmt
	@echo "Running application on port ${PORT}..."
	go run $(MAIN_FILE) -port=$(PORT)

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
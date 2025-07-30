
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
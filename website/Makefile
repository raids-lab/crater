# Makefile for Crater Website

NODE_VERSION := $(strip $(shell cat .nvmrc))
NVM_DIR ?= $(HOME)/.nvm
NVM_EXEC = bash -c 'set -e; export NVM_DIR="$(NVM_DIR)"; if [ ! -s "$$NVM_DIR/nvm.sh" ]; then echo "Error: NVM is required. Install it from https://github.com/nvm-sh/nvm."; exit 1; fi; . "$$NVM_DIR/nvm.sh"; if [ "$$(nvm version "$(NODE_VERSION)")" = "N/A" ]; then nvm install "$(NODE_VERSION)" --no-progress >/dev/null; fi; nvm use --silent "$(NODE_VERSION)" >/dev/null; "$$@"' --

.PHONY: help
help: ## Display this help message
	@echo "Usage:"
	@echo "  make install          - Install dependencies with the configured Node.js version"
	@echo "  make run              - Start the local docs site"
	@echo "  make build            - Build the docs site"
	@echo "  make pre-commit-check - Run pre-commit checks (WebP images and Chart version)"

.PHONY: install
install: ## Install dependencies with the configured Node.js version.
	@$(NVM_EXEC) pnpm install

.PHONY: run
run: ## Start the local docs site.
	@$(NVM_EXEC) pnpm dev

.PHONY: build
build: ## Build the docs site.
	@$(NVM_EXEC) pnpm build

.PHONY: pre-commit-check
pre-commit-check: ## Run pre-commit checks.
	@if [ ! -f scripts/check-webp-images.sh ]; then \
		echo "❌ Error: scripts/check-webp-images.sh not found"; \
		exit 1; \
	fi
	@if [ ! -f scripts/check-chart-version.sh ]; then \
		echo "❌ Error: scripts/check-chart-version.sh not found"; \
		exit 1; \
	fi
	@chmod +x scripts/check-webp-images.sh scripts/check-chart-version.sh
	@scripts/check-webp-images.sh
	@scripts/check-chart-version.sh

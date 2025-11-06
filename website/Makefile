# Makefile for Crater Website

.PHONY: help
help: ## Display this help message
	@echo "Usage:"
	@echo "  make pre-commit-check - Run pre-commit checks (WebP image format validation)"

.PHONY: pre-commit-check
pre-commit-check: ## Run pre-commit checks (check for non-WebP images in newly added files).
	@if [ ! -f scripts/check-webp-images.sh ]; then \
		echo "❌ Error: scripts/check-webp-images.sh not found"; \
		exit 1; \
	fi
	@chmod +x scripts/check-webp-images.sh
	@scripts/check-webp-images.sh


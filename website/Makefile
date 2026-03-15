# Makefile for Crater Website

.PHONY: help
help: ## Display this help message
	@echo "Usage:"
	@echo "  make pre-commit-check - Run pre-commit checks (WebP images and Chart version)"

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


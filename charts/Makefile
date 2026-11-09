# Makefile for Crater Helm charts

.PHONY: help
help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*?##/ { printf "  %-20s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: pre-commit-check
pre-commit-check: ## Run checks for staged chart changes.
	@bash hack/check-chart-version.sh --staged

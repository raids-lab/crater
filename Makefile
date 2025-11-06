# Makefile for Crater Monorepo

# 颜色定义
RED := \033[31m
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
MAGENTA := \033[35m
CYAN := \033[36m
WHITE := \033[37m
RESET := \033[0m

.PHONY: help
help: ## Display this help message
	@echo "$(CYAN)🌋 Crater Monorepo Commands$(RESET)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; category = ""} \
		/^## / { category = substr($$0, 4); printf "\n$(BLUE)%s$(RESET)\n", category; next } \
		/^[a-zA-Z_-]+:.*?##/ { printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

##@ Git Hooks

## Location to install git hooks to
GIT_HOOKS_DIR ?= $(shell git rev-parse --git-path hooks 2>/dev/null || echo ".git/hooks")
$(GIT_HOOKS_DIR):
	@echo "$(BLUE)Creating git hooks directory...$(RESET)"
	@mkdir -p $(GIT_HOOKS_DIR)

.PHONY: install-hooks
install-hooks: $(GIT_HOOKS_DIR)/pre-commit ## Install git hooks from .githook directory.
$(GIT_HOOKS_DIR)/pre-commit: $(GIT_HOOKS_DIR) .githook/pre-commit
	@echo "$(BLUE)Installing git pre-commit hook...$(RESET)"
	@cp .githook/pre-commit $(GIT_HOOKS_DIR)/pre-commit
	@chmod +x $(GIT_HOOKS_DIR)/pre-commit
	@echo "$(GREEN)✅ Git pre-commit hook installed successfully!$(RESET)"

.PHONY: pre-commit-check
pre-commit-check: install-hooks ## Run the installed pre-commit hook.
	@echo "$(BLUE)Running pre-commit hook...$(RESET)"
	@$(GIT_HOOKS_DIR)/pre-commit

# 默认目标
.DEFAULT_GOAL := help


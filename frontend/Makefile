# Makefile for Crater Frontend Development

# 颜色定义
RED := \033[31m
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
MAGENTA := \033[35m
CYAN := \033[36m
WHITE := \033[37m
RESET := \033[0m

# 设置端口号变量
PORT := $(shell if [ -f .env.development ]; then grep PORT .env.development | cut -d '=' -f2; else echo "5180"; fi)

.PHONY: help
help: ## 💡 Display this help message
	@echo "$(CYAN)🌋 Crater Frontend Development Commands$(RESET)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make $(YELLOW)<target>$(RESET)\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(GREEN)%-15s$(RESET) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: prepare
prepare: ## 🔧 Prepare development environment
	@echo "$(BLUE)Preparing development environment...$(RESET)"
	@if [ ! -f .env.development ]; then \
		echo "$(YELLOW)Creating .env.development file...$(RESET)"; \
		echo 'VITE_API_BASE_URL="http://localhost:8098/"' > .env.development; \
		echo 'VITE_USE_MSW=false  # Enable API mocking' >> .env.development; \
		echo 'PORT=5180           # Dev server port' >> .env.development; \
		echo "$(GREEN)✅ .env.development created successfully!$(RESET)"; \
	else \
		echo "$(GREEN)✅ .env.development already exists$(RESET)"; \
	fi
	@if [ ! -d node_modules ]; then \
		echo "$(YELLOW)Installing dependencies...$(RESET)"; \
		pnpm install; \
		echo "$(GREEN)✅ Dependencies installed successfully!$(RESET)"; \
	else \
		echo "$(GREEN)✅ Dependencies already installed$(RESET)"; \
	fi

.PHONY: run
run: prepare ## 🚀 Start development server
	@echo "$(GREEN)Starting development server on port $(PORT)...$(RESET)"
	pnpm dev --port $(PORT)

.PHONY: build
build: ## 📦 Build for production
	@echo "$(BLUE)Building for production...$(RESET)"
	pnpm build

.PHONY: build-testing
build-testing: ## 📦 Build for testing environment
	@echo "$(BLUE)Building for testing environment...$(RESET)"
	pnpm build-testing

.PHONY: preview
preview: ## 👀 Preview production build
	@echo "$(BLUE)Starting preview server...$(RESET)"
	pnpm preview

.PHONY: format
format: ## ✨ Format code with Prettier
	@echo "$(MAGENTA)Formatting code with Prettier...$(RESET)"
	pnpm prettier --write .

.PHONY: lint
lint: ## 🔍 Run ESLint and TypeScript checks
	@echo "$(YELLOW)Running TypeScript checks...$(RESET)"
	pnpm tsc --noEmit
	@echo "$(YELLOW)Running ESLint...$(RESET)"
	pnpm eslint .

.PHONY: lint-fix
lint-fix: ## 🔧 Fix ESLint issues automatically
	@echo "$(YELLOW)Running TypeScript checks...$(RESET)"
	pnpm tsc --noEmit
	@echo "$(YELLOW)Fixing ESLint issues...$(RESET)"
	pnpm eslint . --fix

.PHONY: type-check
type-check: ## 📝 Run TypeScript type checking
	@echo "$(YELLOW)Running TypeScript type checking...$(RESET)"
	pnpm tsc --noEmit

.PHONY: test
test: ## 🧪 Run tests
	@echo "$(CYAN)Running tests...$(RESET)"
	@echo "$(YELLOW)Note: Add test command to package.json if needed$(RESET)"

.PHONY: clean
clean: ## 🧹 Clean build artifacts and dependencies
	@echo "$(RED)Cleaning build artifacts and dependencies...$(RESET)"
	rm -rf dist/
	rm -rf node_modules/
	rm -rf .turbo/
	@echo "$(GREEN)✅ Cleanup completed$(RESET)"

.PHONY: deps-update
deps-update: ## 📦 Update dependencies
	@echo "$(BLUE)Checking for outdated dependencies...$(RESET)"
	pnpm outdated
	@echo "$(YELLOW)Run 'pnpm update' to update minor versions$(RESET)"
	@echo "$(YELLOW)Run 'pnpm update --latest' to update major versions$(RESET)"

.PHONY: generate
generate: ## 🛠️ Generate error codes from backend
	@echo "$(BLUE)Generating code...$(RESET)"
	python3 ./src/services/generator.py ../web-backend/internal/resputil/code.go ./src/services/error_code.ts

.PHONY: dev
dev: run ## 🚀 Alias for run command

# 默认目标
.DEFAULT_GOAL := help

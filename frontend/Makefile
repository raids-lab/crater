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

PYTHON ?= $(shell if command -v python3 >/dev/null 2>&1 && python3 -c "import sys" >/dev/null 2>&1; then echo python3; elif command -v python >/dev/null 2>&1; then echo python; fi)

# 设置端口号变量
PORT := $(shell if [ -f .env.development ]; then grep PORT .env.development | cut -d '=' -f2; else echo "5180"; fi)
.PHONY: help
help: ## 💡 Display this help message
	@echo "$(CYAN)🌋 Crater Frontend Development Commands$(RESET)"
	@echo ""
	@echo "$(YELLOW)📋 Prerequisites:$(RESET)"
	@echo "  • Backend API server should be running on http://localhost:8088/"
	@echo "  • Storage server should be running on http://localhost:7320/"
	@echo "  • Run 'make backend-help' for backend startup instructions"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; category = ""} \
		/^## / { category = substr($$0, 4); printf "\n$(BLUE)%s$(RESET)\n", category; next } \
		/^[a-zA-Z_-]+:.*?##/ { printf "  $(GREEN)%-15s$(RESET) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

## 🛠️  Development Environment
.PHONY: prepare
prepare: ## 🔧 Prepare development environment with updated configs
	@echo "$(BLUE)Preparing development environment...$(RESET)"
	@if [ ! -f .env.development ]; then \
		echo "$(YELLOW)Creating .env.development file...$(RESET)"; \
		echo 'VITE_SERVER_PROXY_BACKEND="http://localhost:8088/"' > .env.development; \
		echo 'VITE_SERVER_PROXY_STORAGE="http://localhost:7320/"' >> .env.development; \
		echo 'VITE_USE_MSW=false  # Enable API mocking' >> .env.development; \
		echo 'VITE_TANSTACK_QUERY_DEVTOOLS=true  # Enable React Query DevTools' >> .env.development; \
		echo 'VITE_TANSTACK_ROUTER_DEVTOOLS=true  # Enable React Router DevTools' >> .env.development; \
		echo 'PORT=5180           # Dev server port' >> .env.development; \
		echo '' >> .env.development; \
		echo '# Version information (for development testing)' >> .env.development; \
		echo 'VITE_APP_VERSION="dev-local"' >> .env.development; \
		echo 'VITE_APP_COMMIT_SHA="1234567890abcdef"' >> .env.development; \
		echo 'VITE_APP_BUILD_TYPE="development"' >> .env.development; \
		echo "VITE_APP_BUILD_TIME=\"$$(date -u +%Y-%m-%dT%H:%M:%SZ)\"" >> .env.development; \
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

.PHONY: run dev
run: prepare ## 🚀 Start development server
	@echo "$(GREEN)Starting development server on port $(PORT)...$(RESET)"
	pnpm dev --port $(PORT)

dev: run ## 🚀 Alias for run command

.PHONY: deps-update
deps-update: ## 📦 Update dependencies
	@echo "$(BLUE)Checking for outdated dependencies...$(RESET)"
	pnpm outdated
	@echo "$(YELLOW)Run 'pnpm update' to update minor versions$(RESET)"
	@echo "$(YELLOW)Run 'pnpm update --latest' to update major versions$(RESET)"

.PHONY: clean
clean: ## 🧹 Clean build artifacts and dependencies
	@echo "$(RED)Cleaning build artifacts and dependencies...$(RESET)"
	rm -rf dist/
	rm -rf node_modules/
	rm -rf .turbo/
	@echo "$(GREEN)✅ Cleanup completed$(RESET)"

## 🏗️  Building & Deployment
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

## 🧪 Testing & Quality Assurance
.PHONY: lint
lint: ## 🔍 Run ESLint and TypeScript checks
	@echo "$(YELLOW)Running TypeScript checks...$(RESET)"
	pnpm tsc --noEmit
	@echo "$(YELLOW)Running ESLint...$(RESET)"
	pnpm eslint .

.PHONY: pre-commit-check
pre-commit-check: ## Run pre-commit checks (lint and lint-staged).
	@echo "$(BLUE)Checking translation file format...$(RESET)"
	@if [ -z "$(PYTHON)" ]; then \
		echo "$(RED)❌ Python interpreter not found (python3/python).$(RESET)"; \
		exit 1; \
	fi
	@if ! $(PYTHON) hack/format_translation.py --check; then \
		echo ""; \
		echo "$(RED)❌ Translation files are not properly formatted!$(RESET)"; \
		echo ""; \
		echo "$(YELLOW)💡 Please run '$(GREEN)make format-translation$(RESET)$(YELLOW)' to format them.$(RESET)"; \
		exit 1; \
	fi
	@echo "$(BLUE)Running lint checks...$(RESET)"
	@if ! pnpm run lint; then \
		echo "$(RED)❌ Frontend lint failed!$(RESET)"; \
		echo ""; \
		echo "$(YELLOW)💡 If you see 'command not found' or 'Cannot find module' errors:$(RESET)"; \
		echo "   Please run 'pnpm install' to install dependencies first."; \
		exit 1; \
	fi
	@echo "$(BLUE)Running lint-staged...$(RESET)"
	@if ! pnpm exec lint-staged; then \
		echo "$(RED)❌ Frontend lint-staged failed!$(RESET)"; \
		exit 1; \
	fi

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

.PHONY: format
format: ## ✨ Format code with Prettier
	@echo "$(MAGENTA)Formatting code with Prettier...$(RESET)"
	pnpm prettier --write .

.PHONY: format-translation
format-translation: ## 🌐 Format translation files
	@echo "$(BLUE)Formatting translation files...$(RESET)"
	@if [ -z "$(PYTHON)" ]; then \
		echo "$(RED)❌ Python interpreter not found (python3/python).$(RESET)"; \
		exit 1; \
	fi
	@if $(PYTHON) hack/format_translation.py; then \
		echo "$(GREEN)✅ Translation files formatted$(RESET)"; \
	else \
		echo "$(RED)❌ Translation file formatting failed!$(RESET)"; \
		exit 1; \
	fi

.PHONY: test
test: ## 🧪 Run tests
	@echo "$(CYAN)Running tests...$(RESET)"
	@echo "$(YELLOW)Note: Add test command to package.json if needed$(RESET)"

## 🛡️  Code Generation & Backend Integration
.PHONY: generate-error-code
generate-error-code: ## 🛠️ Generate error codes from backend
	@echo "$(BLUE)Generating error codes from backend...$(RESET)"
	@if [ -z "$(PYTHON)" ]; then \
		echo "$(RED)❌ Python interpreter not found (python3/python).$(RESET)"; \
		exit 1; \
	fi
	$(PYTHON) ./src/services/generator.py ../backend/internal/resputil/code.go ./src/services/error_code.ts

.PHONY: backend-help
backend-help: ## ❓ Show backend server startup instructions
	@echo "$(CYAN)🚀 Backend Server Startup Instructions$(RESET)"
	@echo ""
	@echo "$(YELLOW)1. API Backend Server (Port 8088):$(RESET)"
	@echo "   • Navigate to your backend directory"
	@echo "   • Run: $(GREEN)make run$(RESET) or $(GREEN)go run main.go$(RESET)"
	@echo "   • Ensure it's accessible at: $(BLUE)http://localhost:8088/$(RESET)"
	@echo ""
	@echo "$(YELLOW)2. Storage Server (Port 7320):$(RESET)"
	@echo "   • Navigate to your storage service directory"
	@echo "   • Run the appropriate startup command for your storage service"
	@echo "   • Ensure it's accessible at: $(BLUE)http://localhost:7320/$(RESET)"
	@echo ""
	@echo "$(YELLOW)3. Verify Services:$(RESET)"
	@echo "   • API Backend: $(GREEN)curl http://localhost:8088/health$(RESET)"
	@echo "   • Storage Server: $(GREEN)curl http://localhost:7320/health$(RESET)"
	@echo ""
	@echo "$(CYAN)💡 Tips:$(RESET)"
	@echo "   • Check your backend documentation for specific startup commands"
	@echo "   • Ensure all required environment variables are set"
	@echo "   • Use Docker Compose if available for easier service management"

# 默认目标
.DEFAULT_GOAL := help

# Makefile for Crater CLI

# 颜色定义
RED := \033[31m
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
MAGENTA := \033[35m
CYAN := \033[36m
RESET := \033[0m

# 变量定义
BINARY_NAME := crater

.PHONY: help
help: ## 显示帮助信息
	@echo "$(CYAN)🌋 Crater CLI 编译管理$(RESET)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; category = ""} \
		/^## / { category = substr($$0, 4); printf "\n$(BLUE)%s$(RESET)\n", category; next } \
		/^[a-zA-Z_-]+:.*?##/ { printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

##@ 编译与构建

.PHONY: build
build: tidy ## 编译生成 crater 可执行文件
	@echo "$(BLUE)🏗️  正在构建 $(BINARY_NAME)...$(RESET)"
	@go build -o $(BINARY_NAME) .
	@echo "$(GREEN)✅ 构建完成: ./$(BINARY_NAME)$(RESET)"

.PHONY: clean
clean: ## 清理构建产物
	@echo "$(YELLOW)🧹 正在清理构建产物...$(RESET)"
	@rm -f $(BINARY_NAME)
	@echo "$(GREEN)✅ 清理完成$(RESET)"

##@ 依赖管理

.PHONY: tidy
tidy: ## 整理 Go 依赖
	@echo "$(BLUE)📦 正在整理依赖...$(RESET)"
	@go mod tidy
	@echo "$(GREEN)✅ 依赖整理完成$(RESET)"

# 默认目标
.DEFAULT_GOAL := help

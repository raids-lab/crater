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

##@ 快照测试（txtar，见 testdata/snapshots/ 与 test/snapshots）

.PHONY: snapshot-check
snapshot-check: ## 校验 golden：go test ./test/snapshots/...（不修改文件）
	@echo "$(BLUE)📸 快照校验...$(RESET)"
	@go test ./test/snapshots/... -count=1
	@echo "$(GREEN)✅ 快照校验通过$(RESET)"

.PHONY: snapshot-update
snapshot-update: ## 重写 golden：UPDATE_SNAPSHOTS=1 go test ./test/snapshots/...（提交前请 git diff）
	@echo "$(YELLOW)📸 更新快照 golden...$(RESET)"
	@UPDATE_SNAPSHOTS=1 go test ./test/snapshots/... -count=1
	@echo "$(GREEN)✅ 已更新 testdata/snapshots/$(RESET)"

##@ 测试

# 单元测试：显式列出当前包含单测的包（避免依赖 grep / shell 过滤）。
# 新增单测时，将对应包路径加入此列表即可。
UNIT_TEST_PKGS := \
	./cmd \
	./internal/completion \
	./internal/output \
	./internal/testenv

.PHONY: unit-test
unit-test: ## 运行 cli 的所有单元测试（排除 ./test/snapshots）
	@echo "$(BLUE)🧪 运行单元测试（排除 ./test/snapshots）...$(RESET)"
	@go test $(UNIT_TEST_PKGS) -count=1
	@echo "$(GREEN)✅ 单元测试通过$(RESET)"

.PHONY: test
test: unit-test snapshot-check ## 依次运行单元测试与快照校验
	@echo "$(GREEN)✅ 全部测试通过$(RESET)"

# 默认目标
.DEFAULT_GOAL := help

# Makefile

# 设置端口号变量
PORT := $(shell grep PORT .env.development | cut -d '=' -f2)

.PHONY: run

# run 目标，使用 pnpm 启动开发服务器
run:
	@echo "Starting development server on port $(PORT)..."
	pnpm dev --port $(PORT)

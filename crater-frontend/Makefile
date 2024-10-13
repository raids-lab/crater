# Makefile

# 设置端口号变量
PORT := $(shell grep PORT .env.development | cut -d '=' -f2)

.PHONY: run

# run 目标，使用 pnpm 启动开发服务器
run:
	@echo "Starting development server on port $(PORT)..."
	pnpm dev --port $(PORT)

.PHONY: generate

# python3 ./src/services/generator.py ../web-backend/internal/resputil/code.go ./src/services/error_code.ts
generate:
	@echo "Generating code..."
	python3 ./src/services/generator.py ../web-backend/internal/resputil/code.go ./src/services/error_code.ts

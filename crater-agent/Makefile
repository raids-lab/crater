.PHONY: install dev build lint format test pre-commit-check

install:
	pnpm install

dev:
	pnpm dev

build:
	pnpm build

lint:
	pnpm lint
	pnpm format:check

test:
	node --import tsx --test src/tools/registry.test.ts src/routes/chat.test.ts

pre-commit-check: lint test build

format:
	pnpm format

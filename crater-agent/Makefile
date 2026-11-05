.PHONY: install dev build lint format pre-commit-check

install:
	pnpm install

dev:
	pnpm dev

build:
	pnpm build

lint:
	pnpm lint
	pnpm format:check

pre-commit-check: lint build

format:
	pnpm format

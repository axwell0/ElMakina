# File: Makefile
# Purpose: Collects the local development, testing, linting, and generation commands.
.PHONY: install docker server web dev build build-web race vet fmt format format-check lint lint-go lint-web install-hooks test test-go ws-contract-generate

install:
	go mod tidy
	bun install
	node scripts/generate-ws-contract.ts

server:
	go run ./server/cmd/server/

web:
	bun run dev

dev:
	@echo "Starting server and web dev..."
	@make server & \
	make web

build-web:
	bun run build

build:
	go build ./...

docker:
	docker compose -f docker-compose.yml up

test: test-go

test-go:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

# Linting and formatting
lint: lint-go lint-web

lint-go:
	./.codex/bin/golangci-lint run ./...

lint-go-fix:
	./.codex/bin/golangci-lint run --fix ./...

lint-web:
	cd web && bunx eslint .

format:
	bun run format
	gofmt -w .

format-check:
	bun run format:check
	gofmt -l .

# Install git hooks
install-hooks:
	bun run prepare

ws-contract-generate:
	node scripts/generate-ws-contract.ts

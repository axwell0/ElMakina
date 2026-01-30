.PHONY: install docker server dev-web build-web docs-generate gen-schema gen-types check-types gen-all server dev-web dev

install:
	go mod tidy
	bun install

server:
	go run ./backend/cmd/server/

web:
	bun --cwd web dev

dev:
	@echo "Starting server and web dev..."
	@make server & \
	make web

build-web:
	bun run --cwd web build

docker:
	docker compose -f docker-compose.yml up

docs-generate:
	bash scripts/generate-go-docs.sh
	bash scripts/generate-ts-docs.sh

gen-schema:
	go run ./backend/scripts/generate-schema/
	bun run generate:ws-contract

gen-types:
	cd backend && go run scripts/generate-types/main.go

check-types:
	cd backend && go run scripts/generate-types/main.go
	git diff --exit-code web/src/types/generated.ts

gen-all: gen-schema gen-types docs-generate

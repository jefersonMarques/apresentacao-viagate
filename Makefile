APP=viagate-commercial

.PHONY: dev generate build test migrate-up migrate-down

generate:
	go run github.com/a-h/templ/cmd/templ@v0.3.943 generate

dev: generate
	go run ./cmd/server

build: generate
	go build -o bin/$(APP) ./cmd/server

test:
	go test ./...

migrate-up:
	@echo "Apply SQL files from migrations/ in order using your PostgreSQL migration runner."

migrate-down:
	@echo "Down migrations are intentionally not automatic for immutable commercial records."

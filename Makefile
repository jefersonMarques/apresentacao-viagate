APP=viagate-commercial

.PHONY: deps dev generate build test check migrate-up migrate-down

deps:
	go mod download

generate:
	go run github.com/a-h/templ/cmd/templ@v0.3.943 generate

dev: generate
	go run ./cmd/server

build: generate
	go build -o bin/$(APP) ./cmd/server

test: generate
	go test ./...

check: generate
	go vet ./...
	go test ./...
	go build ./cmd/server
	go build ./cmd/migrate

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	@echo "Down migrations are intentionally disabled for immutable commercial/legal records. Restore from backup or add a reviewed forward migration."

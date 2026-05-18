.PHONY: help run build migrate-up migrate-down migrate-create install tidy

# Default DB URL from .env (loaded via godotenv in code; here we read manually)
DB_URL := $(shell grep -E '^DATABASE_URL=' .env 2>/dev/null | cut -d= -f2-)

# Resolve `migrate` binary: PATH first, then $GOPATH/bin fallback
MIGRATE := $(shell command -v migrate 2>/dev/null || echo "$(shell go env GOPATH)/bin/migrate")

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

install: ## Install dependencies (Go modules + tools)
	go mod download
	@test -x "$(MIGRATE)" || { echo "→ Installing golang-migrate…"; go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest; }
	@echo "✓ Tools installed."
	@echo "  migrate: $(MIGRATE)"
	@echo ""
	@echo "→ Pastikan \$$(go env GOPATH)/bin ada di PATH (lihat README → Troubleshooting)."

tidy: ## Tidy go.mod
	go mod tidy

run: ## Run dev server with hot reload (requires air; falls back to go run)
	@command -v air >/dev/null 2>&1 && air || go run ./cmd/server

build: ## Build binary
	go build -o bin/server ./cmd/server

migrate-up: ## Apply pending migrations
	@test -n "$(DB_URL)" || (echo "DATABASE_URL not set in .env"; exit 1)
	@test -x "$(MIGRATE)" || (echo "migrate binary not found. Run 'make install' first."; exit 1)
	"$(MIGRATE)" -path internal/db/migrations -database "$(DB_URL)" up

migrate-down: ## Rollback last migration
	@test -n "$(DB_URL)" || (echo "DATABASE_URL not set in .env"; exit 1)
	@test -x "$(MIGRATE)" || (echo "migrate binary not found. Run 'make install' first."; exit 1)
	"$(MIGRATE)" -path internal/db/migrations -database "$(DB_URL)" down 1

migrate-create: ## Create new migration: make migrate-create name=add_users
	@test -n "$(name)" || (echo "Usage: make migrate-create name=<migration_name>"; exit 1)
	@test -x "$(MIGRATE)" || (echo "migrate binary not found. Run 'make install' first."; exit 1)
	"$(MIGRATE)" create -ext sql -dir internal/db/migrations -seq $(name)

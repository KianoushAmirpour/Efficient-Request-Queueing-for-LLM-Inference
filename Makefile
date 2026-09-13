include .env
export

# Go configuration
GO := go
MAIN := ./cmd

# Docker Compose configuration
DOCKER_COMPOSE := docker compose
COMPOSE_FILE := infra/docker-compose.yml

# Database configuration
DB_DSN := postgres://$(MIGRATION_USER):$(MIGRATION_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=$(POSTGRES_SSLMODE)
MIGRATIONS_DIR := ./migrations

# Phony targets
.PHONY: run up down migrate_up migrate_down lint

run: ## Run the main Go application
	$(GO) run $(MAIN)

up: ## Start the development environment via Docker Compose (detached)
	$(DOCKER_COMPOSE) --env-file .env -f $(COMPOSE_FILE) up -d

down: ## Tear down containers and networks cleanly
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) down

migrate_up: ## Run database migrations up
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_DSN)" up

lint:
	golangci-lint run ./...